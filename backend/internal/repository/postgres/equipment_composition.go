package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// ===== Состав логического комплекта =====

func (r *EquipmentRepo) GetOpenComposition(ctx context.Context, kitID uuid.UUID) ([]*models.Equipment, error) {
	query := equipmentSelectBase() + fmt.Sprintf(` JOIN %s kc ON kc.component_id = e.id AND kc.ended_at IS NULL
		WHERE kc.kit_id = $1 ORDER BY e.name`, Tables.KitCompositions)

	rows, err := r.db.Query(ctx, query, kitID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data, _, err := scanEquipmentRows(rows)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// compositionSelectBase — запись состава со справочными подписями: имя компонента
// и ФИО того, кто добавил/исключил. Подписи резолвит SQL — как в equipmentSelectBase.
func compositionSelectBase() string {
	return fmt.Sprintf(`SELECT kc.id, kc.kit_id, kc.component_id, kc.started_at, kc.ended_at,
			kc.started_by, kc.ended_by, kc.reason, kc.act_id,
			c.name AS component_name,
			%s AS started_by_name,
			%s AS ended_by_name
		FROM %s kc
		LEFT JOIN %s c ON c.id = kc.component_id
		LEFT JOIN %s sb ON sb.id = kc.started_by
		LEFT JOIN %s eb ON eb.id = kc.ended_by`,
		fullNameSQL("sb"), fullNameSQL("eb"),
		Tables.KitCompositions, Tables.Equipment, Tables.Users, Tables.Users)
}

func scanComposition(row rowScanner) (*models.KitComposition, error) {
	c := &models.KitComposition{}
	var componentName, startedByName, endedByName *string

	if err := row.Scan(&c.ID, &c.KitID, &c.ComponentID, &c.StartedAt, &c.EndedAt, &c.StartedBy, &c.EndedBy,
		&c.Reason, &c.ActID, &componentName, &startedByName, &endedByName); err != nil {
		return nil, err
	}
	if componentName != nil {
		c.ComponentName = *componentName
	}
	if startedByName != nil {
		c.StartedByName = *startedByName
	}
	if endedByName != nil {
		c.EndedByName = *endedByName
	}
	return c, nil
}

func (r *EquipmentRepo) GetCompositionHistory(ctx context.Context, kitID uuid.UUID) ([]*models.KitComposition, error) {
	query := compositionSelectBase() + ` WHERE kc.kit_id = $1 ORDER BY kc.started_at DESC`

	rows, err := r.db.Query(ctx, query, kitID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.KitComposition{}
	for rows.Next() {
		item, err := scanComposition(rows)
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

func (r *EquipmentRepo) GetOpenByComponent(ctx context.Context, componentID uuid.UUID) (*models.KitComposition, error) {
	query := compositionSelectBase() +
		` WHERE kc.component_id = $1 AND kc.ended_at IS NULL ORDER BY kc.started_at DESC LIMIT 1`

	c, err := scanComposition(r.db.QueryRow(ctx, query, componentID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return c, nil
}

func (r *EquipmentRepo) OpenComposition(ctx context.Context, tx Tx, c *models.KitComposition) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	insert := fmt.Sprintf(`INSERT INTO %s (id, kit_id, component_id, started_at, started_by, reason, act_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, insert, c.ID, c.KitID, c.ComponentID, c.StartedAt, c.StartedBy, c.Reason, c.ActID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = $2 WHERE id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, c.ComponentID, c.KitID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseComposition(ctx context.Context, tx Tx, c *models.KitComposition) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = $2, ended_by = $3, reason = CASE WHEN $4 = '' THEN reason ELSE $4 END, act_id = COALESCE($5, act_id)
		WHERE id = $1`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, c.ID, c.EndedAt, c.EndedBy, c.Reason, c.ActID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseOpenByComponent(ctx context.Context, tx Tx, componentID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = NOW(), ended_by = $2, reason = CASE WHEN $3 = '' THEN reason ELSE $3 END, act_id = COALESCE($4, act_id)
		WHERE component_id = $1 AND ended_at IS NULL`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, componentID, endedBy, reason, actID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = NULL WHERE id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, componentID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseOpenByKit(ctx context.Context, tx Tx, kitID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = NOW(), ended_by = $2, reason = CASE WHEN $3 = '' THEN reason ELSE $3 END, act_id = COALESCE($4, act_id)
		WHERE kit_id = $1 AND ended_at IS NULL`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, kitID, endedBy, reason, actID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = NULL WHERE kit_id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, kitID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
