package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DueDateRequests — доступ к запросам на перенос срока заявки. Статусы —
// models.DueDateRequestStatus*: их пишут и сервис, и репозиторий.
type DueDateRequests interface {
	// Create создаёт новый запрос на перенос срока (статус — pending).
	Create(ctx context.Context, tx Tx, req *models.DueDateRequest) error
	// GetByID возвращает запрос по идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*models.DueDateRequest, error)
	// GetPendingByTicket возвращает единственный активный (pending) запрос тикета;
	// nil без ошибки, если активного запроса нет.
	GetPendingByTicket(ctx context.Context, ticketID uuid.UUID) (*models.DueDateRequest, error)
	// Decide закрывает запрос: approved → новый срок утверждён, rejected → отклонён.
	// Выполняется только при status = 'pending'; иначе ErrDueDateRequestNotPending.
	Decide(ctx context.Context, tx Tx, req *models.DueDateRequest) (*models.DueDateRequest, error)
}

type DueDateRequestRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewDueDateRequestRepo(db *pgxpool.Pool, tr Transaction) *DueDateRequestRepo {
	return &DueDateRequestRepo{
		db:          db,
		Transaction: tr,
	}
}

// dueDateRequestSelect — общий SELECT строки запроса вместе с заявителем. Источник
// передаётся параметром: обычная таблица или CTE (в Decide — результат UPDATE).
// Так список колонок и порядок Scan живут в одном месте.
func dueDateRequestSelect(source string) string {
	return fmt.Sprintf(`
		SELECT
			r.id, r.ticket_id, r.requester_id, r.current_due_date, r.proposed_due_date, r.reason, r.status,
			r.decided_by, r.decided_at, r.decision_comment, r.created_at,
			u.id, u.username, u.first_name, u.last_name, u.email, u.internal_number
		FROM %s r
		JOIN %s u ON u.id = r.requester_id`,
		source, Tables.Users,
	)
}

// scanDueDateRequest читает строку dueDateRequestSelect в модель.
func scanDueDateRequest(row pgx.Row) (*models.DueDateRequest, error) {
	req := &models.DueDateRequest{
		Requester: &models.UserShort{},
	}
	if err := row.Scan(
		&req.ID, &req.TicketID, &req.RequesterID, &req.CurrentDueDate, &req.ProposedDueDate, &req.Reason, &req.Status,
		&req.DecidedBy, &req.DecidedAt, &req.DecisionComment, &req.CreatedAt,
		&req.Requester.ID, &req.Requester.Username, &req.Requester.FirstName, &req.Requester.LastName, &req.Requester.Email, &req.Requester.InternalNumber,
	); err != nil {
		return nil, err
	}
	return req, nil
}

func (r *DueDateRequestRepo) Create(ctx context.Context, tx Tx, req *models.DueDateRequest) error {
	// Заполняем служебные поля, если их не проставил вызывающий: возвращённая
	// модель должна совпадать с тем, что легло в БД.
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	if req.Status == "" {
		req.Status = models.DueDateRequestStatusPending
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, ticket_id, requester_id, current_due_date, proposed_due_date, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		Tables.TicketDueDateRequests,
	)

	if _, err := r.getExec(tx).Exec(ctx, query,
		req.ID,
		req.TicketID,
		req.RequesterID,
		req.CurrentDueDate,
		req.ProposedDueDate,
		req.Reason,
		req.Status,
		req.CreatedAt,
	); err != nil {
		// Один pending на тикет держит частичный уникальный индекс. Гонка двух
		// одновременных запросов ловится здесь, поэтому unique_violation — это не
		// общий «уже существует», а доменный ErrDueDateRequestAlreadyPending.
		mapped := MapError(fmt.Errorf("failed to execute query: %w", err))
		if errors.Is(mapped, models.ErrAlreadyExists) {
			return models.ErrDueDateRequestAlreadyPending
		}
		return mapped
	}
	return nil
}

func (r *DueDateRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DueDateRequest, error) {
	query := dueDateRequestSelect(Tables.TicketDueDateRequests) + ` WHERE r.id = $1`

	req, err := scanDueDateRequest(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return req, nil
}

func (r *DueDateRequestRepo) GetPendingByTicket(ctx context.Context, ticketID uuid.UUID) (*models.DueDateRequest, error) {
	// ORDER BY/LIMIT страхуют от легаси-дублей, хотя уникальный индекс уже
	// гарантирует один pending на тикет.
	query := dueDateRequestSelect(Tables.TicketDueDateRequests) + `
		WHERE r.ticket_id = $1 AND r.status = $2
		ORDER BY r.created_at DESC
		LIMIT 1`

	req, err := scanDueDateRequest(r.db.QueryRow(ctx, query, ticketID, models.DueDateRequestStatusPending))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return req, nil
}

// Decide закрывает pending-запрос и возвращает обновлённую строку. UPDATE и чтение
// объединены одним запросом (CTE): решение и его результат атомарны, а отсутствие
// строки означает, что запрос уже обработан или не существует.
func (r *DueDateRequestRepo) Decide(ctx context.Context, tx Tx, req *models.DueDateRequest) (*models.DueDateRequest, error) {
	// Решение закрывает запрос: любое другое значение статуса записало бы в БД
	// состояние, из которого запрос уже не выйдет (в колонке нет CHECK).
	switch req.Status {
	case models.DueDateRequestStatusApproved, models.DueDateRequestStatusRejected:
	default:
		return nil, models.ErrInvalidInput
	}

	query := fmt.Sprintf(`
		WITH decided AS (
			UPDATE %s
			SET status = $2, decided_by = $3, decided_at = $4, decision_comment = $5
			WHERE id = $1 AND status = $6
			RETURNING *
		)`, Tables.TicketDueDateRequests) + dueDateRequestSelect("decided")

	updated, err := scanDueDateRequest(r.getExec(tx).QueryRow(ctx, query,
		req.ID,
		req.Status,
		req.DecidedBy,
		req.DecidedAt,
		req.DecisionComment,
		models.DueDateRequestStatusPending,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrDueDateRequestNotPending
	}
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return updated, nil
}
