package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

// NotifyDuplicate рассылает событие связывания/отвязки дубля. Получатели —
// автор и заказчик дубля, а также автор и заказчик основной заявки; актор
// уведомление о собственном действии не получает. Опрос качества здесь не
// отправляется: дубль — не новое решение, а уточнение связи.
func (s *NotificationService) NotifyDuplicate(ctx context.Context, ticket, primary *models.Ticket, actorID uuid.UUID, linked bool) error {
	if ticket == nil || primary == nil {
		return nil
	}

	recipients := make(map[uuid.UUID]struct{})
	add := func(u *models.UserShort) {
		if u == nil || u.ID == uuid.Nil || u.ID == actorID {
			return
		}
		recipients[u.ID] = struct{}{}
	}
	add(&ticket.Creator)
	add(ticket.Owner)
	add(&primary.Creator)
	add(primary.Owner)

	if len(recipients) == 0 {
		return nil
	}

	event := models.NotificationDuplicateLinked
	title := "Заявка отмечена дублем"
	if !linked {
		event = models.NotificationDuplicateUnlinked
		title = "Дубль отвязан"
	}

	data, err := json.Marshal(map[string]any{
		"ticket_id":         ticket.ID.String(),
		"primary_ticket_id": primary.ID.String(),
		"title":             ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal duplicate notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(event),
		Title: title,
		// ticketRef уже печатает «№N» — лишний префикс давал бы «№№123 → №456».
		Body: fmt.Sprintf("%s → %s", ticketRef(ticket), ticketRef(primary)),
		Data: data,
	}
	delivered := s.deliver(ctx, ticket, recipientsToSlice(recipients), dto)
	s.persistForUsers(ctx, delivered, dto)
	return nil
}
