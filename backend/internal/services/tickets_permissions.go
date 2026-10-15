package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// isCreatorOrManager проверяет, является ли пользователь автором тикета или менеджером его группы —
// строгая проверка, используемая для закрытия/отмены тикета (см. модель доступа). Владелец здесь
// учитывается отдельно: для него допустимы только ограниченные переходы (ownerTransitionAllowed).
func (s *TicketService) isCreatorOrManager(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) (bool, error) {
	if ticket.Creator.ID == actorID {
		return true, nil
	}
	return s.isManagerOfGroup(ctx, ticket.Group, actorID)
}

// canChangeStatus определяет, может ли пользователь переводить тикет по статусам:
// только автор, исполнитель, владелец, менеджер группы или админ реалма (realm supervisor).
// Политика Casbin write сама по себе права на смену статуса не даёт — участник группы
// с realm-wide write-правом (не admin) менять статус не может. Закрытие/отмена дополнительно
// гейтятся isCreatorOrManager/владельцем (строгая проверка), терминальные статусы — ErrTicketFrozen.
func (s *TicketService) canChangeStatus(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, realm string) (bool, error) {
	if ticket.Creator.ID == userID {
		return true, nil
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == userID {
		return true, nil
	}
	if ticket.Owner != nil && ticket.Owner.ID == userID {
		return true, nil
	}
	isManager, err := s.isManagerOfGroup(ctx, ticket.Group, userID)
	if err != nil {
		return false, err
	}
	if isManager {
		return true, nil
	}
	return s.access.IsRealmSupervisor(ctx, userID, realm)
}

// isManagerOfGroup проверяет, является ли пользователь менеджером указанной группы тикета.
// Для тикета без группы всегда возвращает false. Определение менеджера — через GetManagedGroups.
func (s *TicketService) isManagerOfGroup(ctx context.Context, group *models.GroupShort, actorID uuid.UUID) (bool, error) {
	if group == nil {
		return false, nil
	}

	managed, err := s.groups.GetManagedGroups(ctx, actorID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get managed groups: %w", err)
	}
	for _, gid := range managed {
		if gid == group.ID {
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
// (не взята в работу). Начальника области это НЕ включает — он правит чужие заявки
// через отдельный режим adminOnlyEdit в Update (все поля кроме заголовка/описания).
// Политика Casbin write сама право правки полей не даёт, поэтому write-пользователи
// без роли здесь получают false.
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
// write/work-доступа). Владелец может только: open → cancelled (отменить новую заявку),
// resolved → closed (принять решение) и resolved → in_progress (вернуть в работу). Все
// прочие смены статусов и изменение полей для него запрещены (проверяется в Update).
func (s *TicketService) ownerTransitionAllowed(ticket *models.Ticket, dto *models.TicketDTO) bool {
	if !dto.HasField("status") || dto.Status == ticket.Status {
		return false
	}
	switch dto.Status {
	case models.StatusCancelled:
		return ticket.Status == models.StatusOpen
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
// Реалм не принимается: все проверки внутри берут его из ticket.RealmID (см.
// TicketAccessService.CheckAccessOnTicket), поэтому клиентский realm-заголовок
// не может расширить набор прав.
func (s *TicketService) GetAccessFlags(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (*models.AccessFlags, error) {
	flags := &models.AccessFlags{CanRead: true}

	writeErr := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Write))
	flags.CanWrite = writeErr == nil

	deleteErr := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Delete))
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

	// IsAdmin — начальник области (realm supervisor): пользователь с realm-wide
	// правами управления областью. Такому пользователю разрешены смена группы
	// заявки и передача исполнителя мимо менеджера группы. Намеренно не через
	// ticket:write — это право есть у рядовых сотрудников, и по нему фронт
	// разблокировал бы смену группы на чужих заявках.
	isAdmin, err := s.access.CanAdministerTickets(ctx, userID, ticket)
	if err != nil {
		return nil, err
	}
	flags.IsAdmin = isAdmin

	// IsManager — менеджер группы, к которой относится тикет (across-check never fails
	// для тикета без группы — false). Такому пользователю разрешена смена исполнителя.
	isManager, err := s.isManagerOfGroup(ctx, ticket.Group, userID)
	if err != nil {
		return nil, err
	}
	flags.IsManager = isManager

	flags.AllowedStatuses = s.computeAllowedStatuses(ctx, ticket, userID)

	return flags, nil
}

// computeAllowedStatuses вычисляет список статусов, доступных пользователю для перевода тикета,
// в зависимости от его роли (автор/менеджер группы, владелец, write/work-доступ) и текущего
// статуса. Логика отражает модель доступа: владелец без write/work получает только три перехода
// (см. ownerTransitionAllowed), исполнитель с work-доступом — активные статусы и resolved
// (при отсутствии незакрытых подзадач), закрытие/отмена активного тикета — только
// автору/менеджеру группы (владелец может отменить активный тикет).
func (s *TicketService) computeAllowedStatuses(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) []models.TicketStatus {
	hasWrite := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Write)) == nil
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
	// Инициализация обязательна: список уходит в JSON как AccessFlags.AllowedStatuses,
	// который во фронте объявлен TicketStatus[] — nil-слайс отдал бы null вопреки типу.
	allowed := []models.TicketStatus{}

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
