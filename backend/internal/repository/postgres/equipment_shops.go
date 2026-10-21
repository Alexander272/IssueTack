package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// GetShops возвращает справочник магазинов реалма — варианты для поля «Магазин».
func (r *EquipmentRepo) GetShops(ctx context.Context, req *models.GetEquipmentShopsDTO) ([]*models.EquipmentShop, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, created_at FROM %s
		WHERE realm_id = $1 ORDER BY name`, Tables.EquipmentShops)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentShop{}
	for rows.Next() {
		item := &models.EquipmentShop{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.CreatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// EnsureShop добавляет название в справочник магазинов, если его там ещё нет.
// Вызывается при сохранении единицы: новое значение из формы (freeSolo) должно
// появиться в подсказках, не требуя ручного заведения. Upsert, а не проверка с
// вставкой: ручной ввод может обогнать другую форму, и гонка не должна падать
// (уникальность (realm_id, name) гасится ON CONFLICT). Пустое имя игнорируется.
func (r *EquipmentRepo) EnsureShop(ctx context.Context, tx Tx, realmID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	query := fmt.Sprintf(`INSERT INTO %s (realm_id, name) VALUES ($1, $2)
		ON CONFLICT (realm_id, name) DO NOTHING`, Tables.EquipmentShops)

	if _, err := r.getExec(tx).Exec(ctx, query, realmID, name); err != nil {
		return MapError(fmt.Errorf("failed to ensure shop: %w", err))
	}
	return nil
}
