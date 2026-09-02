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
	tickets      Tickets
	ticketAccess TicketAccessChecker
}

func NewTicketSubscriptionService(repo repository.TicketSubscriptions, tickets Tickets, ticketAccess TicketAccessChecker) *TicketSubscriptionService {
	return &TicketSubscriptionService{
		repo:         repo,
		tickets:      tickets,
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
	ticket, err := s.tickets.GetSummary(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	if err := s.ticketAccess.CheckAccessOnTicket(ctx, ticket, userID, string(access.Read), realm); err != nil {
		return nil, err
	}

	allowed, err := s.ticketAccess.CanManage(ctx, userID, ticket)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrPermissionDenied
	}
	return ticket, nil
}

// TicketSubscriptionOps — операции с подписками на заявки, нужные сервисам без
// зависимости от тикетов. Отдельный тонкий сервис используется NotificationService,
// которому нужны выборки подписчиков и служебная авто-подписка без проверки прав
// (надзители реалма и менеджеры групп подписываются на создаваемую заявку без
// прохождения полного доступа).
type TicketSubscriptionOps interface {
	// GetByTicket возвращает ID всех подписанных на заявку пользователей.
	GetByTicket(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error)
	// GetSubscribersByEvent возвращает подписанных на заявку пользователей, у которых
	// включено событие eventField для категории categoryID тикета.
	GetSubscribersByEvent(ctx context.Context, ticketID, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error)
	// SubscribeInternal подписывает пользователя на заявку без проверки прав —
	// единственный write-порт, используется только для служебной авто-подписки
	// при создании заявки (autoSubscribeOnCreate).
	SubscribeInternal(ctx context.Context, ticketID, userID uuid.UUID) error
}

// TicketSubscriptionOpsService — тонкий сервис операций с подписками поверх репозитория.
// Не содержит бизнес-правил проверки доступа: для действий с подписками (subscribe/отписка)
// используется TicketSubscriptionService.
type TicketSubscriptionOpsService struct {
	repo repository.TicketSubscriptions
}

// NewTicketSubscriptionOpsService создаёт TicketSubscriptionOpsService.
func NewTicketSubscriptionOpsService(repo repository.TicketSubscriptions) *TicketSubscriptionOpsService {
	return &TicketSubscriptionOpsService{repo: repo}
}

// GetByTicket возвращает ID всех подписанных на заявку пользователей.
func (s *TicketSubscriptionOpsService) GetByTicket(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetByTicket(ctx, ticketID)
}

// GetSubscribersByEvent возвращает подписанных на заявку пользователей с включённым событием.
func (s *TicketSubscriptionOpsService) GetSubscribersByEvent(ctx context.Context, ticketID, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error) {
	return s.repo.GetSubscribersByEvent(ctx, ticketID, categoryID, eventField)
}

// SubscribeInternal подписывает пользователя на заявку без проверки прав.
func (s *TicketSubscriptionOpsService) SubscribeInternal(ctx context.Context, ticketID, userID uuid.UUID) error {
	return s.repo.Subscribe(ctx, nil, ticketID, userID)
}
