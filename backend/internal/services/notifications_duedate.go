package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// NotifyDueDateRequested — уведомление менеджеру группы о новом запросе исполнителя
// на перенос срока заявки. Получатель — менеджер тикета (ticket.Manager); если его
// нет — уведомление не отправляется.
func (s *NotificationService) NotifyDueDateRequested(ctx context.Context, ticket *models.Ticket, request *models.DueDateRequest) error {
	if ticket == nil || request == nil || ticket.Manager == nil {
		return nil
	}

	data, err := json.Marshal(map[string]any{
		"ticket_id":         ticket.ID.String(),
		"requester_id":      request.RequesterID.String(),
		"proposed_due_date": request.ProposedDueDate.Format(time.RFC3339),
		"current_due_date":  formatDueDateValue(request.CurrentDueDate),
		"reason":            request.Reason,
	})
	if err != nil {
		return err
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketDueDateRequested),
		Title: "Запрос на перенос срока",
		Body:  ticket.Title,
		Data:  data,
	}

	recipients := map[uuid.UUID]struct{}{ticket.Manager.ID: {}}
	delivered := s.deliver(ctx, ticket, recipientsToSlice(recipients), dto)
	s.persistForUsers(ctx, delivered, dto)

	return nil
}

// NotifyDueDateDecided — уведомление исполнителю о принятии решения по его запросу
// на перенос срока (утверждено/отклонено, с комментарием при отклонении).
func (s *NotificationService) NotifyDueDateDecided(ctx context.Context, ticket *models.Ticket, request *models.DueDateRequest) error {
	if ticket == nil || request == nil || request.RequesterID == uuid.Nil {
		return nil
	}

	data, err := json.Marshal(map[string]any{
		"ticket_id":         ticket.ID.String(),
		"approved":          request.Status == "approved",
		"proposed_due_date": request.ProposedDueDate.Format(time.RFC3339),
		"decision_comment":  request.DecisionComment,
	})
	if err != nil {
		return err
	}

	dto := &models.CreateNotificationDTO{
		UserID: request.RequesterID,
		Type:   string(models.NotificationTicketDueDateApproved),
		Title:  "Запрос на перенос срока",
		Body:   ticket.Title,
		Data:   data,
	}
	if request.Status != "approved" {
		dto.Type = string(models.NotificationTicketDueDateRejected)
	}

	recipients := map[uuid.UUID]struct{}{request.RequesterID: {}}
	delivered := s.deliver(ctx, ticket, recipientsToSlice(recipients), dto)
	s.persistForUsers(ctx, delivered, dto)

	return nil
}
