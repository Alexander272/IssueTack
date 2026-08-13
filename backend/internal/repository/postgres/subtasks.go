package postgres

import (
	"context"
	"fmt"
	"strings"

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
			u.id, u.username, u.first_name, u.last_name, u.internal_number
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
		var assigneeUsername, assigneeFirstName, assigneeLastName *string
		var assigneeInternalNumber *string
		if err := rows.Scan(
			&item.ID, &item.TicketID, &item.Title, &item.Description,
			&item.Status, &item.Priority, &item.DueDate, &item.ClosedAt,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
			&assigneeID, &assigneeUsername, &assigneeFirstName, &assigneeLastName, &assigneeInternalNumber,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if assigneeID != nil {
			item.Assignee = &models.UserShort{ID: *assigneeID, Username: *assigneeUsername, FirstName: *assigneeFirstName, LastName: *assigneeLastName, InternalNumber: *assigneeInternalNumber}
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
			u.id, u.username, u.first_name, u.last_name, u.internal_number
		FROM %s s
		LEFT JOIN %s u ON s.assignee_id = u.id
		WHERE s.id = $1`,
		Tables.Subtasks, Tables.Users,
	)

	item := &models.Subtask{}
	var assigneeID *uuid.UUID
	var assigneeUsername, assigneeFirstName, assigneeLastName *string
	var assigneeInternalNumber *string
	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&item.ID, &item.TicketID, &item.Title, &item.Description,
		&item.Status, &item.Priority, &item.DueDate, &item.ClosedAt,
		&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
		&assigneeID, &assigneeUsername, &assigneeFirstName, &assigneeLastName, &assigneeInternalNumber,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if assigneeID != nil {
		item.Assignee = &models.UserShort{ID: *assigneeID, Username: *assigneeUsername, FirstName: *assigneeFirstName, LastName: *assigneeLastName, InternalNumber: *assigneeInternalNumber}
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
	sets := make([]string, 0, 8)
	args := []interface{}{dto.ID, dto.TicketID}
	n := 2

	add := func(jsonKey, column string, value interface{}) {
		if !dto.HasField(jsonKey) {
			return
		}
		n++
		sets = append(sets, fmt.Sprintf("%s=$%d", column, n))
		args = append(args, value)
	}

	add("title", "title", dto.Title)
	add("description", "description", dto.Description)
	add("status", "status", dto.Status)
	add("priority", "priority", dto.Priority)
	add("assigneeId", "assignee_id", dto.AssigneeID)
	add("dueDate", "due_date", dto.DueDate)
	add("sortOrder", "sort_order", dto.SortOrder)

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE %s SET %s, updated_at=NOW() WHERE id=$1 AND ticket_id=$2`,
		Tables.Subtasks, strings.Join(sets, ", "),
	)

	_, err := r.getExec(tx).Exec(ctx, query, args...)
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
