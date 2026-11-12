package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// formatDueDateValue — человекочитаемое значение срока для снапшота журнала
// активности (RFC3339 для даты; «none» для отсутствия). Граница пакета models не
// позволяет использовать unexported formatFieldValue — дублируем здесь.
func formatDueDateValue(v *time.Time) string {
	if v == nil {
		return "none"
	}
	return v.Format(time.RFC3339)
}

// CreateDueDateRequest — исполнитель запрашивает перенос срока заявки.
// Запрос имеет смысл только при установленном сроке: перенос без срока — это
// новое самостоятельное требование, которое исполнитель не может сформулировать.
// Активная заявка должна быть назначена исполнителю; один pending-запрос на
// тикет существует ограниченно (unique partial index + ErrDueDateRequestAlreadyPending).
func (s *TicketService) CreateDueDateRequest(ctx context.Context, dto *models.CreateDueDateRequestDTO) (*models.DueDateRequest, error) {
	if dto == nil || dto.TicketID == uuid.Nil || dto.ProposedDueDate.IsZero() || strings.TrimSpace(dto.Reason) == "" {
		return nil, models.ErrInvalidInput
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID, Actor: dto.Actor, RealmID: dto.RealmID})
	if err != nil {
		return nil, err
	}

	// Запрос делает только исполнитель (assignee) — не создатель/заказчик и не
	// «просто участник группы»: работа именно его, и решать он умеет лучше всех.
	if ticket.Assignee == nil || ticket.Assignee.ID != dto.Actor.ID {
		return nil, models.ErrPermissionDenied
	}
	if isTicketInactive(ticket.Status) {
		return nil, models.ErrTicketFrozen
	}
	if ticket.DueDate == nil {
		return nil, models.ErrInvalidInput
	}
	if !dto.ProposedDueDate.After(*ticket.DueDate) {
		return nil, models.ErrInvalidInput
	}

	if existing, err := s.dueRequests.GetPendingByTicket(ctx, dto.TicketID); err == nil && existing != nil {
		return nil, models.ErrDueDateRequestAlreadyPending
	}

	newRequest := &models.DueDateRequest{
		ID:              uuid.New(),
		TicketID:        dto.TicketID,
		RequesterID:     dto.Actor.ID,
		Requester:       ticket.Assignee,
		CurrentDueDate:  ticket.DueDate,
		ProposedDueDate: dto.ProposedDueDate,
		Reason:          dto.Reason,
		Status:          "pending",
		CreatedAt:       time.Now(),
	}
	if err := s.dueRequests.Create(ctx, nil, newRequest); err != nil {
		return nil, fmt.Errorf("failed to create due date request: %w", err)
	}

	// Журнал: запрос на перенос срока (тег due_date_requested, старое/новое — сроки).
	log := &models.ActivityLogDTO{
		Action:        "updated",
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		EntityType:    string(access.ResourceTicket),
		EntityID:      dto.TicketID,
		Entity:        ticket.Title,
	}
	if err := log.SetOldValues(map[string]string{"due_date_requested": formatDueDateValue(ticket.DueDate)}); err != nil {
		return nil, fmt.Errorf("set old values: %w", err)
	}
	if err := log.SetNewValues(map[string]string{"due_date_requested": formatDueDateValue(&dto.ProposedDueDate)}); err != nil {
		return nil, fmt.Errorf("set new values: %w", err)
	}
	if err := s.logs.Create(ctx, nil, []*models.ActivityLogDTO{log}); err != nil {
		return nil, fmt.Errorf("store log: %w", err)
	}

	// Уведомление менеджеру группы о новом запросе.
	if ticket.Manager != nil {
		if err := s.notifications.NotifyDueDateRequested(ctx, ticket, newRequest); err != nil {
			// Уведомление — best-effort: запрос уже сохранён и зафиксирован в журнале.
			s.notifyBestEffort("due_date_requested", dto.TicketID, ticket.Title, err)
		}
	}

	return newRequest, nil
}

