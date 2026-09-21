package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// Notifier — канал доставки уведомлений пользователю. Уведомление формируется в
// NotificationService один раз (нейтральный CreateNotificationDTO), а каждый канал
// сам решает, может ли он доставить данному пользователю (наличие адреса/настроек/
// интеграции) и как оформить сообщение. Ошибки доставки не фатальны — диспетчер
// только логирует их (best-effort, после фикса транзакции).
type Notifier interface {
	// Name возвращает имя канала для диагностики в логах.
	Name() string
	// Notify доставляет уведомление одному пользователю.
	Notify(ctx context.Context, userID uuid.UUID, n *models.CreateNotificationDTO, ticket *models.Ticket) error
}
