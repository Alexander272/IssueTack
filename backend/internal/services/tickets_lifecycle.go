package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// Take — «взять заявку в работу»: пользователь назначает себя исполнителем
// (assignee) и переводит заявку в статус in_progress. Разрешено, если заявка
// активна (не resolved/closed/cancelled), пользователь имеет read-доступ и заявка
// «забираема» — не имеет исполнителя вовсе, либо исполнитель — другой пользователь,
// а заявка ещё в статусе open. Если исполнитель уже является текущим пользователем —
// отказ (нет смысла брать собственную заявку).
func (s *TicketService) Take(ctx context.Context, dto *models.TakeTicketDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		return models.ErrTicketFrozen
	}

	if err := s.access.CheckAccessOnTicket(ctx, ticket, dto.Actor.ID, string(access.Read)); err != nil {
		return err
	}

	if ticket.Assignee != nil {
		if ticket.Assignee.ID == dto.Actor.ID {
			return models.ErrPermissionDenied
		}
		if ticket.Status != models.StatusOpen {
			return models.ErrPermissionDenied
		}
	}

	var realmID *uuid.UUID
	if dto.RealmID != "" {
		if rid, parseErr := uuid.Parse(dto.RealmID); parseErr == nil {
			realmID = &rid
		}
	}
	dtoUpdate := &models.TicketDTO{
		ID:         &dto.ID,
		Actor:      dto.Actor,
		RealmID:    realmID,
		Status:     models.StatusInProgress,
		AssigneeID: &dto.Actor.ID,
	}
	dtoUpdate.MarkProvided("status")
	dtoUpdate.MarkProvided("assigneeId")

	var changes []*models.FieldChange
	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, loadErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if loadErr != nil {
			return loadErr
		}

		changes = dtoUpdate.GetChanges(oldTicket)

		if err := s.repo.Update(ctx, newTx, dtoUpdate); err != nil {
			return fmt.Errorf("failed to update ticket. error: %w", err)
		}

		if len(changes) > 0 {
			oldMap := make(map[string]string, len(changes))
			newMap := make(map[string]string, len(changes))
			for _, c := range changes {
				oldMap[string(c.Tag)] = c.OldVal
				newMap[string(c.Tag)] = c.NewVal
			}

			log := &models.ActivityLogDTO{
				Action:        "updated",
				ChangedBy:     dto.Actor.ID,
				ChangedByName: dto.Actor.Name,
				EntityType:    string(access.ResourceTicket),
				EntityID:      dto.ID,
				Entity:        oldTicket.Title,
			}
			if err := log.SetOldValues(oldMap); err != nil {
				return fmt.Errorf("set old values: %w", err)
			}
			if err := log.SetNewValues(newMap); err != nil {
				return fmt.Errorf("set new values: %w", err)
			}
			if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
				return fmt.Errorf("store logs: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(changes) > 0 {
		updated, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			s.notifyBestEffort("update", dto.ID, ticket.Title, err)
		} else if err := s.notifications.TicketUpdated(ctx, updated, dto.Actor.ID, changes); err != nil {
			s.notifyBestEffort("update", dto.ID, ticket.Title, err)
		}
	}
	return nil
}

// Transfer передаёт заявку от текущего исполнителя другому участнику той же
// группы. Только текущий исполнитель (assignee) может передать заявку; статус
// при передаче не меняется. Замороженные (resolved/closed/cancelled) заявки
// передавать нельзя.
func (s *TicketService) Transfer(ctx context.Context, dto *models.TransferTicketDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		return models.ErrTicketFrozen
	}

	// Передавать может только текущий исполнитель.
	if ticket.Assignee == nil || ticket.Assignee.ID != dto.Actor.ID {
		return models.ErrPermissionDenied
	}

	// Новый исполнитель обязан быть участником той же группы, что и тикет.
	if err := s.validateAssigneeForAssign(ctx, ticket, dto.AssigneeID); err != nil {
		return err
	}

	var realmID *uuid.UUID
	if dto.RealmID != "" {
		if rid, parseErr := uuid.Parse(dto.RealmID); parseErr == nil {
			realmID = &rid
		}
	}

	dtoUpdate := &models.TicketDTO{
		ID:         dto.ID,
		Actor:      dto.Actor,
		RealmID:    realmID,
		AssigneeID: dto.AssigneeID,
	}
	dtoUpdate.MarkProvided("assigneeId")

	var changes []*models.FieldChange
	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, loadErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if loadErr != nil {
			return loadErr
		}

		changes = dtoUpdate.GetChanges(oldTicket)

		if err := s.repo.Update(ctx, newTx, dtoUpdate); err != nil {
			return fmt.Errorf("failed to update ticket. error: %w", err)
		}

		if len(changes) > 0 {
			oldMap := make(map[string]string, len(changes))
			newMap := make(map[string]string, len(changes))
			for _, c := range changes {
				oldMap[string(c.Tag)] = c.OldVal
				newMap[string(c.Tag)] = c.NewVal
			}

			log := &models.ActivityLogDTO{
				Action:        "updated",
				ChangedBy:     dto.Actor.ID,
				ChangedByName: dto.Actor.Name,
				EntityType:    string(access.ResourceTicket),
				EntityID:      *dto.ID,
				Entity:        oldTicket.Title,
			}
			if err := log.SetOldValues(oldMap); err != nil {
				return fmt.Errorf("set old values: %w", err)
			}
			if err := log.SetNewValues(newMap); err != nil {
				return fmt.Errorf("set new values: %w", err)
			}
			if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
				return fmt.Errorf("store logs: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(changes) > 0 {
		updated, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if err != nil {
			s.notifyBestEffort("transfer", *dto.ID, ticket.Title, err)
		} else if err := s.notifications.TicketUpdated(ctx, updated, dto.Actor.ID, changes); err != nil {
			s.notifyBestEffort("transfer", *dto.ID, ticket.Title, err)
		}
	}
	return nil
}

