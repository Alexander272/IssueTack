package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type TicketService struct {
	repo repository.Tickets
	tx   TransactionManager
	logs ActivityLog
}

func NewTicketService(repo repository.Tickets, tx TransactionManager, logs ActivityLog) *TicketService {
	return &TicketService{
		repo: repo,
		tx:   tx,
		logs: logs,
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
	return data, nil
}

func (s *TicketService) Create(ctx context.Context, dto *models.TicketDTO) error {
	return s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
		}
		return nil
	})
}

func (s *TicketService) Update(ctx context.Context, dto *models.TicketDTO) error {
	return s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}

		var logs []*models.ActivityLogDTO

		// 2. Сравниваем поля и готовим логи
		for _, change := range dto.GetChanges(oldTicket) {
			logs = append(logs, &models.ActivityLogDTO{
				TicketID: dto.ID,
				UserID:   dto.UserID,
				Type:     change.Tag,
				OldValue: &change.OldVal,
				NewValue: &change.NewVal,
			})
		}

		// 3. Сохраняем изменения тикета (SQL UPDATE)
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
}

func (s *TicketService) Delete(ctx context.Context, dto *models.DeleteTicketDTO) error {
	return s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		// TODO надо бы сделать какие ограничения для удаления тикета
		if err := s.repo.Delete(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to delete ticket. error: %w", err)
		}
		return nil
	})
}
