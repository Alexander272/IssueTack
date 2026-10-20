package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

func (r *EquipmentRepo) GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, kind, icon, sort_order, created_at, updated_at
		FROM %s WHERE realm_id = $1 ORDER BY sort_order, name`, Tables.EquipmentTypes)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentType{}
	for rows.Next() {
		item := &models.EquipmentType{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.Kind, &item.Icon,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *EquipmentRepo) GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, kind, icon, sort_order, created_at, updated_at
		FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentTypes)

	item := &models.EquipmentType{}
	if err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(
		&item.ID, &item.RealmID, &item.Name, &item.Kind, &item.Icon, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *EquipmentRepo) CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	id := uuid.New()
	dto.ID = &id

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, kind, icon, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)`, Tables.EquipmentTypes)

	_, err := r.db.Exec(ctx, query, id, dto.RealmID, dto.Name, dto.Kind, dto.Icon, dto.SortOrder)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, kind=$3, icon=$4, sort_order=$5, updated_at=NOW()
		WHERE id=$1 AND realm_id=$6`, Tables.EquipmentTypes)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Kind, dto.Icon, dto.SortOrder, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *EquipmentRepo) DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentTypes)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// CountByType возвращает число единиц типа в реалме (для гарда удаления типа).
func (r *EquipmentRepo) CountByType(ctx context.Context, realmID, typeID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE realm_id = $1 AND type_id = $2`, Tables.Equipment)

	var count int
	if err := r.db.QueryRow(ctx, query, realmID, typeID).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return count, nil
}