// Delete удаляет тикет: проверяет право на удаление, сохраняет снимок данных
// в журнале и отправляет уведомление об удалении.
func (s *TicketService) Delete(ctx context.Context, dto *models.DeleteTicketDTO) error {
	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: dto.ID, UserID: dto.Actor.ID, Action: string(access.Delete)}); err != nil {
		return err
	}

	var ticket *models.Ticket
	// Директории вложений удаляются после коммита, поэтому id подзадач нужно
	// сохранить из транзакции наружу.
	var deletedSubIDs []uuid.UUID
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		var loadErr error
		ticket, loadErr = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if loadErr != nil {
			return fmt.Errorf("failed to load ticket: %w", loadErr)
		}

		snapshot := map[string]any{
			"title":    ticket.Title,
			"status":   ticket.Status,
			"priority": ticket.Priority,
		}
		if ticket.Assignee != nil {
			snapshot["assignee"] = ticket.Assignee.ID.String()
		}
		log := &models.ActivityLogDTO{
			Action:        "deleted",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      dto.ID,
			Entity:        ticket.Title,
		}
		if err := log.SetOldValues(snapshot); err != nil {
			return fmt.Errorf("set old values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		// Вложения не привязаны к тикету внешним ключом (полиморфный entity_id),
		// поэтому при удалении тикета их нужно вычищать явно: записи из БД
		// (ticket/{id} и subtask/{id}). Директории с файлами удаляются ПОСЛЕ
		// коммита транзакции — файловая система в неё не входит, и удаление до
		// коммита при откате (например, на s.repo.Delete ниже) потеряло бы файлы
		// безвозвратно. См. AttachmentService.DeleteByEntity / RemoveEntityDir.
		if err := s.attachments.DeleteByEntity(ctx, newTx, "ticket", dto.ID); err != nil {
			return fmt.Errorf("failed to cleanup ticket attachments: %w", err)
		}
		subs, subsErr := s.subtasks.GetByTicketID(ctx, dto.ID, dto.Actor.ID)
		if subsErr != nil {
			return fmt.Errorf("failed to load ticket subtasks: %w", subsErr)
		}
		subIDs := make([]uuid.UUID, 0, len(subs))
		for _, sub := range subs {
			if err := s.attachments.DeleteByEntity(ctx, newTx, "subtask", sub.ID); err != nil {
				return fmt.Errorf("failed to cleanup subtask attachments: %w", err)
			}
			subIDs = append(subIDs, sub.ID)
		}

		if err := s.repo.Delete(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to delete ticket. error: %w", err)
		}
		deletedSubIDs = subIDs
		return nil
	})
	if err != nil {
		return err
	}

	// Транзакция закоммичена — теперь директории можно снимать с диска.
	s.attachments.RemoveEntityDir("ticket", dto.ID)
	for _, subID := range deletedSubIDs {
		s.attachments.RemoveEntityDir("subtask", subID)
	}

	if err := s.notifications.TicketDeleted(ctx, ticket); err != nil {
		s.notifyBestEffort("delete", ticket.ID, ticket.Title, err)
	}

	return nil
}

// AutoCloseResolved автоматически закрывает resolved-тикеты с истёкшей
// задержкой и возвращает количество закрытых. При неположительной задержке
// ничего не делает.
func (s *TicketService) AutoCloseResolved(ctx context.Context, delay time.Duration) (int64, error) {
	if delay <= 0 {
		return 0, nil
	}

	cutoff := time.Now().Add(-delay)
	return s.repo.CloseResolved(ctx, cutoff)
}
