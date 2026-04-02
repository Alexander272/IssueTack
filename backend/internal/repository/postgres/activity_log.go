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
	query := fmt.Sprintf(`SELECT a.id, ticket_id, user_id, CONCAT_WS(' ', u.last_name, u.first_name) AS user_name, 
		type, old_value, new_value, a.created_at 
		FROM %s AS a INNER JOIN %s AS u ON a.user_id = u.id 
		WHERE ticket_id = $1 ORDER BY a.created_at DESC`,
		Tables.ActivityLog, Tables.Users,
	)

	var data []*models.ActivityLog
	rows, err := r.db.Query(ctx, query, req.TicketID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.ActivityLog{}
		if err := rows.Scan(
			&item.ID, &item.TicketID, &item.UserID, &item.UserName, &item.Type,
			&item.OldValue, &item.NewValue, &item.CreatedAt,
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
			v.TicketID,
			v.UserID,
			v.Type,
			v.OldValue,
			v.NewValue,
		}
	}

	columns := []string{"id", "ticket_id", "user_id", "type", "old_value", "new_value"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.ActivityLog},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return fmt.Errorf("failed to execute query. error: %w", err)
	}
	return nil
}
