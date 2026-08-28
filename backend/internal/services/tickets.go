package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// TicketService — сервис тикетов: реализует бизнес-правила доступа,
// переходов по статусам, автозакрытия и журналирование действий.
type TicketService struct {
	repo          repository.Tickets
	tx            TransactionManager
	logs          ActivityLog
	subtasks      Subtasks
	attachments   Attachments
	notifications Notifications
	groups        Groups
	policies      AccessPolicies
	access        TicketAccessChecker
}

// NewTicketService создаёт новый сервис тикетов на основе переданных зависимостей.
func NewTicketService(deps *TicketDeps) *TicketService {
	return &TicketService{
		repo:          deps.Repo,
		tx:            deps.TxManager,
		logs:          deps.Logs,
		subtasks:      deps.Subtasks,
		attachments:   deps.Attachments,
		notifications: deps.Notifications,
		groups:        deps.Groups,
		policies:      deps.Policies,
		access:        deps.Access,
	}
}

// TicketDeps — зависимости, необходимые для создания TicketService.
type TicketDeps struct {
	Repo          repository.Tickets
	TxManager     TransactionManager
	Logs          ActivityLog
	Subtasks      Subtasks
	Attachments   Attachments
	Notifications Notifications
	Groups        Groups
	Policies      AccessPolicies
	Access        TicketAccessChecker
}

// Tickets — интерфейс сервиса тикетов: описывает чтение, создание,
// обновление и удаление тикетов, а также автоматическое закрытие.
type Tickets interface {
	// Get возвращает список тикетов с учётом прав актора и общее количество.
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error)
	// GetByID возвращает тикет по идентификатору вместе с подзадачами, вложениями и флагами доступа.
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	// Create создаёт новый тикет.
	Create(ctx context.Context, dto *models.TicketDTO) error
	// Update изменяет поля тикета с соблюдением правил доступа и переходов по статусам.
	Update(ctx context.Context, dto *models.TicketDTO) error
	// Delete удаляет тикет.
	Delete(ctx context.Context, dto *models.DeleteTicketDTO) error
	// AutoCloseResolved закрывает resolved-тикеты по истечении задержки и возвращает их количество.
	AutoCloseResolved(ctx context.Context, delay time.Duration) (int64, error)
}

// Get возвращает список тикетов и общее количество с учётом прав актора:
// при отсутствии повышенных прав список ограничивается его группами
// и тикетами, где он является создателем или исполнителем.
func (s *TicketService) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error) {
	realmStr := ""
	if req.RealmID != nil {
		realmStr = req.RealmID.String()
	}

	elevated, err := s.policies.Enforce(req.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return nil, 0, fmt.Errorf("policy check failed: %w", err)
	}
	if !elevated {
		elevated, err = s.policies.Enforce(req.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Delete))
		if err != nil {
			return nil, 0, fmt.Errorf("policy check failed: %w", err)
		}
	}

	if !elevated {
		managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
		}
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}

		seen := make(map[uuid.UUID]struct{})
		var all []uuid.UUID
		for _, gid := range managed {
			if _, ok := seen[gid]; !ok {
				seen[gid] = struct{}{}
				all = append(all, gid)
			}
		}
		for _, gid := range member {
			if _, ok := seen[gid]; !ok {
				seen[gid] = struct{}{}
				all = append(all, gid)
			}
		}

		if len(all) > 0 {
			req.GroupIDs = all
		}

		req.IncludeUngroupedAssignedTo = &req.Actor.ID
	}

	// Handle mode filtering
	if req.Mode != nil {
		switch *req.Mode {
		case "created":
			req.CreatorID = &req.Actor.ID
		case "assigned":
			req.AssigneeID = &req.Actor.ID
		}
	}

	data, total, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
	}
	return data, total, nil
}

