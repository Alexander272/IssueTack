package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// operationTaskSelectBase — общая выборка правила с подписями справочников.
// Используется и списком, и чтением по id, чтобы набор колонок не разъезжался.
func operationTaskSelectBase() string {
	return fmt.Sprintf(`SELECT
		t.id, t.realm_id, t.name, t.operation, t.action, t.title, t.description,
		t.category_id, t.group_id, t.priority, t.enabled, t.created_at, t.updated_at,
		COALESCE(c.name, '') AS category_name,
		COALESCE(g.name, '') AS group_name
	FROM %s t
	LEFT JOIN %s c ON c.id = t.category_id
	LEFT JOIN %s g ON g.id = t.group_id`,
		Tables.EquipmentOperationTasks, Tables.Categories, Tables.Groups)
}

func scanOperationTask(row pgx.Row) (*models.EquipmentOperationTask, error) {
	var task models.EquipmentOperationTask
	if err := row.Scan(
		&task.ID, &task.RealmID, &task.Name, &task.Operation, &task.Action,
		&task.Title, &task.Description, &task.CategoryID, &task.GroupID,
		&task.Priority, &task.Enabled, &task.CreatedAt, &task.UpdatedAt,
		&task.CategoryName, &task.GroupName,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, MapError(fmt.Errorf("failed to scan equipment operation task: %w", err))
	}
	return &task, nil
}

// GetOperationTasks возвращает правила реалма. operation != nil сужает список
// до правил одной операции (выдача/возврат).
func (r *EquipmentRepo) GetOperationTasks(ctx context.Context, realmID uuid.UUID, operation *models.EquipmentOperation) ([]*models.EquipmentOperationTask, error) {
	query := operationTaskSelectBase() + " WHERE t.realm_id = $1"
	args := []any{realmID}
	if operation != nil {
		query += " AND t.operation = $2"
		args = append(args, *operation)
	}
	query += " ORDER BY t.operation, t.name"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to query equipment operation tasks: %w", err))
	}
	defer rows.Close()

	tasks := []*models.EquipmentOperationTask{}
	for rows.Next() {
		task, err := scanOperationTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetOperationTaskByID читает правило по id в рамках реалма.
func (r *EquipmentRepo) GetOperationTaskByID(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentOperationTask, error) {
	query := operationTaskSelectBase() + " WHERE t.id = $1 AND t.realm_id = $2"
	return scanOperationTask(r.getExec(tx).QueryRow(ctx, query, id, realmID))
}

// CreateOperationTask создаёт правило и возвращает его с подписями справочников.
func (r *EquipmentRepo) CreateOperationTask(ctx context.Context, tx Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error) {
	query := fmt.Sprintf(`INSERT INTO %s (
		realm_id, name, operation, action, title, description,
		category_id, group_id, priority, enabled
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id`, Tables.EquipmentOperationTasks)

	var id uuid.UUID
	if err := r.getExec(tx).QueryRow(ctx, query,
		dto.RealmID, dto.Name, dto.Operation, dto.Action, dto.Title, dto.Description,
		dto.CategoryID, dto.GroupID, dto.Priority, dto.Enabled,
	).Scan(&id); err != nil {
		return nil, MapError(fmt.Errorf("failed to create equipment operation task: %w", err))
	}
	return r.GetOperationTaskByID(ctx, tx, id, dto.RealmID)
}

// UpdateOperationTask обновляет правило. RealmID в WHERE не даёт выйти за
// пределы реалма через подменённый id.
func (r *EquipmentRepo) UpdateOperationTask(ctx context.Context, tx Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error) {
	query := fmt.Sprintf(`UPDATE %s SET
		name = $2, operation = $3, action = $4, title = $5, description = $6,
		category_id = $7, group_id = $8, priority = $9, enabled = $10,
		updated_at = NOW()
		WHERE id = $1 AND realm_id = $11`, Tables.EquipmentOperationTasks)

	result, err := r.getExec(tx).Exec(ctx, query,
		*dto.ID, dto.Name, dto.Operation, dto.Action, dto.Title, dto.Description,
		dto.CategoryID, dto.GroupID, dto.Priority, dto.Enabled, dto.RealmID,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to update equipment operation task: %w", err))
	}
	if result.RowsAffected() == 0 {
		return nil, models.ErrNotFound
	}
	return r.GetOperationTaskByID(ctx, tx, *dto.ID, dto.RealmID)
}

// DeleteOperationTask удаляет правило реалма.
func (r *EquipmentRepo) DeleteOperationTask(ctx context.Context, dto *models.DelEquipmentOperationTaskDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentOperationTasks)

	result, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to delete equipment operation task: %w", err))
	}
	if result.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}
