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

var allowedEntityTypes = map[string]bool{
	"ticket":  true,
	"subtask": true,
}

type AttachmentService struct {
	repo          repository.Attachments
	conf          *config.FileServerConfig
	ticketAccess  TicketAccessChecker
	subtaskRepo   repository.Subtasks
}

func NewAttachmentService(repo repository.Attachments, conf *config.FileServerConfig, ticketAccess TicketAccessChecker, subtaskRepo repository.Subtasks) *AttachmentService {
	return &AttachmentService{
		repo:         repo,
		conf:         conf,
		ticketAccess: ticketAccess,
		subtaskRepo:  subtaskRepo,
	}
}

func (s *AttachmentService) SetTicketAccess(checker TicketAccessChecker) {
	s.ticketAccess = checker
}

func (s *AttachmentService) checkEntityAccess(ctx context.Context, entityType string, entityID, actorID uuid.UUID, action string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	switch entityType {
	case "ticket":
		return s.ticketAccess.CheckAccess(ctx, entityID, actorID, action)
	case "subtask":
		sub, err := s.subtaskRepo.GetByID(ctx, &models.GetSubtaskDTO{ID: entityID})
		if err != nil {
			return fmt.Errorf("failed to load subtask for access check: %w", err)
		}
		return s.ticketAccess.CheckAccess(ctx, sub.TicketID, actorID, action)
	}
	return fmt.Errorf("unknown entity type: %s", entityType)
}

type Attachments interface {
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID, actorID uuid.UUID) ([]*models.Attachment, error)
	Upload(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID, fileName string, file io.Reader, uploadedBy uuid.UUID) (*models.Attachment, error)
	Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID, actorID uuid.UUID) error
}

func (s *AttachmentService) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID, actorID uuid.UUID) ([]*models.Attachment, error) {
	if err := s.checkEntityAccess(ctx, entityType, entityID, actorID, "read"); err != nil {
		return nil, err
	}
	data, err := s.repo.GetByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	return data, nil
}

func (s *AttachmentService) Upload(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID, fileName string, file io.Reader, uploadedBy uuid.UUID) (*models.Attachment, error) {
	if !allowedEntityTypes[entityType] {
		return nil, fmt.Errorf("invalid entity type: %s", entityType)
	}

	if err := s.checkEntityAccess(ctx, entityType, entityID, uploadedBy, "write"); err != nil {
		return nil, err
	}

	ext := filepath.Ext(fileName)
	base := filepath.Base(fileName[:len(fileName)-len(ext)])
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

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	dst.Close()

	att := &models.Attachment{
		EntityType: entityType,
		EntityID:   entityID,
		FileName:   fileName,
		FilePath:   absPath,
		UploadedBy: uploadedBy,
	}

	if err := s.repo.Create(ctx, tx, att); err != nil {
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to save attachment: %w", err)
	}

	return att, nil
}

func (s *AttachmentService) Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID, actorID uuid.UUID) error {
	att, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load attachment: %w", err)
	}

	if err := s.checkEntityAccess(ctx, att.EntityType, att.EntityID, actorID, "write"); err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}

	if err := s.repo.Delete(ctx, tx, id); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if err := os.Remove(att.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
