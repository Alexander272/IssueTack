package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubtaskRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewSubtaskRepo(db *pgxpool.Pool, tr Transaction) *SubtaskRepo {
	return &SubtaskRepo{
		db:          db,
		Transaction: tr,
	}
}

type Subtasks interface {
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*models.Subtask, error)
	GetByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error)
	Create(ctx context.Context, tx Tx, dto *models.SubtaskDTO) error
	CreateSeveral(ctx context.Context, tx Tx, dto []*models.SubtaskDTO) error
	Update(ctx context.Context, tx Tx, dto *models.SubtaskDTO) error
	Delete(ctx context.Context, tx Tx, dto *models.DelSubtaskDTO) error
}

func (r *SubtaskRepo) GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*models.Subtask, error) {
	query := fmt.Sprintf(`SELECT 
			s.id, s.ticket_id, s.title, s.description, s.status, s.priority, s.due_date, s.closed_at, s.sort_order, s.created_at, s.updated_at,
			u.id, CONCAT_WS(' ', u.last_name, u.first_name)
		FROM %s s
		LEFT JOIN %s u ON s.assignee_id = u.id
		WHERE s.ticket_id = $1
		ORDER BY s.sort_order, s.created_at`,
		Tables.Subtasks, Tables.Users,
	)

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Subtask
	for rows.Next() {
		item := &models.Subtask{}
		var assigneeID *uuid.UUID
		var assigneeName *string
		if err := rows.Scan(
			&item.ID, &item.TicketID, &item.Title, &item.Description,
			&item.Status, &item.Priority, &item.DueDate, &item.ClosedAt,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
			&assigneeID, &assigneeName,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if assigneeID != nil {
			item.Assignee = &models.UserShort{ID: *assigneeID, FullName: *assigneeName}
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.Subtask{}, nil
	}
	return data, nil
}

func (r *SubtaskRepo) GetByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error) {
	query := fmt.Sprintf(`SELECT 
			s.id, s.ticket_id, s.title, s.description, s.status, s.priority, s.due_date, s.closed_at, s.sort_order, 
			s.created_at, s.updated_at,
			u.id, CONCAT_WS(' ', u.last_name, u.first_name)
		FROM %s s
		LEFT JOIN %s u ON s.assignee_id = u.id
		WHERE s.id = $1`,
		Tables.Subtasks, Tables.Users,
	)

	item := &models.Subtask{}
	var assigneeID *uuid.UUID
	var assigneeName *string
	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&item.ID, &item.TicketID, &item.Title, &item.Description,
		&item.Status, &item.Priority, &item.DueDate, &item.ClosedAt,
		&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
		&assigneeID, &assigneeName,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if assigneeID != nil {
		item.Assignee = &models.UserShort{ID: *assigneeID, FullName: *assigneeName}
	}
	return item, nil
}

func (r *SubtaskRepo) Create(ctx context.Context, tx Tx, dto *models.SubtaskDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, ticket_id, title, description, status, priority, assignee_id, due_date, sort_order) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		Tables.Subtasks,
	)
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.TicketID, dto.Title, dto.Description,
		dto.Status, dto.Priority, dto.AssigneeID, dto.DueDate, dto.SortOrder,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *SubtaskRepo) CreateSeveral(ctx context.Context, tx Tx, dto []*models.SubtaskDTO) error {
	if len(dto) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(dto))
	for i, v := range dto {
		if v.ID == uuid.Nil {
			v.ID = uuid.New()
		}
		rows[i] = []interface{}{
			v.ID, v.TicketID, v.Title, v.Description,
			v.Status, v.Priority, v.AssigneeID, v.DueDate, v.SortOrder,
		}
	}

	columns := []string{"id", "ticket_id", "title", "description", "status", "priority", "assignee_id", "due_date", "sort_order"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.Subtasks},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query. error: %w", err))
	}
	return nil
}

func (r *SubtaskRepo) Update(ctx context.Context, tx Tx, dto *models.SubtaskDTO) error {
	query := fmt.Sprintf(`UPDATE %s 
		SET title=$3, description=$4, status=$5, priority=$6, assignee_id=$7, due_date=$8, sort_order=$9, updated_at=NOW()
		WHERE id=$1 AND ticket_id=$2`,
		Tables.Subtasks,
	)

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.TicketID, dto.Title, dto.Description,
		dto.Status, dto.Priority, dto.AssigneeID, dto.DueDate, dto.SortOrder,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *SubtaskRepo) Delete(ctx context.Context, tx Tx, dto *models.DelSubtaskDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Subtasks)

	_, err := r.getExec(tx).Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
