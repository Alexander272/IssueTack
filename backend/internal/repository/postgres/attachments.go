package postgres

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"

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

func fillMimeType(a *models.Attachment) {
	if a.MimeType == "" {
		a.MimeType = mime.TypeByExtension(filepath.Ext(a.FileName))
	}
}

type Attachments interface {
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Attachment, error)
	Create(ctx context.Context, tx Tx, dto *models.Attachment) error
	Delete(ctx context.Context, tx Tx, id uuid.UUID) error
	// GetByComments возвращает вложения, привязанные к комментариям указанного
	// тикета, сгруппированные по comment_id и с флагом внутреннего комментария.
	GetByComments(ctx context.Context, ticketID uuid.UUID) (map[uuid.UUID]bool, []*models.Attachment, error)
}

func (r *AttachmentRepo) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error) {
	query := fmt.Sprintf(`SELECT id, entity_type, entity_id, file_name, file_size, mime_type, uploaded_by, comment_id, created_at
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
			&item.UploadedBy, &item.CommentID, &item.CreatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		fillMimeType(item)
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

func (r *AttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Attachment, error) {
	query := fmt.Sprintf(`SELECT id, entity_type, entity_id, file_name, file_path, file_size, mime_type, uploaded_by, comment_id, created_at
		FROM %s WHERE id = $1`,
		Tables.Attachments,
	)

	item := &models.Attachment{}
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.EntityType, &item.EntityID,
		&item.FileName, &item.FilePath, &item.FileSize, &item.MimeType,
		&item.UploadedBy, &item.CommentID, &item.CreatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	fillMimeType(item)
	return item, nil
}

func (r *AttachmentRepo) Create(ctx context.Context, tx Tx, dto *models.Attachment) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, entity_type, entity_id, file_name, file_path, file_size, mime_type, uploaded_by, comment_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		Tables.Attachments,
	)
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, dto.EntityType, dto.EntityID,
		dto.FileName, dto.FilePath, dto.FileSize, dto.MimeType, dto.UploadedBy, dto.CommentID,
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

// GetByComments возвращает вложения тикета, привязанные к комментариям, вместе
// с набором comment_id, у которых комментарий внутренний. Возвращаемый map —
// comment_id -> is_internal.
func (r *AttachmentRepo) GetByComments(ctx context.Context, ticketID uuid.UUID) (map[uuid.UUID]bool, []*models.Attachment, error) {
	query := fmt.Sprintf(`SELECT a.id, a.entity_type, a.entity_id, a.file_name, a.file_size, a.mime_type,
			a.uploaded_by, a.comment_id, a.created_at, c.is_internal
		FROM %s a
		JOIN %s c ON c.id = a.comment_id
		WHERE a.entity_type = 'ticket' AND a.entity_id = $1 AND a.comment_id IS NOT NULL
		ORDER BY a.created_at`,
		Tables.Attachments, Tables.Comments,
	)

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	internal := map[uuid.UUID]bool{}
	var data []*models.Attachment
	for rows.Next() {
		item := &models.Attachment{}
		var commentID uuid.UUID
		var isInternal bool
		if err := rows.Scan(
			&item.ID, &item.EntityType, &item.EntityID,
			&item.FileName, &item.FileSize, &item.MimeType,
			&item.UploadedBy, &commentID, &item.CreatedAt, &isInternal,
		); err != nil {
			return nil, nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		item.CommentID = &commentID
		fillMimeType(item)
		data = append(data, item)
		if isInternal {
			internal[commentID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		data = []*models.Attachment{}
	}
	return internal, data, nil
}