// autoAssign назначает исполнителя при создании тикета, если исполнитель не указан явно:
// приоритет — ответственный по умолчанию группы (DefaultAssigneeID), иначе — единственный
// участник группы, чтобы тикет не остался без исполнителя. Если в группе несколько участников,
// исполнитель не назначается — его указывают позже вручную.
func (s *TicketService) autoAssign(ctx context.Context, dto *models.TicketDTO) error {
	group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	if group.DefaultAssigneeID != nil {
		dto.AssigneeID = group.DefaultAssigneeID
		return nil
	}

	count, err := s.groups.GetMemberCount(ctx, *dto.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get member count: %w", err)
	}
	if count == 1 {
		members, err := s.groups.GetMembers(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
		if err != nil {
			return fmt.Errorf("failed to get members: %w", err)
		}
		if len(members) > 0 {
			dto.AssigneeID = &members[0].ID
		}
	}
	return nil
}

// GetByID возвращает тикет по идентификатору с проверкой права чтения,
// дополняя его подзадачами, вложениями и флагами доступа.
func (s *TicketService) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: req.ID, UserID: req.Actor.ID, Action: string(access.Read), Realm: req.RealmID}); err != nil {
		return nil, err
	}

	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	subtasks, err := s.subtasks.GetByTicketID(ctx, data.ID, req.Actor.ID, req.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	data.Subtasks = subtasks

	attachments, err := s.attachments.GetByEntity(ctx, &models.EntityAccessDTO{
		EntityType: string(access.ResourceTicket),
		EntityID:   data.ID,
		ActorID:    req.Actor.ID,
		Realm:      req.RealmID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	data.Attachments = attachments

	realmStr := ""
	if req.RealmID != "" {
		realmStr = req.RealmID
	}
	flags, err := s.GetAccessFlags(ctx, data, req.Actor.ID, realmStr)
	if err == nil {
		data.Access = flags
	}

	return data, nil
}

// Create создаёт новый тикет: проверяет права на создание, заполняет
// владельца/исполнителя (при необходимости — автоматически), фиксирует
// действие в журнале и отправляет уведомление.
func (s *TicketService) Create(ctx context.Context, dto *models.TicketDTO) error {
	realmStr := ""
	if dto.RealmID != nil {
		realmStr = dto.RealmID.String()
	}

	ok, err := s.policies.Enforce(dto.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}

	if dto.GroupID == nil {
		return fmt.Errorf("group is required")
	}

	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if dto.AssigneeID == nil {
		if err := s.autoAssign(ctx, dto); err != nil {
			return fmt.Errorf("auto-assign: %w", err)
		}
	}

	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
		}

		log := &models.ActivityLogDTO{
			Action:        "created",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      *dto.ID,
			Entity:        dto.Title,
		}
		if err := log.SetNewValues(map[string]string{"title": dto.Title}); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := s.notifications.TicketCreated(ctx, dto); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

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
	ownerOnly := false
	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: *dto.ID, UserID: dto.Actor.ID, Action: string(access.Write), Realm: realmStr}); err != nil {
		if workErr := s.access.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: *dto.ID, UserID: dto.Actor.ID, Realm: realmStr}); workErr != nil {
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

		if ownerOnly && !s.ownerTransitionAllowed(oldTicket, dto) {
			return models.ErrPermissionDenied
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

		if assignedOnly {
			for _, change := range changes {
				if change.Tag != models.ActionStatusChanged && change.Tag != models.ActionClosed {
					return models.ErrPermissionDenied
				}
			}
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
		if err := s.notifications.TicketUpdated(ctx, *dto.ID, dto.Actor.ID, changes); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
	}
	return nil
}

// Delete удаляет тикет: проверяет право на удаление, сохраняет снимок данных
// в журнале и отправляет уведомление об удалении.
func (s *TicketService) Delete(ctx context.Context, dto *models.DeleteTicketDTO) error {
	if err := s.access.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: dto.ID, UserID: dto.Actor.ID, Action: string(access.Delete), Realm: dto.RealmID}); err != nil {
		return err
	}

	var ticket *models.Ticket
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		var loadErr error
		ticket, loadErr = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if loadErr != nil {
			return fmt.Errorf("failed to load ticket: %w", loadErr)
		}

		snapshot := map[string]interface{}{
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

		if err := s.repo.Delete(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to delete ticket. error: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := s.notifications.TicketDeleted(ctx, ticket); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// isCreatorOrManager проверяет, является ли пользователь автором тикета или менеджером его группы —
// строгая проверка, используемая для закрытия/отмены тикета (см. модель доступа). Владелец здесь
// учитывается отдельно: для него допустимы только ограниченные переходы (ownerTransitionAllowed).
func (s *TicketService) isCreatorOrManager(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) (bool, error) {
	if ticket.Creator.ID == actorID {
		return true, nil
	}
	if ticket.Group == nil {
		return false, nil
	}

	managed, err := s.groups.GetManagedGroups(ctx, actorID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get managed groups: %w", err)
	}
	for _, gid := range managed {
		if gid == ticket.Group.ID {
			return true, nil
		}
	}
	return false, nil
}

func (s *TicketService) isOwner(ticket *models.Ticket, actorID uuid.UUID) bool {
	return ticket.Owner != nil && ticket.Owner.ID == actorID
}

// ownerTransitionAllowed — допустимые переходы по статусам для «чистого» владельца (без
// write/work-доступа). Владелец может только: активный статус → cancelled (отменить заявку),
// resolved → closed (принять решение) и resolved → in_progress (вернуть в работу). Все прочие
// смены статусов и изменение полей для него запрещены (проверяется в Update).
func (s *TicketService) ownerTransitionAllowed(ticket *models.Ticket, dto *models.TicketDTO) bool {
	if !dto.HasField("status") || dto.Status == ticket.Status {
		return false
	}
	switch dto.Status {
	case models.StatusCancelled:
		return ticket.Status != models.StatusResolved &&
			ticket.Status != models.StatusClosed &&
			ticket.Status != models.StatusCancelled
	case models.StatusClosed:
		return ticket.Status == models.StatusResolved
	case models.StatusInProgress:
		return ticket.Status == models.StatusResolved
	default:
		return false
	}
}

// isTicketInactive возвращает true, если тикет находится в «замороженном»
// (неактивном) статусе: resolved, closed или cancelled. Для таких тикетов
// изменение данных (комментарии, вложения, подзадачи, поля) запрещено.
func isTicketInactive(status models.TicketStatus) bool {
	switch status {
	case models.StatusResolved, models.StatusClosed, models.StatusCancelled:
		return true
	default:
		return false
	}
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

// activeStatuses — статусы, которые считаются «активными» (тикет в работе): из/в них разрешены
// переходы, а их наличие блокирует закрытие через resolved (см. GetUnresolvedCount).
var activeStatuses = []models.TicketStatus{
	models.StatusOpen,
	models.StatusInProgress,
	models.StatusPending,
	models.StatusOnHold,
}

// GetAccessFlags вычисляет флаги доступа к тикету для пользователя
// (чтение, запись, удаление, работа) и список доступных ему переходов по статусам.
func (s *TicketService) GetAccessFlags(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, realm string) (*models.AccessFlags, error) {
	flags := &models.AccessFlags{CanRead: true}

	writeErr := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Write), realm)
	flags.CanWrite = writeErr == nil

	deleteErr := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Delete), realm)
	flags.CanDelete = deleteErr == nil

	flags.CanWork = flags.CanWrite || (ticket.Assignee != nil && ticket.Assignee.ID == userID)

	flags.AllowedStatuses = s.computeAllowedStatuses(ctx, ticket, userID, realm)

	return flags, nil
}

