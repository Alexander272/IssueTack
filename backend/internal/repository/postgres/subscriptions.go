package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type subscriptionRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewSubscriptionRepo(db *pgxpool.Pool, tr Transaction) *subscriptionRepository {
	return &subscriptionRepository{
		db:          db,
		Transaction: tr,
	}
}

type TicketSubscriptions interface {
	Subscribe(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID) error
	Unsubscribe(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID) error
	Exists(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID) (bool, error)
	GetByTicket(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error)
	// GetSubscribersByEvent возвращает подписанных на тикет пользователей, у которых включён
	// мастер-переключатель уведомлений и событие eventField (в настройках категории тикета)
	// для категории categoryID. categoryID должен быть non-nil.
	GetSubscribersByEvent(ctx context.Context, ticketID, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error)
}

func (r *subscriptionRepository) Subscribe(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (ticket_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (ticket_id, user_id) DO NOTHING`, Tables.TicketSubscriptions)

	_, err := r.getExec(tx).Exec(ctx, query, ticketID, userID)
	if err != nil {
		return MapError(fmt.Errorf("failed to subscribe: %w", err))
	}
	return nil
}

func (r *subscriptionRepository) Unsubscribe(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE ticket_id = $1 AND user_id = $2`, Tables.TicketSubscriptions)

	_, err := r.getExec(tx).Exec(ctx, query, ticketID, userID)
	if err != nil {
		return MapError(fmt.Errorf("failed to unsubscribe: %w", err))
	}
	return nil
}

func (r *subscriptionRepository) Exists(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE ticket_id = $1 AND user_id = $2)`, Tables.TicketSubscriptions)

	var exists bool
	if err := r.db.QueryRow(ctx, query, ticketID, userID).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to check subscription: %w", err))
	}
	return exists, nil
}

func (r *subscriptionRepository) GetByTicket(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`SELECT user_id FROM %s WHERE ticket_id = $1`, Tables.TicketSubscriptions)

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get subscriptions: %w", err))
	}
	defer rows.Close()

	var data []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *subscriptionRepository) GetSubscribersByEvent(ctx context.Context, ticketID, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT ts.user_id
		FROM %s ts
		JOIN %s uns ON uns.user_id = ts.user_id
		WHERE ts.ticket_id = $1
			AND uns.settings->>'enabled' = 'true'
			AND EXISTS (
				SELECT 1 FROM jsonb_array_elements(uns.settings->'categories') c
				WHERE c->>'id' = $2 AND c->>$3 = 'true'
			)`, Tables.TicketSubscriptions, Tables.NotificationSettings)

	rows, err := r.db.Query(ctx, query, ticketID, categoryID.String(), eventField)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get subscribers by event: %w", err))
	}
	defer rows.Close()

	var data []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, id)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}
