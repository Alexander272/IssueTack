package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
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
	GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error)
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

func (r *notificationRepository) GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error) {
	query := fmt.Sprintf(`SELECT user_id, settings FROM %s WHERE user_id = $1`, Tables.NotificationSettings)

	settings := &models.NotificationSettings{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&settings.UserID, &settings.Settings)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &models.NotificationSettings{
				UserID:   userID,
				Settings: []byte(`{"push":true}`),
			}, nil
		}
		return nil, MapError(fmt.Errorf("failed to get notification settings: %w", err))
	}
	return settings, nil
}