// computeAllowedStatuses вычисляет список статусов, доступных пользователю для перевода тикета,
// в зависимости от его роли (автор/менеджер группы, владелец, write/work-доступ) и текущего
// статуса. Логика отражает модель доступа: владелец без write/work получает только три перехода
// (см. ownerTransitionAllowed), исполнитель с work-доступом — активные статусы и resolved
// (при отсутствии незакрытых подзадач), закрытие/отмена активного тикета — только
// автору/менеджеру группы (владелец может отменить активный тикет).
func (s *TicketService) computeAllowedStatuses(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, realm string) []models.TicketStatus {
	hasWrite := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Write), realm) == nil
	isCreator := ticket.Creator.ID == userID

	isMgr := false
	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err == nil {
			for _, gid := range managed {
				if gid == ticket.Group.ID {
					isMgr = true
					break
				}
			}
		}
	}
	isCreatorOrMgr := isCreator || isMgr
	isOwner := ticket.Owner != nil && ticket.Owner.ID == userID
	hasWork := hasWrite || (ticket.Assignee != nil && ticket.Assignee.ID == userID)

	current := ticket.Status
	var allowed []models.TicketStatus

	// add добавляет статусы в список разрешённых, исключая дубликаты
	// (одна роль может давать статус, уже добавленный другой ролью).
	add := func(statuses ...models.TicketStatus) {
		for _, st := range statuses {
			found := false
			for _, a := range allowed {
				if a == st {
					found = true
					break
				}
			}
			if !found {
				allowed = append(allowed, st)
			}
		}
	}

	switch {
	case current == models.StatusClosed || current == models.StatusCancelled:
		if hasWrite || hasWork {
			add(activeStatuses...)
		}
		if isCreatorOrMgr || isOwner {
			if current == models.StatusClosed {
				add(models.StatusCancelled)
			} else {
				add(models.StatusClosed)
			}
		}

	case current == models.StatusResolved:
		if isCreatorOrMgr || isOwner {
			add(models.StatusClosed)
		}
		if hasWrite || hasWork || isOwner {
			add(models.StatusInProgress)
		}
		if hasWrite || hasWork {
			for _, st := range activeStatuses {
				if st != models.StatusInProgress {
					add(st)
				}
			}
		}
		if isCreatorOrMgr {
			add(models.StatusCancelled)
		}

	default:
		if hasWrite || hasWork {
			for _, st := range activeStatuses {
				if st != current {
					add(st)
				}
			}
			n, err := s.subtasks.GetUnresolvedCount(ctx, ticket.ID)
			if err == nil && n == 0 {
				add(models.StatusResolved)
			}
		}
		if isCreatorOrMgr || isOwner {
			add(models.StatusCancelled)
		}
	}

	return allowed
}
