package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

// Update обновляет поля тикета с учётом прав доступа и правил переходов по статусам.
// Пользователь с work-доступом может менять только статус; «чистый» владелец —
// только допустимые переходы из трёх (отменить, принять решение, вернуть в работу);
// статус closed можно поставить только из resolved.
func (s *TicketService) Update(ctx context.Context, dto *models.TicketDTO) error {
	realmStr := ""
	if dto.RealmID != nil {
		realmStr = dto.RealmID.String()
	}

	assignedOnly := false
	adminOnlyEdit := false
	ownerOnly := false
	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: *dto.ID, UserID: dto.Actor.ID, Action: string(access.Write)}); err != nil {
		if workErr := s.access.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: *dto.ID, UserID: dto.Actor.ID}); workErr != nil {
			old, loadErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
			if loadErr != nil {
				return fmt.Errorf("failed to load ticket for access check: %w", loadErr)
			}
			if !s.isOwner(old, dto.Actor.ID) {
				return err
			}
			ownerOnly = true
		}
		assignedOnly = true
	}

	var changes []*models.FieldChange
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if err != nil {
			return err
		}

		if dto.HasField("status") && dto.Status != oldTicket.Status {
			ok, err := s.canChangeStatus(ctx, oldTicket, dto.Actor.ID, realmStr)
			if err != nil {
				return fmt.Errorf("failed to check status access: %w", err)
			}
			if !ok {
				return models.ErrPermissionDenied
			}
		}

		if ownerOnly && dto.HasField("status") && dto.Status != oldTicket.Status && !s.ownerTransitionAllowed(oldTicket, dto) {
			return models.ErrPermissionDenied
		}

		// Закрытая/отменённая заявка — терминальна: её статус изменить нельзя
		// (resolved при этом допускает переходы →closed и →in_progress выше).
		if dto.HasField("status") && oldTicket.Status != dto.Status &&
			(oldTicket.Status == models.StatusClosed || oldTicket.Status == models.StatusCancelled) {
			return models.ErrTicketFrozen
		}

		if dto.HasField("status") && dto.Status != oldTicket.Status &&
			(dto.Status == models.StatusClosed || dto.Status == models.StatusCancelled) {
			ok, err := s.isCreatorOrManager(ctx, oldTicket, dto.Actor.ID)
			if err != nil {
				return fmt.Errorf("failed to check close access: %w", err)
			}
			if !ok && s.isOwner(oldTicket, dto.Actor.ID) {
				ok = true
			}
			if !ok {
				return models.ErrPermissionDenied
			}
		}

		if dto.HasField("status") && dto.Status != oldTicket.Status {
			switch dto.Status {
			case models.StatusResolved:
				n, err := s.subtasks.GetUnresolvedCount(ctx, *dto.ID)
				if err != nil {
					return fmt.Errorf("failed to check subtasks: %w", err)
				}
				if n > 0 {
					return models.ErrSubtasksNotResolved
				}
			case models.StatusClosed:
				if oldTicket.Status != models.StatusResolved {
					return models.ErrCloseRequiresResolved
				}
			}
		}

		if dto.HasField("status") {
			now := time.Now()
			switch dto.Status {
			case models.StatusResolved:
				if oldTicket.ResolvedAt == nil {
					dto.ResolvedAt = &now
					dto.MarkProvided("resolvedAt")
				}
			case models.StatusClosed, models.StatusCancelled:
				if oldTicket.ClosedAt == nil {
					dto.ClosedAt = &now
					dto.MarkProvided("closedAt")
				}
			default:
				if oldTicket.ClosedAt != nil {
					dto.ClosedAt = nil
					dto.MarkProvided("closedAt")
				}
				if oldTicket.ResolvedAt != nil {
					dto.ResolvedAt = nil
					dto.MarkProvided("resolvedAt")
				}
			}
		}

		changes = dto.GetChanges(oldTicket)

		// Неактивные статусы (resolved/closed/cancelled) — «замороженные»: менять
		// разрешено только сам статус (resolved→closed, resolved→in_progress и т.п.).
		// Все прочие поля (срок, приоритет, исполнитель, заголовок и др.) изменить нельзя.
		if isTicketInactive(oldTicket.Status) {
			for _, change := range changes {
				if change.Tag != models.ActionStatusChanged && change.Tag != models.ActionClosed {
					return models.ErrTicketFrozen
				}
			}
		}

		// Тонкие права операций: смена группы заявки доступна только начальнику
		// области, смена исполнителя — начальнику области или менеджеру группы
		// тикета. Начальником области считается пользователь с realm-wide правами
		// управления областью (IsRealmSupervisor), а не обладатель ticket:write:
		// последнее есть у рядовых сотрудников и заявителей, и по нему они не должны
		// ни переносить заявку в другую группу, ни переназначать чужую. Менеджер
		// (manager_id) определяется менеджером группы автоматически — ручное
		// изменение запрещено всем. Эти ограничения действуют поверх прочих прав —
		// независимо от assignedOnly/ownerOnly и права правки полей (canEditFields).
		// Эхо текущего значения (фронт передаёт его в каждом запросе) не создаёт
		// change и проходит беспрепятственно.
		isAdmin, err := s.access.CanAdministerTickets(ctx, dto.Actor.ID, oldTicket)
		if err != nil {
			return fmt.Errorf("failed to check admin access: %w", err)
		}
		for _, change := range changes {
			switch change.Tag {
			case models.ActionGroupChanged, models.ActionGroupAssigned:
				if !isAdmin {
					return models.ErrPermissionDenied
				}
			case models.ActionAssigned, models.ActionAssignChanged:
				if isAdmin {
					continue
				}
				isManager, mgrErr := s.isManagerOfGroup(ctx, oldTicket.Group, dto.Actor.ID)
				if mgrErr != nil {
					return mgrErr
				}
				if !isManager {
					return models.ErrPermissionDenied
				}
			case models.ActionManagerChanged:
				return models.ErrPermissionDenied
			}
		}

		// Право правки полей (заголовок/описание/приоритет/срок/группа и т.п.)
		// имеет только создатель, менеджер группы или владелец (в статусе open).
		// Политика Casbin write сама по себе права правки полей не даёт — она
		// остаётся источником «рабочего» доступа (комментарии/вложения/подзадачи/
		// смена статуса). Пользователь лишь со «рабочим» доступом (assignedOnly)
		// может менять только статус; передача исполнителя выполняется через
		// отдельный эндпоинт Transfer (см. выше), а смена assignee via Update
		// доступна только админу/менеджеру группы (проверено в «тонких правах»).
		// Начальник области (realm supervisor), не являющийся создателем/
		// менеджером/владельцем тикета, получает отдельный режим adminOnlyEdit:
		// ему разрешено править любые поля кроме заголовка и описания.
		if !assignedOnly && !ownerOnly {
			canEdit, editErr := s.canEditFields(ctx, oldTicket, dto.Actor.ID)
			if editErr != nil {
				return editErr
			}
			if !canEdit {
				supervisor, supErr := s.access.IsRealmSupervisor(ctx, dto.Actor.ID, ticketRealm(oldTicket))
				if supErr != nil {
					return fmt.Errorf("failed to check supervisor access: %w", supErr)
				}
				if supervisor {
					adminOnlyEdit = true
				} else {
					assignedOnly = true
				}
			}
		}

		if assignedOnly && !ownerOnly {
			for _, change := range changes {
				switch change.Tag {
				case models.ActionStatusChanged, models.ActionClosed:
					// допустимо: изменение статуса
				case models.ActionGroupChanged, models.ActionGroupAssigned:
					// допустимо только для админа: авторизация проведена в «тонких правах» выше
				case models.ActionAssigned, models.ActionAssignChanged:
					// допустимо только для админа/менеджера группы: авторизация
					// проведена в «тонких правах» выше (исполнитель с work-доступом
					// сюда не попадает — его передача идёт через Transfer)
				default:
					return models.ErrPermissionDenied
				}
			}
		}

		// Режим правки начальника области: любые поля кроме заголовка/описания.
		// Смена статуса остаётся под canChangeStatus (проверен выше), закрытие/
		// отмена — под строгой проверкой, «замороженные» — под ErrTicketFrozen,
		// менеджер группы (manager_id) — под запретом в «тонких правах».
		if adminOnlyEdit && !ownerOnly {
			for _, change := range changes {
				switch change.Tag {
				case models.ActionTitleChanged, models.ActionDescriptionChanged:
					return models.ErrPermissionDenied
				}
			}
		}

		// Для владельца без write/work-доступа переходы по статусам уже проверены
		// через ownerTransitionAllowed выше; здесь ограничиваем правку не-статусных
		// полей: владелец может их менять только пока заявка ещё в статусе open.
		if ownerOnly {
			for _, change := range changes {
				if change.Tag == models.ActionStatusChanged || change.Tag == models.ActionClosed {
					continue
				}
				ownerEdit, editErr := s.canEditFields(ctx, oldTicket, dto.Actor.ID)
				if editErr != nil {
					return editErr
				}
				if !ownerEdit {
					return models.ErrPermissionDenied
				}
			}
		}

		if err := s.checkGroupRealm(ctx, oldTicket, dto); err != nil {
			return err
		}

		if err := s.repo.Update(ctx, newTx, dto); err != nil {
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
			s.notifyBestEffort("update", *dto.ID, dto.Title, err)
		} else if err := s.notifications.TicketUpdated(ctx, updated, dto.Actor.ID, changes); err != nil {
			s.notifyBestEffort("update", *dto.ID, dto.Title, err)
		}
	}
	return nil
}
