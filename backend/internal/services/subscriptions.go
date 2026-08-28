package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// TicketSubscriptionService — сервис управления индивидуальными подписками пользователей
// на уведомления по конкретной заявке.
type TicketSubscriptionService struct {
	repo         repository.TicketSubscriptions
	ticketRepo   repository.Tickets
	ticketAccess TicketAccessChecker
}

func NewTicketSubscriptionService(repo repository.TicketSubscriptions, ticketRepo repository.Tickets, ticketAccess TicketAccessChecker) *TicketSubscriptionService {
	return &TicketSubscriptionService{
		repo:         repo,
		ticketRepo:   ticketRepo,
		ticketAccess: ticketAccess,
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
	if err := s.ensureReadAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return err
	}
	return s.repo.Subscribe(ctx, nil, dto.TicketID, dto.ActorID)
}

// Unsubscribe отписывает пользователя от уведомлений по заявке.
func (s *TicketSubscriptionService) Unsubscribe(ctx context.Context, dto *models.SubscribeDTO) error {
	if err := s.ensureReadAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return err
	}
	return s.repo.Unsubscribe(ctx, nil, dto.TicketID, dto.ActorID)
}

// IsSubscribed проверяет, подписан ли пользователь на уведомления по заявке.
func (s *TicketSubscriptionService) IsSubscribed(ctx context.Context, dto *models.IsSubscribedDTO) (bool, error) {
	if err := s.ensureReadAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return false, err
	}
	return s.repo.Exists(ctx, dto.TicketID, dto.ActorID)
}

func (s *TicketSubscriptionService) ensureReadAccess(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return err
	}

	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	return s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: ticketID,
		UserID:   userID,
		Action:   string(access.Read),
		Realm:    realm,
	})
}
