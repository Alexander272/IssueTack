package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
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
	if dto.Actor == nil {
		return nil, models.ErrPermissionDenied
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
	// Новый срок — всегда в будущем: у просроченной заявки текущий срок уже в
	// прошлом, поэтому одной проверки «позже текущего» недостаточно.
	if !dto.ProposedDueDate.After(time.Now()) {
		return nil, models.ErrInvalidInput
	}

	// Ошибку чтения нельзя проглатывать: иначе при сбое БД мы пойдём на INSERT и
	// вернём пользователю сырую ошибку вместо причины.
	existing, err := s.dueRequests.GetPendingByTicket(ctx, dto.TicketID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending due date request: %w", err)
	}
	if existing != nil {
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
		Status:          models.DueDateRequestStatusPending,
		CreatedAt:       time.Now(),
	}
	if err := s.dueRequests.Create(ctx, nil, newRequest); err != nil {
		return nil, fmt.Errorf("failed to create due date request: %w", err)
	}

	// Журнал: запрос на перенос срока — один тег due_date_requested.
	// newValue содержит новый срок и причину (чтобы в истории это было одним изменением,
	// а не «изменил поля» с двумя тегами).
	log := &models.ActivityLogDTO{
		Action:        "updated",
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		EntityType:    string(access.ResourceTicket),
		EntityID:      dto.TicketID,
		Entity:        ticket.Title,
	}
	if err := log.SetOldValues(map[string]string{
		string(models.ActionDueDateRequested): formatDueDateValue(ticket.DueDate),
	}); err != nil {
		return nil, fmt.Errorf("set old values: %w", err)
	}
	// newValue: новый срок + причина (разделены \n для рендера в две строки).
	newVal := formatDueDateValue(&dto.ProposedDueDate)
	if strings.TrimSpace(dto.Reason) != "" {
		newVal += "\nПричина: " + dto.Reason
	}
	if err := log.SetNewValues(map[string]string{
		string(models.ActionDueDateRequested): newVal,
	}); err != nil {
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
// Доступ проверяется по атрибутам заявки (как в GetByID): одного realm-wide
// ticket:read мало — запрос содержит срок и причину, их видит только тот, кому
// доступна сама заявка.
func (s *TicketService) GetDueDateRequest(ctx context.Context, dto *models.GetDueDateRequestDTO) (*models.DueDateRequest, error) {
	if dto == nil || dto.TicketID == uuid.Nil {
		return nil, models.ErrInvalidInput
	}
	if dto.Actor == nil {
		return nil, models.ErrPermissionDenied
	}

	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: dto.TicketID,
		UserID:   dto.Actor.ID,
		Action:   string(access.Read),
	}); err != nil {
		return nil, err
	}

	return s.dueRequests.GetPendingByTicket(ctx, dto.TicketID)
}

// DecideDueDateRequest — решение менеджера (менеджер группы или начальник области)
// по запросу исполнителя на перенос срока. Утверждение закрывает запрос и
// применяет новый срок к заявке (через updateTicket в общей транзакции: менеджер
// уже имеет право править срок); отклонение — только статус запроса. Решение пишет
// журнал и уведомляет исполнителя. Решение и его последствия (закрытие запроса +
// смена срока / журнал отклонения) выполняются в одной транзакции.
func (s *TicketService) DecideDueDateRequest(ctx context.Context, dto *models.DecideDueDateRequestDTO) (*models.DueDateRequest, error) {
	if dto == nil || dto.RequestID == uuid.Nil || (dto.DecisionComment == "" && !dto.Approved) {
		return nil, models.ErrInvalidInput
	}
	if dto.Actor == nil {
		return nil, models.ErrPermissionDenied
	}

	request, err := s.dueRequests.GetByID(ctx, dto.RequestID)
	if err != nil {
		return nil, err
	}
	// :id в пути — не декорация: решение принимается только по запросу той заявки,
	// которая открыта у пользователя.
	if dto.TicketID != uuid.Nil && request.TicketID != dto.TicketID {
		return nil, models.ErrNotFound
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

	// Утверждение меняет срок заявки, а на «замороженной» заявке (resolved/closed/
	// cancelled) срок неизменяем — отказываем сразу, до транзакции. Отклонить запрос
	// на такой заявке можно: это лишь закрывает его.
	if dto.Approved && isTicketInactive(ticket.Status) {
		return nil, models.ErrTicketFrozen
	}

	newStatus := models.DueDateRequestStatusRejected
	if dto.Approved {
		newStatus = models.DueDateRequestStatusApproved
	}
	decidedAt := time.Now()
	request.Status = newStatus
	request.DecidedBy = &dto.Actor.ID
	request.DecidedAt = &decidedAt
	request.DecisionComment = dto.DecisionComment

	// Решение и его последствия — одна транзакция: утверждение закрывает запрос и
	// меняет срок заявки, отклонение закрывает запрос и пишет журнал. Раздельные
	// операции оставляли запрос закрытым, если смена срока падала, — запрос «решён»,
	// а срок прежний, и повторить решение уже нельзя.
	var (
		decidedRequest *models.DueDateRequest
		changes        []*models.FieldChange
	)
	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		decided, err := s.dueRequests.Decide(ctx, newTx, request)
		if err != nil {
			return err
		}
		decidedRequest = decided

		if dto.Approved {
			// Утверждение — смена срока через общее тело Update: права на срок у
			// принимающего решение уже есть, а журнал/проверки остаются в одном месте.
			// В истории это не «изменил срок», а «утвердил перенос» — подменяем тег
			// только в снапшоте (уведомления получают исходные изменения).
			newDueDate := decided.ProposedDueDate
			changes, err = s.updateTicket(ctx, newTx, &models.TicketDTO{
				ID:      &decided.TicketID,
				RealmID: ticket.RealmID,
				Actor:   dto.Actor,
				DueDate: &newDueDate,
				Provided: map[string]bool{
					"dueDate": true,
				},
			}, &ticketUpdateOptions{
				logTagAlias: map[models.ActivityType]models.ActivityType{
					models.ActionDueDateChanged: models.ActionDueDateApproved,
				},
			})
			return err
		}

		// Отклонение — один тег due_date_rejected: в истории это отказ, а не
		// «изменил поля». newValue — текущий срок и причина отклонения.
		log := &models.ActivityLogDTO{
			Action:        "updated",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      request.TicketID,
			Entity:        ticket.Title,
		}
		oldVal := formatDueDateValue(&request.ProposedDueDate)
		newVal := formatDueDateValue(request.CurrentDueDate)
		if strings.TrimSpace(dto.DecisionComment) != "" {
			newVal += "\nПричина: " + dto.DecisionComment
		}
		if err := log.SetOldValues(map[string]string{
			string(models.ActionDueDateRejected): oldVal,
		}); err != nil {
			return fmt.Errorf("set old values: %w", err)
		}
		if err := log.SetNewValues(map[string]string{
			string(models.ActionDueDateRejected): newVal,
		}); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Уведомления — после коммита: подписчики не должны узнать об откатившемся решении.
	if dto.Approved {
		s.notifyTicketUpdated(ctx, &models.TicketDTO{ID: &decidedRequest.TicketID, Actor: dto.Actor}, changes)
	}
	if ticket.Assignee != nil && ticket.Assignee.ID != dto.Actor.ID {
		if err := s.notifications.NotifyDueDateDecided(ctx, ticket, decidedRequest); err != nil {
			s.notifyBestEffort("due_date_decided", decidedRequest.TicketID, ticket.Title, err)
		}
	}

	return decidedRequest, nil
}
