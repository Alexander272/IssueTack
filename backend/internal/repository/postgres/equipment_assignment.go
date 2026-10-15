package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// ===== Выдачи =====

func scanAssignment(row rowScanner) (*models.EquipmentAssignment, error) {
	a := &models.EquipmentAssignment{}
	if err := row.Scan(&a.ID, &a.EquipmentID, &a.UserID, &a.HolderName, &a.SiteID, &a.Location,
		&a.AssignedAt, &a.UnassignedAt, &a.AssignedBy, &a.Comment); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *EquipmentRepo) GetOpenAssignment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error) {
	query := fmt.Sprintf(`SELECT id, equipment_id, user_id, holder_name, site_id, location, assigned_at, unassigned_at, assigned_by, comment
		FROM %s WHERE equipment_id = $1 AND unassigned_at IS NULL ORDER BY assigned_at DESC LIMIT 1`, Tables.EquipmentAssignments)

	a, err := scanAssignment(r.db.QueryRow(ctx, query, equipmentID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return a, nil
}

func (r *EquipmentRepo) GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error) {
	query := fmt.Sprintf(`SELECT a.id, a.equipment_id, a.user_id, a.holder_name, a.site_id, a.location,
			a.assigned_at, a.unassigned_at, a.assigned_by, a.comment
		FROM %s a JOIN %s e ON e.id = a.equipment_id
		WHERE a.equipment_id = $1 AND e.realm_id = $2 ORDER BY a.assigned_at DESC`,
		Tables.EquipmentAssignments, Tables.Equipment)

	rows, err := r.db.Query(ctx, query, equipmentID, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentAssignment{}
	for rows.Next() {
		item, err := scanAssignment(rows)
		if err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *EquipmentRepo) CreateAssignment(ctx context.Context, tx Tx, a *models.EquipmentAssignment) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, equipment_id, user_id, holder_name, site_id, location, assigned_at, unassigned_at, assigned_by, comment)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, $8, $9, $10)`, Tables.EquipmentAssignments)

	_, err := r.getExec(tx).Exec(ctx, query, a.ID, a.EquipmentID, a.UserID, a.HolderName, a.SiteID, a.Location,
		a.AssignedAt, a.UnassignedAt, a.AssignedBy, a.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseAssignment(ctx context.Context, tx Tx, id uuid.UUID, at time.Time, comment string) error {
	query := fmt.Sprintf(`UPDATE %s SET unassigned_at = $2, comment = CASE WHEN $3 = '' THEN comment ELSE $3 END WHERE id = $1`, Tables.EquipmentAssignments)

	_, err := r.getExec(tx).Exec(ctx, query, id, at, comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// ===== События =====

func (r *EquipmentRepo) CreateEvent(ctx context.Context, tx Tx, e *models.EquipmentEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, equipment_id, event_type, operation_id, component_id, old_value, new_value, user_id, comment)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8, $9)`, Tables.EquipmentEvents)

	_, err := r.getExec(tx).Exec(ctx, query, e.ID, e.EquipmentID, e.EventType, e.OperationID, e.ComponentID,
		nullableJSON(e.OldValue), nullableJSON(e.NewValue), e.UserID, e.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error) {
	query := fmt.Sprintf(`SELECT ev.id, ev.equipment_id, ev.event_type, ev.operation_id, ev.component_id,
			ev.old_value, ev.new_value, ev.user_id, ev.comment, ev.created_at
		FROM %s ev JOIN %s e ON e.id = ev.equipment_id
		WHERE ev.equipment_id = $1 AND e.realm_id = $2 ORDER BY ev.created_at DESC`,
		Tables.EquipmentEvents, Tables.Equipment)

	args := []any{equipmentID, realmID}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $4"
			args = append(args, offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentEvent{}
	for rows.Next() {
		item := &models.EquipmentEvent{}
		if err := rows.Scan(&item.ID, &item.EquipmentID, &item.EventType, &item.OperationID, &item.ComponentID,
			&item.OldValue, &item.NewValue, &item.UserID, &item.Comment, &item.CreatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// nullableJSON возвращает nil для пустого jsonb, иначе строку.
func nullableJSON(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return string(v)
}
