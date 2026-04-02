package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type ActivityLogService struct {
	repo      repository.ActivityLog
	txManager TransactionManager
}

func NewActivityLogService(repo repository.ActivityLog, txManager TransactionManager) *ActivityLogService {
	return &ActivityLogService{
		repo:      repo,
		txManager: txManager,
	}
}

type ActivityLog interface {
	Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error)
	Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error
}

func (s *ActivityLogService) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity log. error: %w", err)
	}
	return data, nil
}

func (s *ActivityLogService) Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	if len(dto) == 0 {
		return nil
	}

	if tx == nil {
		// Если транзакция не передана, создаем новую
		return s.txManager.WithinTransaction(ctx, func(newTx postgres.Tx) error {
			return s.executeCreate(ctx, newTx, dto)
		})
	}
	// Если транзакция передана, используем её
	return s.executeCreate(ctx, tx, dto)
}
func (s *ActivityLogService) executeCreate(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	if err := s.repo.Create(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create activity log. error: %w", err)
	}
	return nil
}
