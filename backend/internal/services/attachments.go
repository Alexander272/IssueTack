package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type AttachmentService struct {
	repo   repository.Attachments
	conf   *config.FileServerConfig
}

func NewAttachmentService(repo repository.Attachments, conf *config.FileServerConfig) *AttachmentService {
	return &AttachmentService{
		repo: repo,
		conf: conf,
	}
}

type Attachments interface {
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error)
	Upload(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID, fileName string, file io.Reader, uploadedBy uuid.UUID) (*models.Attachment, error)
	Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID) error
}

func (s *AttachmentService) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error) {
	data, err := s.repo.GetByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	return data, nil
}

func (s *AttachmentService) Upload(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID, fileName string, file io.Reader, uploadedBy uuid.UUID) (*models.Attachment, error) {
	ext := filepath.Ext(fileName)
	base := fileName[:len(fileName)-len(ext)]
	safeName := fmt.Sprintf("%s_%s%s", uuid.New().String(), base, ext)

	relPath := filepath.Join(entityType, entityID.String(), safeName)
	absPath := filepath.Join(s.conf.UploadDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	att := &models.Attachment{
		EntityType: entityType,
		EntityID:   entityID,
		FileName:   fileName,
		FilePath:   absPath,
		UploadedBy: uploadedBy,
	}

	if err := s.repo.Create(ctx, tx, att); err != nil {
		return nil, fmt.Errorf("failed to save attachment: %w", err)
	}

	return att, nil
}

func (s *AttachmentService) Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, tx, id); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	return nil
}
