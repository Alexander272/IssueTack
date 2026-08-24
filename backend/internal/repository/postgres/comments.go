package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewCommentRepo(db *pgxpool.Pool, tr Transaction) *CommentRepo {
	return &CommentRepo{
		db:          db,
		Transaction: tr,
	}
}

type Comments interface {
	GetByTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, showAllInternal bool) ([]*models.Comment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	Create(ctx context.Context, tx Tx, dto *models.Comment) error
	Delete(ctx context.Context, tx Tx, id uuid.UUID) error
}

func (r *CommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	query := fmt.Sprintf(`SELECT c.id, c.text, c.user_id, c.ticket_id, c.is_internal, c.created_at
		FROM %s c WHERE c.id = $1`,
		Tables.Comments,
	)

	item := &models.Comment{}
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.Text, &item.UserID, &item.TicketID, &item.IsInternal, &item.CreatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *CommentRepo) GetByTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, showAllInternal bool) ([]*models.Comment, error) {
	var query string
	if showAllInternal {
		query = fmt.Sprintf(`SELECT c.id, c.text, c.user_id, c.ticket_id, c.is_internal, c.created_at,
			u.id, u.username, u.first_name, u.last_name, u.internal_number
			FROM %s c
			LEFT JOIN %s u ON u.id = c.user_id
			WHERE c.ticket_id = $1
			ORDER BY c.created_at ASC`,
			Tables.Comments, Tables.Users,
		)
	} else {
		query = fmt.Sprintf(`SELECT c.id, c.text, c.user_id, c.ticket_id, c.is_internal, c.created_at,
			u.id, u.username, u.first_name, u.last_name, u.internal_number
			FROM %s c
			LEFT JOIN %s u ON u.id = c.user_id
			WHERE c.ticket_id = $1 AND (c.is_internal = FALSE OR c.user_id = $2)
			ORDER BY c.created_at ASC`,
			Tables.Comments, Tables.Users,
		)
	}

	var rows pgx.Rows
	var err error
	if showAllInternal {
		rows, err = r.db.Query(ctx, query, ticketID)
	} else {
		rows, err = r.db.Query(ctx, query, ticketID, userID)
	}
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Comment
	for rows.Next() {
		item := &models.Comment{User: &models.UserShort{}}
		if err := rows.Scan(
			&item.ID, &item.Text, &item.UserID, &item.TicketID, &item.IsInternal, &item.CreatedAt,
			&item.User.ID, &item.User.Username, &item.User.FirstName,
			&item.User.LastName, &item.User.InternalNumber,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.Comment{}, nil
	}
	return data, nil
}

func (r *CommentRepo) Create(ctx context.Context, tx Tx, dto *models.Comment) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, text, user_id, ticket_id, is_internal)
		VALUES ($1, $2, $3, $4, $5)`,
		Tables.Comments,
	)

	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.Text, dto.UserID, dto.TicketID, dto.IsInternal,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CommentRepo) Delete(ctx context.Context, tx Tx, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Comments)

	_, err := r.getExec(tx).Exec(ctx, query, id)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
