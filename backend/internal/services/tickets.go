package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
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
	categories    Categories
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
		categories:    deps.Categories,
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
	Categories    Categories
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
	// Take назначает пользователя исполнителем заявки и переводит её в статус
	// in_progress («взять в работу»). Доступно, если заявка активна и «забираема»:
	// исполнителя нет, либо исполнитель — другой пользователь и статус открыт.
	Take(ctx context.Context, dto *models.TakeTicketDTO) error
	// Transfer передаёт заявку от текущего исполнителя другому участнику той же
	// группы. Статус не меняется.
	Transfer(ctx context.Context, dto *models.TransferTicketDTO) error
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

	// Для страницы «Избранное» отдаём только избранные пользователя без ограничений по
	// группам/режиму: пользователь уже имеет read-доступ к своим избранным заявкам.
	if req.FavoritesByUser != nil {
		data, total, err := s.repo.Get(ctx, req)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
		}
		return data, total, nil
	}

	canViewAll, err := s.isRealmSupervisor(ctx, req.Actor.ID, realmStr)
	if err != nil {
		return nil, 0, fmt.Errorf("realm supervisor check failed: %w", err)
	}

	// «Мои задачи» (mode=assigned): лично назначенные ИЛИ задачи групп пользователя.
	// Работает и для рядовых пользователей, и для супервайзеров: супервайзер видит
	// в «Задачах» свои назначения и задачи групп, где состоит (а не все заявки реалма).
	if req.Mode != nil && *req.Mode == "assigned" {
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}
		groupIDs := member
		if !canViewAll {
			managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
			}
			groupIDs = unionGroupIDs(managed, member)
		}
		req.MyWork = &models.MyWorkFilter{UserID: req.Actor.ID, GroupIDs: groupIDs}
	} else if !canViewAll {
		managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
		}
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}

		// «Созданные вами заявки» (mode=created): только заявки, где пользователь автор,
		// без ограничения группой — иначе теряются внегрупповые заявки автора.
		if req.Mode != nil && *req.Mode == "created" {
			req.CreatorID = &req.Actor.ID
		} else {
			// Обзор без mode: все заявки групп пользователя + внегрупповые, назначенные ему.
			if groups := unionGroupIDs(managed, member); len(groups) > 0 {
				req.GroupIDs = groups
			}
			req.IncludeUngroupedAssignedTo = &req.Actor.ID
		}
	}

	data, total, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
	}
	return data, total, nil
}

// unionGroupIDs объединяет два списка групп без дубликатов (порядок: managed, затем member).
func unionGroupIDs(managed, member []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(managed)+len(member))
	out := make([]uuid.UUID, 0, len(managed)+len(member))
	for _, gid := range managed {
		if _, ok := seen[gid]; !ok {
			seen[gid] = struct{}{}
			out = append(out, gid)
		}
	}
	for _, gid := range member {
		if _, ok := seen[gid]; !ok {
			seen[gid] = struct{}{}
			out = append(out, gid)
		}
	}
	return out
}

// isRealmSupervisor определяет, является ли пользователь «начальником области» в реалме:
// ему выданы realm-wide пермишены управления областью (category:write или site:write).
// Такие пользователи видят все заявки реалма в списке.
func (s *TicketService) isRealmSupervisor(ctx context.Context, userID uuid.UUID, realmStr string) (bool, error) {
	return isRealmSupervisor(s.policies, userID, realmStr)
}

