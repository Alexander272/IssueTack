package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type activityRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewActivityRepo(db *pgxpool.Pool, tr Transaction) *activityRepository {
	return &activityRepository{
		db:          db,
		Transaction: tr,
	}
}

type ActivityLog interface {
	Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error)
	Create(ctx context.Context, tx Tx, dto []*models.ActivityLogDTO) error
}

func (r *activityRepository) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	where := ""
	args := make([]any, 0)

	if req.ParentID != nil {
		where = "WHERE entity_id = $1 OR parent_id = $1"
		args = append(args, *req.ParentID)
	} else if req.EntityID != nil {
		where = "WHERE entity_id = $1"
		args = append(args, *req.EntityID)
		if req.EntityType != nil {
			where += fmt.Sprintf(" AND entity_type = $%d", len(args)+1)
			args = append(args, *req.EntityType)
		}
	}

	if req.RealmID != nil {
		if where == "" {
			where = "WHERE realm_id = $1"
			args = append(args, *req.RealmID)
		} else {
			where += fmt.Sprintf(" AND realm_id = $%d", len(args)+1)
			args = append(args, *req.RealmID)
		}
	}

	query := fmt.Sprintf(`SELECT id, action, changed_by, changed_by_name, entity_type, entity_id, entity, parent_id, realm_id, realm_name, old_value, new_value, created_at 
		FROM %s %s ORDER BY created_at DESC`,
		Tables.ActivityLog, where,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.ActivityLog
	for rows.Next() {
		item := &models.ActivityLog{}
		if err := rows.Scan(
			&item.ID, &item.Action, &item.ChangedBy, &item.ChangedByName,
			&item.EntityType, &item.EntityID, &item.Entity, &item.ParentID,
			&item.RealmID, &item.RealmName,
			&item.OldValues, &item.NewValues, &item.CreatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *activityRepository) Create(ctx context.Context, tx Tx, dto []*models.ActivityLogDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(dto))

	for i, v := range dto {
		if v.ID == uuid.Nil {
			v.ID = uuid.New()
		}

		rows[i] = []interface{}{
			v.ID,
			v.Action,
			v.ChangedBy,
			v.ChangedByName,
			v.EntityType,
			v.EntityID,
			v.Entity,
			v.ParentID,
			v.RealmID,
			v.RealmName,
			v.OldValues,
			v.NewValues,
		}
	}

	columns := []string{"id", "action", "changed_by", "changed_by_name", "entity_type", "entity_id", "entity", "parent_id", "realm_id", "realm_name", "old_value", "new_value"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.ActivityLog},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return MapError(fmt.Errorf("failed to execute query. error: %w", err))
	}
	return nil
}
