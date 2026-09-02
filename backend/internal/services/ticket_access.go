package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// TicketAccessChecker реализует модель доступа к тикетам (read/write/delete/work-доступ).
type TicketAccessChecker interface {
	// CheckAccess проверяет право пользователя на заданное действие над тикетом
	// (read/write/delete) согласно модели доступа. Если Casbin-правило не разрешает,
	// используется логика по атрибутам тикета.
	CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckWorkAccess проверяет право на "рабочий" доступ к тикету:
	// write-доступ или пользователь является исполнителем (assignee).
	CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckInternalAssigneeAccess проверяет, что пользователь является исполнителем (assignee)
	// тикета или менеджером его группы.
	CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	// CheckAccessOnTicket выполняет проверку доступа по уже загруженному тикету —
	// общая основа для CheckAccess и логики статусов.
	CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string, realm string) error
	// IsRealmSupervisor определяет, является ли пользователь «начальником области» в реалме:
	// ему выданы realm-wide пермишены управления областью (category:write или site:write).
	// Единственная точка владения этой проверкой — через policies.enforcer.
	IsRealmSupervisor(ctx context.Context, userID uuid.UUID, realm string) (bool, error)
	// CanManage проверяет, может ли пользователь «управлять» тикетом: является ли он
	// начальником области (realm supervisor) или менеджером группы, к которой относится
	// тикет. Используется для админских операций поверх тикета (подписки и т.п.).
	CanManage(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error)
	// CanCreateTicket проверяет наличие realm-wide write-политики на ресурс тикетов —
	// достаточно ли прав создать заявку напрямую (без ограничений «рабочего» режима).
	CanCreateTicket(ctx context.Context, userID uuid.UUID, realm string) (bool, error)
}

// TicketAccessService реализует проверку прав доступа к тикетам.
type TicketAccessService struct {
	repo     repository.Tickets
	groups   Groups
	policies AccessPolicies
}

// NewTicketAccessService создаёт TicketAccessService.
func NewTicketAccessService(repo repository.Tickets, groups Groups, policies AccessPolicies) *TicketAccessService {
	return &TicketAccessService{
		repo:     repo,
		groups:   groups,
		policies: policies,
	}
}

// CheckAccessOnTicket — сердце модели доступа к тикету, работает по уже загруженному тикету
// (общая основа для CheckAccess и проверок способностей в TicketService.Update).
// Сначала Enforce по Casbin: если явное правило (в контексте realm) разрешает действие,
// тикет доступен без проверки атрибутов. Иначе применяется модель по атрибутам тикета:
//   - Read: участник группы, менеджер группы или исполнитель; у тикета без группы —
//     также его создатель;
//   - Write: создатель тикета или менеджер группы;
//   - Delete: только менеджер группы — создатель удалять не может.
//
// Менеджер группы определяется перебором GetManagedGroups, поэтому на одно действие
// управляемые группы запрашиваются один раз. Если ни одно из правил не сработало — отказ.
func (s *TicketAccessService) CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string, realm string) error {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceTicket), action)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return nil
	}

	if ticket.Group == nil {
		if action == string(access.Read) && (ticket.Assignee != nil && ticket.Assignee.ID == userID || ticket.Creator.ID == userID) {
			return nil
		}
		return models.ErrPermissionDenied
	}

	switch action {
	case string(access.Read):
		isMember, err := s.groups.IsMember(ctx, ticket.Group.ID, userID)
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if isMember {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
		if ticket.Assignee != nil && ticket.Assignee.ID == userID {
			return nil
		}
	case string(access.Write):
		if ticket.Creator.ID == userID {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	case string(access.Delete):
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	}
	return models.ErrPermissionDenied
}

// CheckAccess — точка входа проверки доступа по DTO. Сначала Casbin-Enforce по realm:
// если явное правило сработало, тикет даже не загружаем. Лишь когда правила нет —
// подгружаем тикет и делегируем единой логике CheckAccessOnTicket, чтобы вся модель
// доступа по атрибутам была описана в одном месте.
func (s *TicketAccessService) CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ok, err := s.policies.Enforce(dto.UserID.String(), dto.Realm, string(access.ResourceTicket), dto.Action)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return nil
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket for access check: %w", err)
	}
	return s.CheckAccessOnTicket(ctx, ticket, dto.UserID, dto.Action, dto.Realm)
}

// CheckWorkAccess — "рабочий" доступ к тикету: либо write-доступ (по модели CheckAccess),
// либо пользователь является исполнителем. Исполнителю разрешены операции ведения тикета —
// смена статуса, подзадачи, вложения, комментарии — даже без прав создателя/менеджера.
//
// Тикет загружается один раз в начале метода, чтобы:
//   - запретить любые "рабочие" операции (комментарии, вложения, подзадачи) для заявок
//     в неактивном статусе (resolved/closed/cancelled) — независимо от прав пользователя;
//   - затем проверить доступ по атрибутам загруженного тикета (write или исполнитель),
//     не выполняя двойную загрузку.
func (s *TicketAccessService) CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		return models.ErrTicketFrozen
	}

	if err := s.CheckAccessOnTicket(ctx, ticket, dto.UserID, string(access.Write), dto.Realm); err == nil {
		return nil
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}
	return models.ErrPermissionDenied
}

// CheckInternalAssigneeAccess — узкая "ролевая" проверка без Casbin-домена (realm)
// и без учёта создателя: пользователь должен быть исполнителем (assignee) тикета или
// менеджером его группы. Используется там, где важно именно членство роли, а не write-права —
// например, чтобы решить, показывать ли пользователю внутренние комментарии
// (см. CommentService.GetByTicket).
func (s *TicketAccessService) CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}

	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}

	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, dto.UserID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	}

	return models.ErrPermissionDenied
}

// IsRealmSupervisor определяет, является ли пользователь «начальником области» в реалме:
// ему выданы realm-wide пермишены управления областью (category:write или site:write).
// Это ролево-настраиваемый критерий (через выдачу прав ролям в БД), без хардкода конкретных ролей.
func (s *TicketAccessService) IsRealmSupervisor(ctx context.Context, userID uuid.UUID, realm string) (bool, error) {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceCategory), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return true, nil
	}
	ok, err = s.policies.Enforce(userID.String(), realm, string(access.ResourceSite), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	return ok, nil
}

// CanManage проверяет, может ли пользователь «управлять» тикетом: начальник области
// (realm supervisor) или менеджер группы, к которой относится тикет.
func (s *TicketAccessService) CanManage(ctx context.Context, userID uuid.UUID, ticket *models.Ticket) (bool, error) {
	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	supervisor, err := s.IsRealmSupervisor(ctx, userID, realm)
	if err != nil {
		return false, err
	}
	if supervisor {
		return true, nil
	}

	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, userID, ticket.RealmID)
		if err != nil {
			return false, err
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return true, nil
			}
		}
	}
	return false, nil
}

// CanCreateTicket проверяет наличие realm-wide write-политики на ресурс тикетов.
func (s *TicketAccessService) CanCreateTicket(ctx context.Context, userID uuid.UUID, realm string) (bool, error) {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return false, fmt.Errorf("policy check failed: %w", err)
	}
	return ok, nil
}