// isRealmSupervisor проверяет, является ли пользователь «начальником области» в реалме:
// ему выданы realm-wide пермишены управления областью (category:write или site:write).
// Это ролево-настраиваемый критерий (через выдачу прав ролям в БД), без хардкода конкретных ролей.
func isRealmSupervisor(policies AccessPolicies, userID uuid.UUID, realmStr string) (bool, error) {
	ok, err := policies.Enforce(userID.String(), realmStr, string(access.ResourceCategory), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return true, nil
	}
	ok, err = policies.Enforce(userID.String(), realmStr, string(access.ResourceSite), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	return ok, nil
}

// autoAssign при создании тикета достраивает поля из группы:
//   - исполнитель, если не указан явно: приоритет — ответственный по умолчанию группы
//     (DefaultAssigneeID), иначе — единственный участник группы, чтобы тикет не остался без
//     исполнителя. Если в группе несколько участников, исполнитель не назначается.
//   - менеджер (ManagerID) — менеджером группы (ManagerID), если не задан явно.
func (s *TicketService) autoAssign(ctx context.Context, dto *models.TicketDTO) error {
	group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	if dto.AssigneeID == nil {
		if group.DefaultAssigneeID != nil {
			dto.AssigneeID = group.DefaultAssigneeID
		} else {
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
		}
	}

	if dto.ManagerID == nil && group.ManagerID != nil {
		dto.ManagerID = group.ManagerID
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
		// Исполнитель (участник хотя бы одной группы) может создать заявку
		// с ограничениями: обязателен заказчик, а приоритет/группа/исполнитель/срок
		// принудительно определяются системой.
		member, err := s.groups.GetMemberGroups(ctx, dto.Actor.ID, dto.RealmID)
		if err != nil {
			return fmt.Errorf("failed to get member groups: %w", err)
		}
		if len(member) == 0 {
			return models.ErrPermissionDenied
		}
		if err := s.applyExecutorCreateRestrictions(ctx, dto, member); err != nil {
			return err
		}
	}

	if dto.GroupID == nil {
		return fmt.Errorf("group is required")
	}

	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if err := s.autoAssign(ctx, dto); err != nil {
		return fmt.Errorf("auto-assign: %w", err)
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
		s.notifyBestEffort(ctx, "create", *dto.ID, dto.Title, err)
	}
	return nil
}

// applyExecutorCreateRestrictions применяет ограничения при создании заявки исполнителем
// (участником группы, не имеющим полных прав). Заказчик обязателен; группа и приоритет
// принудительно берутся из категории, срок отсутствует. Если категория принадлежит одной
// из групп исполнителя — заявку назначают на него, иначе исполнитель подбирается по умолчанию.
func (s *TicketService) applyExecutorCreateRestrictions(ctx context.Context, dto *models.TicketDTO, memberGroups []uuid.UUID) error {
	if dto.OwnerID == nil {
		return models.ErrOwnerRequired
	}
	if dto.RealmID == nil {
		return fmt.Errorf("realm is required")
	}

	category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: *dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}
	if category.GroupID == uuid.Nil {
		return fmt.Errorf("category has no group")
	}

	dto.GroupID = &category.GroupID
	dto.Priority = category.Priority
	dto.DueDate = nil

	if containsUUID(memberGroups, category.GroupID) {
		dto.AssigneeID = &dto.CreatorID
	}
	return nil
}

