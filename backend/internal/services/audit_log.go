package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type auditLogService struct {
	repo repository.AuditLogs
	tm   TransactionManager
}

func NewAuditLogService(repo repository.AuditLogs, tm TransactionManager) *auditLogService {
	return &auditLogService{
		repo: repo,
		tm:   tm,
	}
}

type AuditLogs interface {
	StartListening(bus *events.PolicyEventManager)
	Get(ctx context.Context, req *models.GetAuditLogsDTO) ([]*models.AuditLog, error)
	GetByRealm(ctx context.Context, req *models.GetAuditLogsByRealmDTO) ([]*models.AuditLog, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.AuditLogDTO) error
}

func (s *auditLogService) StartListening(bus *events.PolicyEventManager) {
	go func() {
		events := bus.Subscribe()
		for event := range events {
			dto := &models.AuditLogDTO{
				ChangedBy: event.ChangedBy,
				Action:    event.Action,
				RoleID:    event.RoleID,
				RuleID:    event.RuleID,
				RealmID:   event.RealmID,
				UserID:    event.UserID,
				OldValues: event.OldValues,
				NewValues: event.NewValues,
			}

			// Записываем в базу данных
			s.Create(context.Background(), nil, dto)
		}
	}()
}

func (s *auditLogService) Get(ctx context.Context, req *models.GetAuditLogsDTO) ([]*models.AuditLog, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs. error: %w", err)
	}
	return data, nil
}

func (s *auditLogService) GetByRealm(ctx context.Context, req *models.GetAuditLogsByRealmDTO) ([]*models.AuditLog, error) {
	data, err := s.repo.GetByRealm(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs. error: %w", err)
	}
	return data, nil
}

func (s *auditLogService) Create(ctx context.Context, tx postgres.Tx, dto *models.AuditLogDTO) error {
	if tx == nil {
		return s.tm.WithinTransaction(ctx, func(newTx postgres.Tx) error {
			return s.executeCreate(ctx, newTx, dto)
		})
	}
	return s.executeCreate(ctx, tx, dto)
}

func (s *auditLogService) executeCreate(ctx context.Context, tx postgres.Tx, dto *models.AuditLogDTO) error {
	if err := s.repo.Create(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create activity log. error: %w", err)
	}
	return nil
}
