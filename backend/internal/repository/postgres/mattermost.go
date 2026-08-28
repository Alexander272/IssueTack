package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MattermostRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewMattermostRepo(db *pgxpool.Pool, tr Transaction) *MattermostRepo {
	return &MattermostRepo{
		db:          db,
		Transaction: tr,
	}
}

type MattermostRepoInterface interface {
	GetByRealm(ctx context.Context, realmID uuid.UUID) (*models.RealmMattermost, error)
	GetByBotUserID(ctx context.Context, botUserID string) (*models.RealmMattermost, error)
	GetActive(ctx context.Context) ([]*models.RealmMattermost, error)
	GetByChannelID(ctx context.Context, channelID string) (*models.RealmMattermost, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.RealmMattermost, error)
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*models.RealmMattermost, error)
	Upsert(ctx context.Context, tx Tx, dto *models.RealmMattermost) error
	Delete(ctx context.Context, tx Tx, realmID uuid.UUID) error
}

func (r *MattermostRepo) GetByRealm(ctx context.Context, realmID uuid.UUID) (*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT realm_id, bot_token, bot_user_id, channel_id, webhook_secret, is_active, created_at, updated_at
		FROM %s WHERE realm_id = $1`,
		Tables.Mattermost,
	)

	item := &models.RealmMattermost{}
	if err := r.db.QueryRow(ctx, query, realmID).Scan(
		&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
		&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) GetByBotUserID(ctx context.Context, botUserID string) (*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT realm_id, bot_token, bot_user_id, channel_id, webhook_secret, is_active, created_at, updated_at
		FROM %s WHERE bot_user_id = $1 AND is_active = TRUE`,
		Tables.Mattermost,
	)

	item := &models.RealmMattermost{}
	if err := r.db.QueryRow(ctx, query, botUserID).Scan(
		&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
		&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) GetActive(ctx context.Context) ([]*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT realm_id, bot_token, bot_user_id, channel_id, webhook_secret, is_active, created_at, updated_at
		FROM %s WHERE is_active = TRUE`,
		Tables.Mattermost,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.RealmMattermost
	for rows.Next() {
		item := &models.RealmMattermost{}
		if err := rows.Scan(
			&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
			&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
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

func (r *MattermostRepo) GetByChannelID(ctx context.Context, channelID string) (*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT realm_id, bot_token, bot_user_id, channel_id, webhook_secret, is_active, created_at, updated_at
		FROM %s WHERE channel_id = $1 AND is_active = TRUE`,
		Tables.Mattermost,
	)

	item := &models.RealmMattermost{}
	if err := r.db.QueryRow(ctx, query, channelID).Scan(
		&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
		&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT rm.realm_id, rm.bot_token, rm.bot_user_id, rm.channel_id,
		rm.webhook_secret, rm.is_active, rm.created_at, rm.updated_at
		FROM %s rm
		INNER JOIN user_realms ur ON ur.realm_id = rm.realm_id
		WHERE ur.user_id = $1 AND rm.is_active = TRUE LIMIT 1`,
		Tables.Mattermost,
	)

	item := &models.RealmMattermost{}
	if err := r.db.QueryRow(ctx, query, userID).Scan(
		&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
		&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*models.RealmMattermost, error) {
	query := fmt.Sprintf(`SELECT rm.realm_id, rm.bot_token, rm.bot_user_id, rm.channel_id,
		rm.webhook_secret, rm.is_active, rm.created_at, rm.updated_at
		FROM %s rm
		INNER JOIN tickets t ON t.realm_id = rm.realm_id
		WHERE t.id = $1 AND rm.is_active = TRUE LIMIT 1`,
		Tables.Mattermost,
	)

	item := &models.RealmMattermost{}
	if err := r.db.QueryRow(ctx, query, ticketID).Scan(
		&item.RealmID, &item.BotToken, &item.BotUserID, &item.ChannelID,
		&item.WebhookSecret, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) Upsert(ctx context.Context, tx Tx, dto *models.RealmMattermost) error {
	query := fmt.Sprintf(`INSERT INTO %s (realm_id, bot_token, bot_user_id, channel_id, webhook_secret, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (realm_id) DO UPDATE SET
			bot_token = EXCLUDED.bot_token, bot_user_id = EXCLUDED.bot_user_id,
			channel_id = EXCLUDED.channel_id,
			webhook_secret = EXCLUDED.webhook_secret, is_active = EXCLUDED.is_active,
			updated_at = now()`,
		Tables.Mattermost,
	)

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.RealmID, dto.BotToken, dto.BotUserID, dto.ChannelID,
		dto.WebhookSecret, dto.IsActive,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *MattermostRepo) Delete(ctx context.Context, tx Tx, realmID uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE realm_id = $1`, Tables.Mattermost)

	_, err := r.getExec(tx).Exec(ctx, query, realmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
