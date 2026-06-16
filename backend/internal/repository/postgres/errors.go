package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MapError конвертирует ошибки Postgres в доменные ошибки
func MapError(err error) error {
	if err == nil {
		return nil
	}

	// 1. Проверка на "запись не найдена"
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}

	// 2. Проверка специфических кодов Postgres (Unique, Foreign Key и т.д.)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "P0001" {
			if strings.Contains(pgErr.Message, "ERR_CIRCULAR") {
				return fmt.Errorf("%w: %v", models.ErrCircularInheritance, pgErr)
			}
			if strings.Contains(pgErr.Message, "ERR_REALM_CHANGE") {
				return fmt.Errorf("%w: %v", models.ErrChangeRealm, pgErr)
			}
		}

		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %v", models.ErrAlreadyExists, pgErr)
		case "23P01": // exclusion_violation
			return fmt.Errorf("%w: %v", models.ErrAlreadyExists, pgErr)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %v", models.ErrRelatedRecordNotFound, pgErr) // если ссылаемся на несуществующий ID
		case "23502": // not_null_violation
			return fmt.Errorf("%w: %v", models.ErrInvalidInput, pgErr)
		}
	}

	return err // возвращаем как есть, если не узнали (будет 500 ошибка)
}
