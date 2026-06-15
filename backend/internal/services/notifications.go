package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/google/uuid"
)

type NotificationService struct {
	hub         *ws_hub.Hub
	repo        repository.Notifications
	ticketRepo  repository.Tickets
	txManager   TransactionManager
}

func NewNotificationService(hub *ws_hub.Hub, repo repository.Notifications, ticketRepo repository.Tickets, txManager TransactionManager) *NotificationService {
	return &NotificationService{
		hub:         hub,
		repo:        repo,
		ticketRepo:  ticketRepo,
		txManager:   txManager,
	}
}

type Notifications interface {
	TicketCreated(ctx context.Context, dto *models.TicketDTO) error
	TicketUpdated(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, changes []*models.FieldChange) error
	TicketDeleted(ctx context.Context, ticket *models.Ticket) error
	SendUnread(ctx context.Context, client *ws_hub.Client) error
}

func (s *NotificationService) TicketCreated(ctx context.Context, dto *models.TicketDTO) error {
	recipients := make(map[uuid.UUID]struct{})

	if dto.ManagerID != nil {
		recipients[*dto.ManagerID] = struct{}{}
	}

	responsible, err := s.repo.GetResponsibleByCategory(ctx, dto.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get responsible by category: %w", err)
	}
	for _, id := range responsible {
		recipients[id] = struct{}{}
	}

	data, _ := json.Marshal(map[string]interface{}{
		"ticket_id": dto.ID.String(),
		"title":     dto.Title,
	})

	for userID := range recipients {
		n := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.created",
			Title:  "Новая задача",
			Body:   dto.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, n); err != nil {
			log.Printf("failed to send notification to user %s: %v", userID, err)
		}
	}

	return nil
}

func (s *NotificationService) TicketUpdated(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, changes []*models.FieldChange) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to get ticket for notification: %w", err)
	}

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	for _, change := range changes {
		switch change.Tag {
		case models.ActionAssigned:
			newAssigneeID, _ := uuid.Parse(change.NewVal)
			if newAssigneeID == actorID {
				responsible, err := s.repo.GetResponsibleByCategory(ctx, ticket.Category.ID)
				if err != nil {
					return fmt.Errorf("failed to get responsible by category: %w", err)
				}
				for _, id := range responsible {
					recipients[id] = struct{}{}
				}
			} else {
				recipients[newAssigneeID] = struct{}{}
			}

		case models.ActionAssignChanged:
			newAssigneeID, _ := uuid.Parse(change.NewVal)
			recipients[newAssigneeID] = struct{}{}
		}
	}

	changesData, _ := json.Marshal(changes)
	data, _ := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   changesData,
	})

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.updated",
			Title:  "Задача обновлена",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			log.Printf("failed to send notification to user %s: %v", userID, err)
		}
	}

	return nil
}

func (s *NotificationService) TicketDeleted(ctx context.Context, ticket *models.Ticket) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	responsible, err := s.repo.GetResponsibleByCategory(ctx, ticket.Category.ID)
	if err != nil {
		return fmt.Errorf("failed to get responsible by category: %w", err)
	}
	for _, id := range responsible {
		recipients[id] = struct{}{}
	}

	data, _ := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.deleted",
			Title:  "Задача удалена",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			log.Printf("failed to send notification to user %s: %v", userID, err)
		}
	}

	return nil
}

func (s *NotificationService) SendUnread(ctx context.Context, client *ws_hub.Client) error {
	notifications, err := s.repo.GetUnread(ctx, client.UserID)
	if err != nil {
		return fmt.Errorf("failed to get unread notifications: %w", err)
	}

	for _, n := range notifications {
		if err := client.SendJSON("notification", n); err != nil {
			log.Printf("failed to send unread notification to user %s: %v", client.UserID, err)
		}
	}

	if len(notifications) > 0 {
		if err := s.repo.MarkAllRead(ctx, nil, client.UserID); err != nil {
			log.Printf("failed to mark notifications as read for user %s: %v", client.UserID, err)
		}
	}

	return nil
}

func (s *NotificationService) send(ctx context.Context, userID uuid.UUID, dto *models.CreateNotificationDTO) error {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get notification settings: %w", err)
	}

	var prefs map[string]bool
	if err := json.Unmarshal(settings.Settings, &prefs); err != nil {
		prefs = map[string]bool{"push": true}
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Create(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create notification: %w", err)
		}

		if prefs["push"] {
			eventData, _ := json.Marshal(map[string]interface{}{
				"type":   dto.Type,
				"title":  dto.Title,
				"body":   dto.Body,
				"data":   dto.Data,
			})
			s.hub.SendToUser(userID, eventData)
		}

		return nil
	})
}
