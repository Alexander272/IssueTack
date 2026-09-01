package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// TicketSubscriptionService — сервис управления индивидуальными подписками пользователей
// на уведомления по конкретной заявке. Подписаться могут только надзители реалма и менеджеры групп.
type TicketSubscriptionService struct {
	repo         repository.TicketSubscriptions
	ticketRepo   repository.Tickets
	ticketAccess TicketAccessChecker
	policies     AccessPolicies
	groups       Groups
}

func NewTicketSubscriptionService(repo repository.TicketSubscriptions, ticketRepo repository.Tickets, ticketAccess TicketAccessChecker, policies AccessPolicies, groups Groups) *TicketSubscriptionService {
	return &TicketSubscriptionService{
		repo:         repo,
		ticketRepo:   ticketRepo,
		ticketAccess: ticketAccess,
		policies:     policies,
		groups:       groups,
	}
}

// Subscriptions — интерфейс управления подписками на заявки.
type Subscriptions interface {
	Subscribe(ctx context.Context, dto *models.SubscribeDTO) error
	Unsubscribe(ctx context.Context, dto *models.SubscribeDTO) error
	IsSubscribed(ctx context.Context, dto *models.IsSubscribedDTO) (bool, error)
}

// Subscribe подписывает пользователя на уведомления по заявке.
func (s *TicketSubscriptionService) Subscribe(ctx context.Context, dto *models.SubscribeDTO) error {
	if _, err := s.getTicketWithSubscribeAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return err
	}
	return s.repo.Subscribe(ctx, nil, dto.TicketID, dto.ActorID)
}

// Unsubscribe отписывает пользователя от уведомлений по заявке.
func (s *TicketSubscriptionService) Unsubscribe(ctx context.Context, dto *models.SubscribeDTO) error {
	if _, err := s.getTicketWithSubscribeAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return err
	}
	return s.repo.Unsubscribe(ctx, nil, dto.TicketID, dto.ActorID)
}

// IsSubscribed проверяет, подписан ли пользователь на уведомления по заявке.
func (s *TicketSubscriptionService) IsSubscribed(ctx context.Context, dto *models.IsSubscribedDTO) (bool, error) {
	if _, err := s.getTicketWithSubscribeAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return false, err
	}
	return s.repo.Exists(ctx, dto.TicketID, dto.ActorID)
}

// getTicketWithSubscribeAccess загружает тикет, проверяет у пользователя read-доступ и право
// подписки (надзитель реалма или менеджер группы заявки). Возвращает тикет для дальнейших действий.
func (s *TicketSubscriptionService) getTicketWithSubscribeAccess(ctx context.Context, ticketID, userID uuid.UUID) (*models.Ticket, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return nil, err
	}

	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: ticketID,
		UserID:   userID,
		Action:   string(access.Read),
		Realm:    realm,
	}); err != nil {
		return nil, err
	}

	allowed, err := s.subscribeAllowed(ctx, ticket, userID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrPermissionDenied
	}
	return ticket, nil
}

// subscribeAllowed возвращает true, если пользователь — надзитель реалма (category:write/site:write)
// или менеджер группы, к которой относится заявка.
func (s *TicketSubscriptionService) subscribeAllowed(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error) {
	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	supervisor, err := isRealmSupervisor(s.policies, userID, realm)
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
