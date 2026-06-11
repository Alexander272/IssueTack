package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type TicketService struct {
	repo          repository.Tickets
	tx            TransactionManager
	logs          ActivityLog
	subtasks      Subtasks
	attachments   Attachments
	notifications Notifications
}

func NewTicketService(repo repository.Tickets, tx TransactionManager, logs ActivityLog, subtasks Subtasks, attachments Attachments, notifications Notifications) *TicketService {
	return &TicketService{
		repo:          repo,
		tx:            tx,
		logs:          logs,
		subtasks:      subtasks,
		attachments:   attachments,
		notifications: notifications,
	}
}

type Tickets interface {
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, error)
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	Create(ctx context.Context, dto *models.TicketDTO) error
	Update(ctx context.Context, dto *models.TicketDTO) error
	Delete(ctx context.Context, dto *models.DeleteTicketDTO) error
}

func (s *TicketService) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickets. error: %w", err)
	}
	return data, nil
}

func (s *TicketService) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	subtasks, err := s.subtasks.GetByTicketID(ctx, data.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	data.Subtasks = subtasks

	attachments, err := s.attachments.GetByEntity(ctx, "ticket", data.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	data.Attachments = attachments

	return data, nil
}

func (s *TicketService) Create(ctx context.Context, dto *models.TicketDTO) error {
	var ticket *models.Ticket
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
		}

		log := &models.ActivityLogDTO{
			Action:        "created",
			ChangedBy:     dto.UserID,
			ChangedByName: dto.UserName,
			EntityType:    "ticket",
			EntityID:      dto.ID,
			Entity:        dto.Title,
			NewValue:      &dto.Title,
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		var loadErr error
		ticket, loadErr = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if loadErr != nil {
			return fmt.Errorf("failed to load created ticket: %w", loadErr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := s.notifications.TicketCreated(ctx, ticket); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func (s *TicketService) Update(ctx context.Context, dto *models.TicketDTO) error {
	var changes []*models.FieldChange
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}

		changes = dto.GetChanges(oldTicket)

		var logs []*models.ActivityLogDTO
		for _, change := range changes {
			logs = append(logs, &models.ActivityLogDTO{
				Action:        string(change.Tag),
				ChangedBy:     dto.UserID,
				ChangedByName: dto.UserName,
				EntityType:    "ticket",
				EntityID:      dto.ID,
				Entity:        oldTicket.Title,
				OldValue:      &change.OldVal,
				NewValue:      &change.NewVal,
			})
		}

		if err := s.repo.Update(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to update ticket. error: %w", err)
		}

		if len(logs) > 0 {
			if err := s.logs.Create(ctx, newTx, logs); err != nil {
				return fmt.Errorf("store logs: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(changes) > 0 {
		if err := s.notifications.TicketUpdated(ctx, dto.ID, dto.UserID, changes); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
	}
	return nil
}

func (s *TicketService) Delete(ctx context.Context, dto *models.DeleteTicketDTO) error {
	return s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Delete(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to delete ticket. error: %w", err)
		}
		return nil
	})
	// Note: notification after transaction — no need to undo notification if transaction fails
	// s.notifications.TicketDeleted(dto.ID) // uncomment when needed
}
