package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// Update обновляет поля тикета с учётом прав доступа и правил переходов по статусам.
// Пользователь с work-доступом может менять только статус; «чистый» владелец —
// только допустимые переходы из трёх (отменить, принять решение, вернуть в работу);
// статус closed можно поставить только из resolved.
func (s *TicketService) Update(ctx context.Context, dto *models.TicketDTO) error {
	var changes []*models.FieldChange
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		var txErr error
		changes, txErr = s.updateTicket(ctx, newTx, dto, nil)
		return txErr
	})
	if err != nil {
		return err
	}

	s.notifyTicketUpdated(ctx, dto, changes)
	return nil
}

// ticketUpdateOptions — необязательные уточнения updateTicket. Сейчас это только
// подмена тега в снапшоте журнала: уведомления и Mattermost получают исходные
// изменения DTO, а история может назвать изменение точнее.
type ticketUpdateOptions struct {
	logTagAlias map[models.ActivityType]models.ActivityType
}

// updateTicket — тело Update внутри уже открытой транзакции: проверки прав, запись
// изменений и журнал. Возвращает изменения для уведомления после коммита — их
// рассылает вызывающий (Update либо DecideDueDateRequest, который проводит решение
// по переносу срока и смену срока одной транзакцией).
func (s *TicketService) updateTicket(
	ctx context.Context,
	newTx postgres.Tx,
	dto *models.TicketDTO,
	opts *ticketUpdateOptions,
) (changes []*models.FieldChange, err error) {
	err = func() error {
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
				// Владелец заявки сохраняет ограниченный доступ к ней даже без
				// write/work-прав: ему доступны решения по статусам через
				// ownerTransitionAllowed. Автор без владельца остаётся «заказчиком»
				// своей заявки (см. canEditFields ниже).
				if !s.isOwnerOnlyOrCreatorWithoutOwner(old, dto.Actor.ID) {
					return err
				}
				ownerOnly = true
			}
			assignedOnly = true
		}

		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if err != nil {
			return err
		}

		// Серверная часть правила выбора заказчика (P2): явно выбранный заказчик
		// должен быть выбран разрешённым пользователем (менеджер/исполнитель) и быть
		// активным пользователем области — иначе изменение заказчика отклоняется.
		// Проверяем до fallback: снятый в форме заказчик (OwnerID == nil) — это не
		// явный выбор, а «без заказчика» → он заменяется создателем без валидации.
		// Эхо текущего заказчика (форма шлёт все поля разом) выбором не считается:
		// иначе автор без групп не смог бы отредактировать заголовок своей заявки.
		if dto.HasField("ownerId") && dto.OwnerID != nil &&
			(oldTicket.Owner == nil || *dto.OwnerID != oldTicket.Owner.ID) {
			if oldTicket.RealmID == nil {
				return models.ErrInvalidInput
			}
			if err := s.validateOwnerSelection(ctx, dto.Actor.ID, *oldTicket.RealmID, *dto.OwnerID); err != nil {
				return err
			}
		}

		// Снятый в форме заказчик трактуется как «заявка для создателя»: колонка
		// owner_id в БД NOT NULL, поэтому очистка поля не должна писать NULL.
		// Так же, как при создании без заказчика, им становится создатель.
		if dto.HasField("ownerId") && dto.OwnerID == nil {
			dto.OwnerID = &oldTicket.Creator.ID
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
		// Исключение: при возврате из resolved в in_progress менеджер может указать новый срок.
		if isTicketInactive(oldTicket.Status) {
			for _, change := range changes {
				// При возврате из resolved разрешены: смена статуса, срок (dueDate), закрытие.
				if change.Tag != models.ActionStatusChanged && change.Tag != models.ActionClosed && change.Tag != models.ActionDueDateChanged {
					return models.ErrTicketFrozen
				}
				// Срок разрешён только при переходе resolved → in_progress.
				if change.Tag == models.ActionDueDateChanged && (oldTicket.Status != models.StatusResolved || dto.Status != models.StatusInProgress) {
					return models.ErrTicketFrozen
				}
			}
		}

		// Тонкие права операций: смена группы и исполнителя доступна начальнику области,
		// менеджеру группы тикета или исполнителю области (участнику групп). Исполнителю
		// можно назначать любого активного исполнителя реалма. Менеджер (manager_id)
		// определяется менеджером группы автоматически — ручное изменение запрещено всем.
		// Эти ограничения действуют поверх прочих прав — независимо от assignedOnly/
		// ownerOnly и права правки полей (canEditFields). Эхо текущего значения не
		// создаёт change и проходит беспрепятственно.
		isAdmin, err := s.access.CanAdministerTickets(ctx, dto.Actor.ID, oldTicket)
		if err != nil {
			return fmt.Errorf("failed to check admin access: %w", err)
		}
		for _, change := range changes {
			switch change.Tag {
			case models.ActionGroupChanged, models.ActionGroupAssigned:
				if isAdmin {
					continue
				}
				isManager, mgrErr := s.isManagerOfGroup(ctx, oldTicket.Group, dto.Actor.ID)
				if mgrErr != nil {
					return mgrErr
				}
				if isManager {
					continue
				}
				if oldTicket.RealmID == nil {
					return models.ErrPermissionDenied
				}
				memberGroups, mgrErr := s.groups.GetMemberGroups(ctx, dto.Actor.ID, oldTicket.RealmID)
				if mgrErr != nil {
					return fmt.Errorf("failed to get member groups: %w", mgrErr)
				}
				if len(memberGroups) == 0 {
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
				if isManager {
					continue
				}
				if oldTicket.RealmID == nil || dto.AssigneeID == nil {
					return models.ErrPermissionDenied
				}
				memberGroups, mgrErr := s.groups.GetMemberGroups(ctx, dto.Actor.ID, oldTicket.RealmID)
				if mgrErr != nil {
					return fmt.Errorf("failed to get member groups: %w", mgrErr)
				}
				if len(memberGroups) == 0 {
					return models.ErrPermissionDenied
				}
				if err := s.validateRealmExecutor(ctx, *dto.AssigneeID, *oldTicket.RealmID); err != nil {
					return err
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

		// Для владельца/автора без write/work-доступа переходы по статусам уже
		// проверены через ownerTransitionAllowed выше; здесь ограничиваем правку
		// не-статусных полей через canEditFields: автор правит всегда, владелец —
		// только пока заявка ещё в статусе open.
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

		if dto.HasField("notifierIds") {
			if err := s.validateAndPrepareNotifiers(ctx, dto, oldTicket); err != nil {
				return err
			}
			if ch, ok := notifierIDsChange(oldTicket.Notifiers, dto.NotifierIDs); ok {
				changes = append(changes, ch)
			}
		}

		if err := s.repo.Update(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to update ticket. error: %w", err)
		}

		if dto.HasField("notifierIds") {
			if err := s.repo.SyncNotifiers(ctx, newTx, *dto.ID, dto.Actor.ID, dto.NotifierIDs); err != nil {
				return fmt.Errorf("failed to sync ticket notifiers: %w", err)
			}
		}

		if len(changes) > 0 {
			oldMap, newMap := s.logValues(ctx, oldTicket, changes)
			if opts != nil {
				renameLogTags(oldMap, newMap, opts.logTagAlias)
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
	}()
	return changes, err
}

// notifyTicketUpdated — уведомление об изменениях заявки. Вызывается только после

func hasActiveRealmMembership(u *models.UserData, realmID uuid.UUID) bool {
	for _, membership := range u.Realms {
		if membership != nil && membership.RealmID == realmID && membership.IsActive {
			return true
		}
	}
	return false
}

func (s *TicketService) validateAndPrepareNotifiers(ctx context.Context, dto *models.TicketDTO, oldTicket *models.Ticket) error {
	if !dto.HasField("notifierIds") {
		return nil
	}
	if dto.Actor == nil {
		return models.ErrPermissionDenied
	}

	var realmID uuid.UUID
	switch {
	case oldTicket != nil && oldTicket.RealmID != nil:
		realmID = *oldTicket.RealmID
	case dto.RealmID != nil:
		realmID = *dto.RealmID
	default:
		return models.ErrInvalidInput
	}

	checkTicket := oldTicket
	if checkTicket == nil {
		// На создании тикета ещё нет — достаточно знать область для проверки роли
		// «заказчик/не заказчик» (IsCustomer смотрит на группы/членство в реалме).
		checkTicket = &models.Ticket{RealmID: &realmID}
	} else if checkTicket.RealmID == nil {
		t := *checkTicket
		t.RealmID = &realmID
		checkTicket = &t
	}

	customer, err := s.access.IsCustomer(ctx, checkTicket, dto.Actor.ID)
	if err != nil {
		return fmt.Errorf("failed to check notifier edit permission: %w", err)
	}
	if customer {
		return models.ErrPermissionDenied
	}

	if len(dto.NotifierIDs) == 0 {
		dto.NotifierIDs = []uuid.UUID{}
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(dto.NotifierIDs))
	uniq := make([]uuid.UUID, 0, len(dto.NotifierIDs))
	for _, id := range dto.NotifierIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		u, err := s.users.GetByID(ctx, id)
		if err != nil || u == nil || !u.IsActive {
			return models.ErrInvalidInput
		}
		if !hasActiveRealmMembership(u, realmID) {
			return models.ErrInvalidInput
		}
		uniq = append(uniq, id)
	}

	dto.NotifierIDs = uniq
	return nil
}

func formatNotifierIDs(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "none"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return strings.Join(out, ",")
}

func notifierIDsChange(old []models.UserShort, newIDs []uuid.UUID) (*models.FieldChange, bool) {
	oldSet := make(map[string]struct{}, len(old))
	for _, u := range old {
		oldSet[u.ID.String()] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newIDs))
	for _, id := range newIDs {
		newSet[id.String()] = struct{}{}
	}
	if len(oldSet) == len(newSet) {
		same := true
		for id := range oldSet {
			if _, ok := newSet[id]; !ok {
				same = false
				break
			}
		}
		if same {
			return nil, false
		}
	}

	oldIDs := make([]uuid.UUID, 0, len(old))
	for _, u := range old {
		oldIDs = append(oldIDs, u.ID)
	}
	return &models.FieldChange{
		Tag:    models.ActionNotifiersChanged,
		OldVal: formatNotifierIDs(oldIDs),
		NewVal: formatNotifierIDs(newIDs),
	}, true
}

// GetAccessFlags вычисляет флаги доступа к тикету для пользователя
// (чтение, запись, удаление, работа) и список доступных ему переходов по статусам.
// Реалм не принимается: все проверки внутри берут его из ticket.RealmID (см.
// TicketAccessService.CheckAccessOnTicket), поэтому клиентский realm-заголовок

func (s *TicketService) notifyTicketUpdated(ctx context.Context, dto *models.TicketDTO, changes []*models.FieldChange) {
	if len(changes) == 0 {
		return
	}
	updated, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
	if err != nil {
		s.notifyBestEffort("update", *dto.ID, dto.Title, err)
		return
	}
	if err := s.notifications.TicketUpdated(ctx, updated, dto.Actor.ID, changes); err != nil {
		s.notifyBestEffort("update", *dto.ID, dto.Title, err)
	}
}

// Take — «взять заявку в работу»: пользователь назначает себя исполнителем
// (assignee) и переводит заявку в статус in_progress. Разрешено, если заявка
// активна (не resolved/closed/cancelled), пользователь имеет read-доступ и заявка
// «забираема» — не имеет исполнителя вовсе, либо исполнитель — другой пользователь,
// а заявка ещё в статусе open. Если исполнитель уже является текущим пользователем —

func (s *TicketService) notifyBestEffort(action string, entityID uuid.UUID, entity string, notifyErr error) {
	if notifyErr == nil {
		return
	}
	errMsg := fmt.Sprintf("failed to send notification (%s): %v", action, notifyErr)
	logger.Error(errMsg,
		logger.StringAttr("entity_id", entityID.String()),
		logger.StringAttr("entity", entity),
	)
	error_bot.Send(nil, errMsg, map[string]string{
		action:      entity,
		"entity_id": entityID.String(),
	})
}