func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
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

		// manager_id определяется менеджером группы автоматически; ручное изменение запрещено.
		if dto.HasField("managerId") {
			return models.ErrPermissionDenied
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

		// Право правки полей (заголовок/описание/приоритет/срок/группа и т.п.)
		// имеет только создатель, менеджер группы или владелец (в статусе open).
		// Политика Casbin write сама по себе права правки полей не даёт — она
		// остаётся источником «рабочего» доступа (комментарии/вложения/подзадачи/
		// смена статуса). Пользователь лишь со «рабочим» доступом может менять
		// только статус и передавать заявку исполнителю из своей группы (assigneeId).
		if !assignedOnly && !ownerOnly {
			canEdit, editErr := s.canEditFields(ctx, oldTicket, dto.Actor.ID)
			if editErr != nil {
				return editErr
			}
			if !canEdit {
				assignedOnly = true
			}
		}

		if assignedOnly && !ownerOnly {
			for _, change := range changes {
				switch change.Tag {
				case models.ActionStatusChanged, models.ActionClosed:
					// допустимо: изменение статуса
				case models.ActionAssigned, models.ActionAssignChanged:
					// допустимо только как передача новому исполнителю из той же группы
					if err := s.validateAssigneeForAssign(ctx, oldTicket, dto.AssigneeID); err != nil {
						return err
					}
				default:
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
			s.notifyBestEffort(ctx, "update", *dto.ID, dto.Title, err)
		}
	}
	return nil
}

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

	if err := s.access.CheckAccessOnTicket(ctx, ticket, dto.Actor.ID, string(access.Read), dto.RealmID); err != nil {
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
		if err := s.notifications.TicketUpdated(ctx, dto.ID, dto.Actor.ID, changes); err != nil {
			s.notifyBestEffort(ctx, "update", dto.ID, ticket.Title, err)
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
		if err := s.notifications.TicketUpdated(ctx, *dto.ID, dto.Actor.ID, changes); err != nil {
			s.notifyBestEffort(ctx, "transfer", *dto.ID, ticket.Title, err)
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
		s.notifyBestEffort(ctx, "delete", ticket.ID, ticket.Title, err)
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

// canEditFields определяет, может ли пользователь править не-статусные поля тикета
// (заголовок/описание/приоритет/срок/группа/исполнитель и т.п.): создатель или
// менеджер группы — всегда; владелец — только пока заявка ещё в статусе open
// (не взята в работу). Политика Casbin write сама право правки полей не даёт,
// поэтому write-пользователи без роли здесь получают false.
func (s *TicketService) canEditFields(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) (bool, error) {
	creatorOrManager, err := s.isCreatorOrManager(ctx, ticket, actorID)
	if err != nil {
		return false, err
	}
	if creatorOrManager {
		return true, nil
	}
	if s.isOwner(ticket, actorID) {
		return ticket.Status == models.StatusOpen, nil
	}
	return false, nil
}

// validateAssigneeForAssign проверяет, что смена исполнителя для пользователя с
// соответствующим (но не полным) доступом допустима: новый исполнитель должен быть
// участником той же группы, что и тикет, иное запрещено. Используется при передаче
// заявки исполнителем через Update и через Transfer.
func (s *TicketService) validateAssigneeForAssign(ctx context.Context, ticket *models.Ticket, assigneeID *uuid.UUID) error {
	if ticket.Group == nil || assigneeID == nil {
		return models.ErrPermissionDenied
	}
	ok, err := s.groups.IsMember(ctx, ticket.Group.ID, *assigneeID)
	if err != nil {
		return fmt.Errorf("failed to check assignee membership: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}
	return nil
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

	// CanTake — заявку можно «взять в работу»: она активна (не resolved/closed/cancelled)
	// и «забираема» — исполнителя нет совсем, либо исполнитель — другой пользователь,
	// а заявка ещё в статусе open (см. Take).
	flags.CanTake = !isTicketInactive(ticket.Status) &&
		(ticket.Assignee == nil ||
			(ticket.Assignee.ID != userID && ticket.Status == models.StatusOpen))

	// CanEditFields — право правки не-статусных полей (заголовок/описание и т.п.):
	// создатель или менеджер группы всегда, владелец — пока заявка в статусе open.
	// Политика write сама по себе права правки полей не даёт.
	canEdit, err := s.canEditFields(ctx, ticket, userID)
	if err != nil {
		return nil, err
	}
	flags.CanEditFields = canEdit

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
		// Закрытая/отменённая заявка — «замороженная»: вернуть её в активные статусы
		// нельзя никому (даже исполнителю с work-доступом). Разрешён лишь взаимообмен
		// closed⇄cancelled автору/менеджеру группы/владельцу.
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

// notifyBestEffort отправляет уведомление после фикса тикета. Ошибки
// уведомления не должны превращать уже совершённую операцию в 500 для клиента,
// но обязательно сигнализируются разработчику (error_bot) и в лог.
func (s *TicketService) notifyBestEffort(ctx context.Context, action string, entityID uuid.UUID, entity string, notifyErr error) {
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
