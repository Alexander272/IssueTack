package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ===== Подписи ссылок на оборудовании =====

// fullNameSQL — выражение «ФИО пользователя или его username» для алиаса таблицы
// users. Одно и то же на всех выборках оборудования, истории и подписей событий.
func fullNameSQL(alias string) string {
	return fmt.Sprintf(`COALESCE(NULLIF(TRIM(%[1]s.last_name || ' ' || %[1]s.first_name), ''), %[1]s.username)`, alias)
}

// GetUserFullName возвращает ФИО пользователя (или username, если ФИО пусто).
// Нужен для снимка события: журнал иммутабелен, а FK объявлены ON DELETE SET NULL,
// поэтому подпись фиксируется в момент записи, а не джойном на чтении.
func (r *EquipmentRepo) GetUserFullName(ctx context.Context, userID uuid.UUID) (string, error) {
	// Алиас обязателен: fullNameSQL строит выражение по колонкам алиаса, а в
	// запросе без `u` Postgres отвечает 42P01 «missing FROM-clause entry».
	query := fmt.Sprintf(`SELECT %s FROM %s u WHERE u.id = $1`, fullNameSQL("u"), Tables.Users)

	var name *string
	if err := r.db.QueryRow(ctx, query, userID).Scan(&name); err != nil {
		return "", MapError(fmt.Errorf("failed to get user name: %w", err))
	}
	if name == nil {
		return "", nil
	}
	return *name, nil
}

// GetSiteName возвращает название площадки для снимка события.
func (r *EquipmentRepo) GetSiteName(ctx context.Context, siteID uuid.UUID) (string, error) {
	query := fmt.Sprintf(`SELECT name FROM %s WHERE id = $1`, Tables.Sites)

	var name *string
	if err := r.db.QueryRow(ctx, query, siteID).Scan(&name); err != nil {
		return "", MapError(fmt.Errorf("failed to get site name: %w", err))
	}
	if name == nil {
		return "", nil
	}
	return *name, nil
}
