package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DueDateRequests interface {
	// Create создаёт новый запрос на перенос срока (статус — pending).
	Create(ctx context.Context, tx Tx, req *models.DueDateRequest) error
	// GetByID возвращает запрос по идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*models.DueDateRequest, error)
	// GetPendingByTicket возвращает единственный активный (pending) запрос тикета.
	GetPendingByTicket(ctx context.Context, ticketID uuid.UUID) (*models.DueDateRequest, error)
	// Decide закрывает запрос: approved → новый срок утверждён, rejected → отклонён.
	// Выполняется только при status = 'pending'; иначе ErrDueDateRequestNotPending.
	Decide(ctx context.Context, tx Tx, id uuid.UUID, status string, decidedBy uuid.UUID, decidedAt time.Time, decisionComment string) (*models.DueDateRequest, error)
}

type dueDateRequestRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewDueDateRequestRepo(db *pgxpool.Pool, tr Transaction) *dueDateRequestRepository {
	return &dueDateRequestRepository{
		db:          db,
		Transaction: tr,
	}
}

func (r *dueDateRequestRepository) Create(ctx context.Context, tx Tx, req *models.DueDateRequest) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, ticket_id, requester_id, current_due_date, proposed_due_date, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW())
	`, "ticket_due_date_requests")

	_, err := r.getExec(tx).Exec(ctx, query,
		req.ID,
		req.TicketID,
		req.RequesterID,
		req.CurrentDueDate,
		req.ProposedDueDate,
		req.Reason,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

func (r *dueDateRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DueDateRequest, error) {
	query := fmt.Sprintf(`
		SELECT
			r.id, r.ticket_id, r.requester_id, r.current_due_date, r.proposed_due_date, r.reason, r.status,
			r.decided_by, r.decided_at, r.decision_comment, r.created_at,
			u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM ticket_due_date_requests r
		JOIN users u ON u.id = r.requester_id
		WHERE r.id = $1
	`)

	req := &models.DueDateRequest{}
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.TicketID, &req.RequesterID, &req.CurrentDueDate, &req.ProposedDueDate, &req.Reason, &req.Status,
		&req.DecidedBy, &req.DecidedAt, &req.DecisionComment, &req.CreatedAt,
		&req.Requester.ID, &req.Requester.Username, &req.Requester.FirstName, &req.Requester.LastName, &req.Requester.Email, &req.Requester.InternalNumber,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	return req, nil
}

func (r *dueDateRequestRepository) GetPendingByTicket(ctx context.Context, ticketID uuid.UUID) (*models.DueDateRequest, error) {
	query := fmt.Sprintf(`
		SELECT
			r.id, r.ticket_id, r.requester_id, r.current_due_date, r.proposed_due_date, r.reason, r.status,
			r.decided_by, r.decided_at, r.decision_comment, r.created_at,
			u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM ticket_due_date_requests r
		JOIN users u ON u.id = r.requester_id
		WHERE r.ticket_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC
		LIMIT 1
	`)

	req := &models.DueDateRequest{}
	if err := r.db.QueryRow(ctx, query, ticketID).Scan(
		&req.ID, &req.TicketID, &req.RequesterID, &req.CurrentDueDate, &req.ProposedDueDate, &req.Reason, &req.Status,
		&req.DecidedBy, &req.DecidedAt, &req.DecisionComment, &req.CreatedAt,
		&req.Requester.ID, &req.Requester.Username, &req.Requester.FirstName, &req.Requester.LastName, &req.Requester.Email, &req.Requester.InternalNumber,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	return req, nil
}

func (r *dueDateRequestRepository) Decide(ctx context.Context, tx Tx, id uuid.UUID, status string, decidedBy uuid.UUID, decidedAt time.Time, decisionComment string) (*models.DueDateRequest, error) {
	query := fmt.Sprintf(`
		UPDATE ticket_due_date_requests
		SET status = $2, decided_by = $3, decided_at = $4, decision_comment = $5
		WHERE id = $1 AND status = 'pending'
	`)

	res, err := r.getExec(tx).Exec(ctx, query, id, status, decidedBy, decidedAt, decisionComment)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, models.ErrDueDateRequestNotPending
	}

	query = fmt.Sprintf(`
		SELECT
			r.id, r.ticket_id, r.requester_id, r.current_due_date, r.proposed_due_date, r.reason, r.status,
			r.decided_by, r.decided_at, r.decision_comment, r.created_at,
			u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM ticket_due_date_requests r
		JOIN users u ON u.id = r.requester_id
		WHERE r.id = $1
	`)

	req := &models.DueDateRequest{}
	if err := r.getExec(tx).QueryRow(ctx, query, id).Scan(
		&req.ID, &req.TicketID, &req.RequesterID, &req.CurrentDueDate, &req.ProposedDueDate, &req.Reason, &req.Status,
		&req.DecidedBy, &req.DecidedAt, &req.DecisionComment, &req.CreatedAt,
		&req.Requester.ID, &req.Requester.Username, &req.Requester.FirstName, &req.Requester.LastName, &req.Requester.Email, &req.Requester.InternalNumber,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	return req, nil
}
