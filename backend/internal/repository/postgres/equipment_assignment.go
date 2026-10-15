package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// ===== Выдачи =====

// assignmentSelectBase — выдача со справочными подписями: ФИО держателя (или
// свободный текст holder_name), название площадки и ФИО того, кто выдал. Подписи
// резолвит SQL — как equipmentSelectBase, чтобы клиенту не приходилось тянуть
// справочники реалма ради строки истории.
func assignmentSelectBase() string {
	return fmt.Sprintf(`SELECT a.id, a.equipment_id, a.user_id, a.holder_name, a.site_id, a.location,
			a.assigned_at, a.unassigned_at, a.assigned_by, a.comment,
			%s AS holder_full_name, s.name AS site_name, %s AS assigned_by_name
		FROM %s a
		LEFT JOIN %s hu ON hu.id = a.user_id
		LEFT JOIN %s s ON s.id = a.site_id
		LEFT JOIN %s ab ON ab.id = a.assigned_by`,
		fullNameSQL("hu"), fullNameSQL("ab"),
		Tables.EquipmentAssignments, Tables.Users, Tables.Sites, Tables.Users)
}

func scanAssignment(row rowScanner) (*models.EquipmentAssignment, error) {
	a := &models.EquipmentAssignment{}
	var holderFullName, siteName, assignedByName *string

	if err := row.Scan(&a.ID, &a.EquipmentID, &a.UserID, &a.HolderName, &a.SiteID, &a.Location,
		&a.AssignedAt, &a.UnassignedAt, &a.AssignedBy, &a.Comment,
		&holderFullName, &siteName, &assignedByName); err != nil {
		return nil, err
	}
	if holderFullName != nil {
		a.HolderFullName = *holderFullName
	}
	if siteName != nil {
		a.SiteName = *siteName
	}
	if assignedByName != nil {
		a.AssignedByName = *assignedByName
	}
	return a, nil
}

func (r *EquipmentRepo) GetOpenAssignment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error) {
	query := assignmentSelectBase() +
		` WHERE a.equipment_id = $1 AND a.unassigned_at IS NULL ORDER BY a.assigned_at DESC LIMIT 1`

	a, err := scanAssignment(r.db.QueryRow(ctx, query, equipmentID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return a, nil
}

// GetLastStoredPlace отдаёт место последнего «складского» периода единицы — строки
// без держателя. Возврат ставит единицу именно туда: держатель снимается, а место
// берётся не у него, а из периода, когда единица лежала на хранении (место —
// реквизит единицы, оно не приезжает и не уезжает вместе с держателем). Строка
// ищется и открытая, и закрытая: если единица прямо сейчас лежит на складе, её
// место и есть домашнее. Если складского периода не было (единицу завели сразу на
// держателя), возвращается ErrNotFound — вызывающий сам решает, какое место
// поставить.
func (r *EquipmentRepo) GetLastStoredPlace(ctx context.Context, equipmentID uuid.UUID) (*uuid.UUID, *string, error) {
	query := fmt.Sprintf(`SELECT site_id, location FROM %s
		WHERE equipment_id = $1 AND user_id IS NULL AND COALESCE(holder_name, '') = ''
		ORDER BY assigned_at DESC LIMIT 1`, Tables.EquipmentAssignments)

	var siteID *uuid.UUID
	var location *string
	if err := r.db.QueryRow(ctx, query, equipmentID).Scan(&siteID, &location); err != nil {
		return nil, nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return siteID, location, nil
}

func (r *EquipmentRepo) GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error) {
	query := assignmentSelectBase() + fmt.Sprintf(` JOIN %s e ON e.id = a.equipment_id
		WHERE a.equipment_id = $1 AND e.realm_id = $2 ORDER BY a.assigned_at DESC`, Tables.Equipment)

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
			ev.old_value, ev.new_value, ev.user_id, ev.comment, ev.created_at,
			%s AS user_name
		FROM %s ev JOIN %s e ON e.id = ev.equipment_id
		LEFT JOIN %s u ON u.id = ev.user_id
		WHERE ev.equipment_id = $1 AND e.realm_id = $2 ORDER BY ev.created_at DESC`,
		fullNameSQL("u"), Tables.EquipmentEvents, Tables.Equipment, Tables.Users)

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
		var userName *string
		if err := rows.Scan(&item.ID, &item.EquipmentID, &item.EventType, &item.OperationID, &item.ComponentID,
			&item.OldValue, &item.NewValue, &item.UserID, &item.Comment, &item.CreatedAt, &userName); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if userName != nil {
			item.UserName = *userName
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
