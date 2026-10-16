package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// ===== Шаблоны комплектов =====

// GetKitTemplates возвращает шаблоны реалма вместе с позициями. Позиции грузятся
// одним запросом на весь список (по массиву id), чтобы не делать N+1.
func (r *EquipmentRepo) GetKitTemplates(ctx context.Context, realmID uuid.UUID) ([]*models.EquipmentKitTemplate, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, description, sort_order, created_at, updated_at
		FROM %s WHERE realm_id = $1 ORDER BY sort_order, name`, Tables.EquipmentKitTemplates)

	rows, err := r.db.Query(ctx, query, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentKitTemplate{}
	ids := []uuid.UUID{}
	for rows.Next() {
		item := &models.EquipmentKitTemplate{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.Description,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if len(ids) == 0 {
		return data, nil
	}

	items, err := r.getKitTemplateItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, t := range data {
		if list, ok := items[t.ID]; ok {
			t.Items = list
		} else {
			t.Items = []*models.EquipmentKitTemplateItem{}
		}
	}
	return data, nil
}

func (r *EquipmentRepo) GetKitTemplateByID(ctx context.Context, req *models.GetEquipmentKitTemplateByIdDTO) (*models.EquipmentKitTemplate, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, description, sort_order, created_at, updated_at
		FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentKitTemplates)

	item := &models.EquipmentKitTemplate{}
	if err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(&item.ID, &item.RealmID,
		&item.Name, &item.Description, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	items, err := r.getKitTemplateItems(ctx, []uuid.UUID{item.ID})
	if err != nil {
		return nil, err
	}
	item.Items = items[item.ID]
	if item.Items == nil {
		item.Items = []*models.EquipmentKitTemplateItem{}
	}
	return item, nil
}

// getKitTemplateItems возвращает позиции шаблонов, сгруппированные по шаблону.
// TypeName/TypeKind берутся джойном: клиенту не нужен отдельный запрос справочника.
func (r *EquipmentRepo) getKitTemplateItems(ctx context.Context, templateIDs []uuid.UUID) (map[uuid.UUID][]*models.EquipmentKitTemplateItem, error) {
	query := fmt.Sprintf(`SELECT i.id, i.template_id, i.type_id, t.name, t.kind, i.sort_order
		FROM %s i JOIN %s t ON t.id = i.type_id
		WHERE i.template_id = ANY($1) ORDER BY i.sort_order, t.name`,
		Tables.EquipmentKitTemplateItems, Tables.EquipmentTypes)

	rows, err := r.db.Query(ctx, query, templateIDs)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := map[uuid.UUID][]*models.EquipmentKitTemplateItem{}
	for rows.Next() {
		item := &models.EquipmentKitTemplateItem{}
		if err := rows.Scan(&item.ID, &item.TemplateID, &item.TypeID, &item.TypeName,
			&item.TypeKind, &item.SortOrder); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data[item.TemplateID] = append(data[item.TemplateID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// CreateKitTemplate создаёт шаблон и его позиции. Позиции пишутся в той же
// транзакции: частично сохранённый набор слотов не нужен.
func (r *EquipmentRepo) CreateKitTemplate(ctx context.Context, tx Tx, dto *models.EquipmentKitTemplateDTO) error {
	id := uuid.New()
	dto.ID = &id

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, description, sort_order)
		VALUES ($1, $2, $3, $4, $5)`, Tables.EquipmentKitTemplates)

	if _, err := r.getExec(tx).Exec(ctx, query, id, dto.RealmID, dto.Name, dto.Description, dto.SortOrder); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	return r.replaceKitTemplateItems(ctx, tx, id, dto.Items)
}

// UpdateKitTemplate обновляет шаблон и заменяет состав присланным: отдельных
// операций над позицией нет. nil-состав оставляет позиции как есть.
func (r *EquipmentRepo) UpdateKitTemplate(ctx context.Context, tx Tx, dto *models.EquipmentKitTemplateDTO) error {
	if dto.ID == nil {
		return models.ErrInvalidInput
	}

	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, sort_order=$4, updated_at=NOW()
		WHERE id=$1 AND realm_id=$5`, Tables.EquipmentKitTemplates)

	res, err := r.getExec(tx).Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.SortOrder, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}

	// nil-состав — «позиции не трогать»; явный пустой массив очищает состав.
	if dto.Items == nil {
		return nil
	}
	return r.replaceKitTemplateItems(ctx, tx, *dto.ID, dto.Items)
}

// replaceKitTemplateItems удаляет прежние позиции и вставляет присланные.
func (r *EquipmentRepo) replaceKitTemplateItems(ctx context.Context, tx Tx, templateID uuid.UUID, items []models.EquipmentKitTemplateItemDTO) error {
	del := fmt.Sprintf(`DELETE FROM %s WHERE template_id = $1`, Tables.EquipmentKitTemplateItems)
	if _, err := r.getExec(tx).Exec(ctx, del, templateID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	if len(items) == 0 {
		return nil
	}

	insert := fmt.Sprintf(`INSERT INTO %s (id, template_id, type_id, sort_order)
		VALUES ($1, $2, $3, $4)`, Tables.EquipmentKitTemplateItems)

	exec := r.getExec(tx)
	for i, item := range items {
		order := item.SortOrder
		if order == 0 {
			order = i
		}
		if _, err := exec.Exec(ctx, insert, uuid.New(), templateID, item.TypeID, order); err != nil {
			return MapError(fmt.Errorf("failed to execute query: %w", err))
		}
	}
	return nil
}

func (r *EquipmentRepo) DeleteKitTemplate(ctx context.Context, dto *models.DelEquipmentKitTemplateDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentKitTemplates)

	if _, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
