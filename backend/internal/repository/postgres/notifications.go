package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type notificationRepository struct {
	db *pgxpool.Pool
	Transaction
}

func NewNotificationRepo(db *pgxpool.Pool, tr Transaction) *notificationRepository {
	return &notificationRepository{
		db:          db,
		Transaction: tr,
	}
}

type Notifications interface {
	Create(ctx context.Context, tx Tx, dto *models.CreateNotificationDTO) error
	GetUnread(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error)
	MarkRead(ctx context.Context, tx Tx, id uuid.UUID) error
	MarkAllRead(ctx context.Context, tx Tx, userID uuid.UUID) error
	GetResponsibleByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error)
	// GetCategoryEventSubscribers возвращает ID пользователей, у которых мастер-переключатель
	// enabled включён и заданное событие (поле eventField в матрице «категория × событие»)
	// включено для категории categoryID.
	GetCategoryEventSubscribers(ctx context.Context, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error)
	// GetGroupEventSubscribers возвращает ID участников группы groupID, у которых включён
	// мастер-переключатель enabled и заданное событие включено для этой группы.
	GetGroupEventSubscribers(ctx context.Context, groupID uuid.UUID, eventField string) ([]uuid.UUID, error)
	GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error)
	SaveSettings(ctx context.Context, tx Tx, userID uuid.UUID, settings json.RawMessage) error
	GetRealmAdmins(ctx context.Context, realmID uuid.UUID) ([]uuid.UUID, error)
	// GetOverdueTicketIDs возвращает ID «активных» тикетов с просроченным сроком (due_date < now).
	GetOverdueTicketIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error)
	// HasNotification возвращает true, если у пользователя уже есть уведомление заданного типа
	// по конкретному тикету (по полю data.ticket_id) — используется для дедупликации просрочки.
	HasNotification(ctx context.Context, userID, ticketID uuid.UUID, notifType string) (bool, error)
}

func (r *notificationRepository) Create(ctx context.Context, tx Tx, dto *models.CreateNotificationDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5, $6)`, Tables.Notifications)
	id := uuid.New()

	_, err := r.getExec(tx).Exec(ctx, query, id, dto.UserID, dto.Type, dto.Title, dto.Body, dto.Data)
	if err != nil {
		return MapError(fmt.Errorf("failed to create notification: %w", err))
	}
	return nil
}

func (r *notificationRepository) GetUnread(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error) {
	query := fmt.Sprintf(`SELECT id, user_id, type, title, body, data, is_read, created_at FROM %s WHERE user_id = $1 AND is_read = FALSE ORDER BY created_at DESC`, Tables.Notifications)

	var data []*models.Notification
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get unread notifications: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Notification{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Title, &item.Body, &item.Data, &item.IsRead, &item.CreatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, tx Tx, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET is_read = TRUE WHERE id = $1`, Tables.Notifications)
	_, err := r.getExec(tx).Exec(ctx, query, id)
	if err != nil {
		return MapError(fmt.Errorf("failed to mark notification as read: %w", err))
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, tx Tx, userID uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`, Tables.Notifications)
	_, err := r.getExec(tx).Exec(ctx, query, userID)
	if err != nil {
		return MapError(fmt.Errorf("failed to mark all notifications as read: %w", err))
	}
	return nil
}

func (r *notificationRepository) GetResponsibleByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT gm.user_id
		FROM %s gm
		JOIN %s c ON c.group_id = gm.group_id
		WHERE c.id = $1
	`, Tables.GroupMembers, Tables.Categories)

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get responsible by category: %w", err))
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

func (r *notificationRepository) GetCategoryEventSubscribers(ctx context.Context, categoryID uuid.UUID, eventField string) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT user_id
		FROM %s
		WHERE settings->>'enabled' = 'true'
			AND EXISTS (
				SELECT 1 FROM jsonb_array_elements(settings->'categories') c
				WHERE c->>'id' = $1 AND c->>$2 = 'true'
			)`, Tables.NotificationSettings)

	return r.querySubscribers(ctx, query, categoryID.String(), eventField, "failed to get category event subscribers")
}

func (r *notificationRepository) GetGroupEventSubscribers(ctx context.Context, groupID uuid.UUID, eventField string) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT gm.user_id
		FROM %s gm
		JOIN %s uns ON uns.user_id = gm.user_id
		WHERE gm.group_id = $1
			AND uns.settings->>'enabled' = 'true'
			AND EXISTS (
				SELECT 1 FROM jsonb_array_elements(uns.settings->'groups') g
				WHERE g->>'id' = $1 AND g->>$2 = 'true'
			)`, Tables.GroupMembers, Tables.NotificationSettings)

	return r.querySubscribers(ctx, query, groupID.String(), eventField, "failed to get group event subscribers")
}

func (r *notificationRepository) querySubscribers(ctx context.Context, query string, id, eventField string, errMsg string) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, query, id, eventField)
	if err != nil {
		return nil, MapError(fmt.Errorf("%s: %w", errMsg, err))
	}
	defer rows.Close()

	var data []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *notificationRepository) GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error) {
	query := fmt.Sprintf(`SELECT user_id, settings FROM %s WHERE user_id = $1`, Tables.NotificationSettings)

	settings := &models.NotificationSettings{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&settings.UserID, &settings.Settings)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &models.NotificationSettings{
				UserID:   userID,
				Settings: []byte(`{"enabled":true,"categories":[],"groups":[]}`),
			}, nil
		}
		return nil, MapError(fmt.Errorf("failed to get notification settings: %w", err))
	}
	return settings, nil
}

// SaveSettings сохраняет персональные настройки уведомлений пользователя (upsert).
func (r *notificationRepository) SaveSettings(ctx context.Context, tx Tx, userID uuid.UUID, settings json.RawMessage) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, settings)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET settings = EXCLUDED.settings`, Tables.NotificationSettings)

	_, err := r.getExec(tx).Exec(ctx, query, userID, settings)
	if err != nil {
		return MapError(fmt.Errorf("failed to save notification settings: %w", err))
	}
	return nil
}

// GetRealmAdmins возвращает ID пользователей реалма с ролью admin или root.
func (r *notificationRepository) GetRealmAdmins(ctx context.Context, realmID uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT ur.user_id
		FROM %s ur
		JOIN %s r ON ur.role_id = r.id
		WHERE ur.realm_id = $1 AND ur.is_active = true AND r.is_active = true
			AND r.slug IN ('admin', 'root')`, Tables.UserRealms, Tables.Roles)

	rows, err := r.db.Query(ctx, query, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get realm admins: %w", err))
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

// GetOverdueTicketIDs возвращает ID активных тикетов с просроченным сроком (due_date < now).
func (r *notificationRepository) GetOverdueTicketIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT id
		FROM %s
		WHERE due_date < $1
			AND closed_at IS NULL
			AND status IN ('open', 'in_progress', 'pending', 'on_hold')`, Tables.Tickets)

	rows, err := r.db.Query(ctx, query, now)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to get overdue ticket ids: %w", err))
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

// HasNotification возвращает true, если у пользователя уже есть уведомление заданного типа
// по конкретному тикету (по полю data.ticket_id).
func (r *notificationRepository) HasNotification(ctx context.Context, userID, ticketID uuid.UUID, notifType string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s
			WHERE user_id = $1 AND type = $2 AND data->>'ticket_id' = $3
		)`, Tables.Notifications)

	var exists bool
	if err := r.db.QueryRow(ctx, query, userID, notifType, ticketID.String()).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to check existing notification: %w", err))
	}
	return exists, nil
}
