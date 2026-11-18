package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type favoriteRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewFavoriteRepo(db *pgxpool.Pool, tr Transaction) *favoriteRepository {
	return &favoriteRepository{
		db:          db,
		Transaction: tr,
	}
}

type TicketFavorites interface {
	Add(ctx context.Context, tx Tx, dto *models.FavoriteDTO) error
	Remove(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID, favoriteType models.FavoriteType) error
	Exists(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, favoriteType models.FavoriteType) (bool, error)
	GetByUser(ctx context.Context, userID uuid.UUID, favoriteType models.FavoriteType) ([]*models.TicketFavorite, error)
	// DeleteTemporaryByTickets снимает «закреплённые» (temporary) всех пользователей
	// с указанных заявок. Вызывается при заморозке заявки (resolved/closed/cancelled)
	// в той же транзакции, что и смена статуса.
	DeleteTemporaryByTickets(ctx context.Context, tx Tx, ticketIDs []uuid.UUID) error
	// DeleteTemporaryOnInactive — страховочная очистка джобом: снимает temporary
	// со всех заявок в «замороженных» статусах. Возвращает число удалённых записей.
	DeleteTemporaryOnInactive(ctx context.Context) (int64, error)
}

func (r *favoriteRepository) Add(ctx context.Context, tx Tx, dto *models.FavoriteDTO) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, user_id, ticket_id, type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ticket_id, type) DO NOTHING`, Tables.TicketFavorites)

	_, err := r.getExec(tx).Exec(ctx, query, uuid.New(), dto.UserID, dto.TicketID, dto.Type)
	if err != nil {
		return MapError(fmt.Errorf("failed to add favorite: %w", err))
	}
	return nil
}

func (r *favoriteRepository) Remove(ctx context.Context, tx Tx, ticketID uuid.UUID, userID uuid.UUID, favoriteType models.FavoriteType) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE ticket_id = $1 AND user_id = $2 AND type = $3`, Tables.TicketFavorites)

	_, err := r.getExec(tx).Exec(ctx, query, ticketID, userID, favoriteType)
	if err != nil {
		return MapError(fmt.Errorf("failed to remove favorite: %w", err))
	}
	return nil
}

func (r *favoriteRepository) Exists(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, favoriteType models.FavoriteType) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE ticket_id = $1 AND user_id = $2 AND type = $3)`, Tables.TicketFavorites)

	var exists bool
	if err := r.db.QueryRow(ctx, query, ticketID, userID, favoriteType).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to check favorite: %w", err))
	}
	return exists, nil
}

func (r *favoriteRepository) GetByUser(ctx context.Context, userID uuid.UUID, favoriteType models.FavoriteType) ([]*models.TicketFavorite, error) {
	query := fmt.Sprintf(`SELECT id, user_id, ticket_id, type, created_at FROM %s WHERE user_id = $1 AND type = $2`, Tables.TicketFavorites)

	rows, err := r.db.Query(ctx, query, userID, favoriteType)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get favorites: %w", err))
	}
	defer rows.Close()

	data := []*models.TicketFavorite{}
	for rows.Next() {
		var fav models.TicketFavorite
		if err := rows.Scan(&fav.ID, &fav.UserID, &fav.TicketID, &fav.Type, &fav.CreatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, &fav)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *favoriteRepository) DeleteTemporaryByTickets(ctx context.Context, tx Tx, ticketIDs []uuid.UUID) error {
	if len(ticketIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE type = $1 AND ticket_id = ANY($2)`, Tables.TicketFavorites)
	if _, err := r.getExec(tx).Exec(ctx, query, models.FavoriteTypeTemporary, ticketIDs); err != nil {
		return MapError(fmt.Errorf("failed to delete temporary favorites: %w", err))
	}
	return nil
}

func (r *favoriteRepository) DeleteTemporaryOnInactive(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s f
		USING %s t
		WHERE t.id = f.ticket_id
			AND f.type = $1
			AND t.status IN ($2, $3, $4)`,
		Tables.TicketFavorites, Tables.Tickets,
	)

	cmd, err := r.db.Exec(ctx, query,
		models.FavoriteTypeTemporary, models.StatusResolved, models.StatusClosed, models.StatusCancelled,
	)
	if err != nil {
		return 0, MapError(fmt.Errorf("failed to delete temporary favorites on inactive tickets: %w", err))
	}
	return cmd.RowsAffected(), nil
}
