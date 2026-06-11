package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewAttachmentRepo(db *pgxpool.Pool, tr Transaction) *AttachmentRepo {
	return &AttachmentRepo{
		db:          db,
		Transaction: tr,
	}
}

type Attachments interface {
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error)
	Create(ctx context.Context, tx Tx, dto *models.Attachment) error
	Delete(ctx context.Context, tx Tx, id uuid.UUID) error
}

func (r *AttachmentRepo) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error) {
	query := fmt.Sprintf(`SELECT id, entity_type, entity_id, file_name, file_size, mime_type, uploaded_by, created_at
		FROM %s WHERE entity_type = $1 AND entity_id = $2 ORDER BY created_at`,
		Tables.Attachments,
	)

	rows, err := r.db.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.Attachment
	for rows.Next() {
		item := &models.Attachment{}
		if err := rows.Scan(
			&item.ID, &item.EntityType, &item.EntityID,
			&item.FileName, &item.FileSize, &item.MimeType,
			&item.UploadedBy, &item.CreatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.Attachment{}, nil
	}
	return data, nil
}

func (r *AttachmentRepo) Create(ctx context.Context, tx Tx, dto *models.Attachment) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, entity_type, entity_id, file_name, file_path, file_size, mime_type, uploaded_by) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		Tables.Attachments,
	)
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.EntityType, dto.EntityID,
		dto.FileName, dto.FilePath, dto.FileSize, dto.MimeType, dto.UploadedBy,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *AttachmentRepo) Delete(ctx context.Context, tx Tx, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Attachments)

	_, err := r.getExec(tx).Exec(ctx, query, id)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
