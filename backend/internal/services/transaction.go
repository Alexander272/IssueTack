package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
)

// TransactionManagerService — сервис управления транзакциями БД.
type TransactionManagerService struct {
	repo repository.Transaction
}

// NewTransactionManager создаёт TransactionManagerService.
func NewTransactionManager(repo repository.Transaction) *TransactionManagerService {
	return &TransactionManagerService{repo: repo}
}

// TransactionManager — интерфейс выполнения функции внутри транзакции БД.
type TransactionManager interface {
	// WithinTransaction выполняет функцию внутри транзакции.
	WithinTransaction(ctx context.Context, fn func(tx postgres.Tx) error) error
}

// WithinTransaction выполняет функцию внутри транзакции, откатывая её при ошибке или панике и коммитя при успехе.
func (tm *TransactionManagerService) WithinTransaction(ctx context.Context, fn func(tx postgres.Tx) error) error {
	tx, err := tm.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logger.Warn("transaction rollback failed on panic", logger.ErrAttr(rbErr))
			}
			panic(p)
		} else if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				logger.Warn("transaction rollback failed on error", logger.ErrAttr(rbErr))
			}
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			logger.Warn("transaction rollback failed after commit error", logger.ErrAttr(rbErr))
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
