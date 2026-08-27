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
	GetUserLinkByUserID(ctx context.Context, userID uuid.UUID) (*models.MattermostUserLink, error)
	GetUserLinkByMmUserID(ctx context.Context, mmUserID string) (*models.MattermostUserLink, error)
	UpsertUserLink(ctx context.Context, tx Tx, dto *models.MattermostUserLink) error
	UpsertUserLinks(ctx context.Context, tx Tx, dto []*models.MattermostUserLink) error
	GetAllLinks(ctx context.Context) ([]*models.MattermostUserLink, error)
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
		INNER JOIN mattermost_user_links mul ON mul.mm_user_id IS NOT NULL
		INNER JOIN user_realms ur ON ur.user_id = mul.user_id AND ur.realm_id = rm.realm_id
		WHERE mul.user_id = $1 AND rm.is_active = TRUE LIMIT 1`,
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

func (r *MattermostRepo) GetUserLinkByUserID(ctx context.Context, userID uuid.UUID) (*models.MattermostUserLink, error) {
	query := fmt.Sprintf(`SELECT user_id, mm_user_id FROM %s WHERE user_id = $1`,
		Tables.MattermostUserLinks,
	)

	item := &models.MattermostUserLink{}
	if err := r.db.QueryRow(ctx, query, userID).Scan(&item.UserID, &item.MmUserID); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) GetUserLinkByMmUserID(ctx context.Context, mmUserID string) (*models.MattermostUserLink, error) {
	query := fmt.Sprintf(`SELECT user_id, mm_user_id FROM %s WHERE mm_user_id = $1`,
		Tables.MattermostUserLinks,
	)

	item := &models.MattermostUserLink{}
	if err := r.db.QueryRow(ctx, query, mmUserID).Scan(&item.UserID, &item.MmUserID); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *MattermostRepo) UpsertUserLink(ctx context.Context, tx Tx, dto *models.MattermostUserLink) error {
	query := fmt.Sprintf(`INSERT INTO %s (user_id, mm_user_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET mm_user_id = EXCLUDED.mm_user_id`,
		Tables.MattermostUserLinks,
	)

	_, err := r.getExec(tx).Exec(ctx, query, dto.UserID, dto.MmUserID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *MattermostRepo) UpsertUserLinks(ctx context.Context, tx Tx, dto []*models.MattermostUserLink) error {
	if len(dto) == 0 {
		return nil
	}
	n := len(dto)
	userIDs := make([]uuid.UUID, n)
	mmIDs := make([]string, n)
	for i, v := range dto {
		userIDs[i] = v.UserID
		mmIDs[i] = v.MmUserID
	}
	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, mm_user_id)
		SELECT * FROM UNNEST($1::uuid[], $2::text[])
		ON CONFLICT (user_id) DO UPDATE SET mm_user_id = EXCLUDED.mm_user_id`,
		Tables.MattermostUserLinks,
	)
	_, err := r.getExec(tx).Exec(ctx, query, userIDs, mmIDs)
	if err != nil {
		return MapError(fmt.Errorf("failed to upsert user links: %w", err))
	}
	return nil
}

func (r *MattermostRepo) GetAllLinks(ctx context.Context) ([]*models.MattermostUserLink, error) {
	query := fmt.Sprintf(`SELECT user_id, mm_user_id FROM %s ORDER BY user_id`,
		Tables.MattermostUserLinks,
	)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.MattermostUserLink
	for rows.Next() {
		item := &models.MattermostUserLink{}
		if err := rows.Scan(&item.UserID, &item.MmUserID); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.MattermostUserLink{}, nil
	}
	return data, nil
}
