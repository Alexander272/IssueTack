package migrate

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/migrate/postgres/migrations"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Migrate(pool *pgxpool.Pool) error {
	goose.SetBaseFS(&migrations.Content)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close() // Закрывает обертку, но НЕ сам пул pgx

	logger.Info("migration up till last")
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to migrate up: %w", err)
	}

	return nil
}
