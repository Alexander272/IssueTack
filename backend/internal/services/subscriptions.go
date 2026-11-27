package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

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
// Ручной подписки больше нет: записи создаются только автоматически при создании заявки.
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
