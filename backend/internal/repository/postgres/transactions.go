package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{
		db: db,
	}
}

type Transaction interface {
	BeginTx(ctx context.Context) (Tx, error)
	getExec(tx Tx) QueryExecutor
}

type Tx interface {
	TX() pgx.Tx
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Результирующий интерфейс для всех SQL-операций
type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// Метод для начала транзакции
func (r *TransactionRepo) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction. error: %w", err)
	}
	return &repoTx{Tx: tx}, nil
}

func (r *TransactionRepo) getExec(tx Tx) QueryExecutor {
	if tx != nil {
		return tx.TX()
	}
	return r.db
}

type repoTx struct {
	Tx pgx.Tx
}

func (t *repoTx) TX() pgx.Tx {
	return t.Tx
}

func (t *repoTx) Commit(ctx context.Context) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction. error: %w", err)
	}
	return nil
}

func (t *repoTx) Rollback(ctx context.Context) error {
	if err := t.Tx.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction. error: %w", err)
	}
	return nil
}