// GetDueDateRequest возвращает активный (pending) запрос тикета на перенос срока.
func (s *TicketService) GetDueDateRequest(ctx context.Context, ticketID uuid.UUID) (*models.DueDateRequest, error) {
	return s.dueRequests.GetPendingByTicket(ctx, ticketID)
}

// DecideDueDateRequest — решение менеджера (менеджер группы или начальник области)
// по запросу исполнителя на перенос срока. Утверждение закрывает запрос и
// применяет новый срок к заявке (через обычный Update: менеджер уже имеет право
// править срок); отклонение — только статус запроса. Решение пишет журнал и
// уведомляет исполнителя.
func (s *TicketService) DecideDueDateRequest(ctx context.Context, dto *models.DecideDueDateRequestDTO) (*models.DueDateRequest, error) {
	if dto == nil || dto.RequestID == uuid.Nil || dto.DecisionComment == "" && !dto.Approved {
		return nil, models.ErrInvalidInput
	}

	request, err := s.dueRequests.GetByID(ctx, dto.RequestID)
	if err != nil {
		return nil, err
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: request.TicketID, Actor: dto.Actor, RealmID: dto.RealmID})
	if err != nil {
		return nil, err
	}

	// Решение принимает менеджер группы тикета или начальник области (realm supervisor).
	decided := false
	supervisor, supErr := s.access.IsRealmSupervisor(ctx, dto.Actor.ID, ticketRealm(ticket))
	if supErr != nil {
		return nil, fmt.Errorf("failed to check supervisor access: %w", supErr)
	}
	decided = supervisor
	if !decided {
		if ok, mgrErr := s.isManagerOfGroup(ctx, ticket.Group, dto.Actor.ID); mgrErr != nil {
			return nil, fmt.Errorf("failed to check manager access: %w", mgrErr)
		} else {
			decided = ok
		}
	}
	if !decided {
		return nil, models.ErrPermissionDenied
	}

	newStatus := "rejected"
	if dto.Approved {
		newStatus = "approved"
	}
	decidedAt := time.Now()
	decidedRequest, err := s.dueRequests.Decide(ctx, nil, request.ID, newStatus, dto.Actor.ID, decidedAt, dto.DecisionComment)
	if err != nil {
		return nil, err
	}

	if dto.Approved {
		// Утверждение — мгновенная смена срока: через общий Update (правки срока уже
		// разрешены у принимающего решение), он проставит due_date и зафиксирует
		// журнал (due_date_changed).
		newDueDate := decidedRequest.ProposedDueDate
		if err := s.Update(ctx, &models.TicketDTO{
			ID:      &decidedRequest.TicketID,
			RealmID: ticket.RealmID,
			Actor:   dto.Actor,
			DueDate: &newDueDate,
			Provided: map[string]bool{
				"dueDate": true,
			},
		}); err != nil {
			return nil, err
		}
	}

	if !dto.Approved {
		// Отклонение — не меняет срок, но фиксирует в истории факт решения:
		// старое значение — запрошенный перенос, новое — фактический (текущий) срок.
		log := &models.ActivityLogDTO{
			Action:        "updated",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:   string(access.ResourceTicket),
			EntityID:      request.TicketID,
			Entity:        ticket.Title,
		}
		if err := log.SetOldValues(map[string]string{"due_date_rejected": formatDueDateValue(&request.ProposedDueDate)}); err != nil {
			return nil, fmt.Errorf("set old values: %w", err)
		}
		if err := log.SetNewValues(map[string]string{"due_date_rejected": formatDueDateValue(request.CurrentDueDate)}); err != nil {
			return nil, fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, nil, []*models.ActivityLogDTO{log}); err != nil {
			return nil, fmt.Errorf("store log: %w", err)
		}
	}

	// Уведомление исполнителя о решении.
	if ticket.Assignee != nil && ticket.Assignee.ID != dto.Actor.ID {
		if err := s.notifications.NotifyDueDateDecided(ctx, ticket, decidedRequest); err != nil {
			s.notifyBestEffort("due_date_decided", decidedRequest.TicketID, ticket.Title, err)
		}
	}

	return decidedRequest, nil
}
