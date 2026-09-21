package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// Notifier — канал доставки уведомлений пользователю. Уведомление формируется в
// NotificationService один раз (нейтральный CreateNotificationDTO), а каждый канал
// сам решает, может ли он доставить данному пользователю (наличие адреса/настроек/
// интеграции) и как оформить сообщение. Notify возвращает признак фактической
// доставки: уведомление считается доставленным, если хотя бы один канал вернул true.
// Ошибки канала не фатальны — диспетчер логирует их (best-effort), а получатель не
// считается доставленным и будет обслужен повторно на следующей попытке.
type Notifier interface {
	// Name возвращает имя канала для диагностики в логах.
	Name() string
	// Notify доставляет уведомление одному пользователю и сообщает, удалось ли это.
	Notify(ctx context.Context, userID uuid.UUID, n *models.CreateNotificationDTO, ticket *models.Ticket) (delivered bool, err error)
}
