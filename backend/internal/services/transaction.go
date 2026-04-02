package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type TransactionManagerService struct {
	repo repository.Transaction
}

func NewTransactionManager(repo repository.Transaction) *TransactionManagerService {
	return &TransactionManagerService{repo: repo}
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(tx postgres.Tx) error) error
}

func (tm *TransactionManagerService) WithinTransaction(ctx context.Context, fn func(tx postgres.Tx) error) error {
	tx, err := tm.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
