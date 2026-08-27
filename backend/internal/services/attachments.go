package services

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
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
	repo         repository.Attachments
	conf         *config.FileServerConfig
	ticketAccess TicketAccessChecker
	subtaskRepo  repository.Subtasks
}

func NewAttachmentService(repo repository.Attachments, conf *config.FileServerConfig, ticketAccess TicketAccessChecker, subtaskRepo repository.Subtasks) *AttachmentService {
	return &AttachmentService{
		repo:         repo,
		conf:         conf,
		ticketAccess: ticketAccess,
		subtaskRepo:  subtaskRepo,
	}
}

func (s *AttachmentService) checkEntityAccess(ctx context.Context, dto *models.EntityAccessDTO, action string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	switch dto.EntityType {
	case "ticket":
		if action == string(access.Write) {
			return s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: dto.EntityID, UserID: dto.ActorID, Realm: dto.Realm})
		}
		return s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: dto.EntityID, UserID: dto.ActorID, Action: action, Realm: dto.Realm})
	case "subtask":
		sub, err := s.subtaskRepo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.EntityID})
		if err != nil {
			return fmt.Errorf("failed to load subtask for access check: %w", err)
		}
		if action == string(access.Write) {
			return s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: sub.TicketID, UserID: dto.ActorID, Realm: dto.Realm})
		}
		return s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: sub.TicketID, UserID: dto.ActorID, Action: action, Realm: dto.Realm})
	}
	return fmt.Errorf("unknown entity type: %s", dto.EntityType)
}

type Attachments interface {
	GetByEntity(ctx context.Context, dto *models.EntityAccessDTO) ([]*models.Attachment, error)
	GetContent(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error)
	Upload(ctx context.Context, tx postgres.Tx, dto *models.UploadAttachmentDTO) (*models.Attachment, error)
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteAttachmentDTO) error
}

func (s *AttachmentService) GetByEntity(ctx context.Context, dto *models.EntityAccessDTO) ([]*models.Attachment, error) {
	if err := s.checkEntityAccess(ctx, dto, string(access.Read)); err != nil {
		return nil, err
	}
	data, err := s.repo.GetByEntity(ctx, dto.EntityType, dto.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	return data, nil
}

func (s *AttachmentService) GetContent(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error) {
	att, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	if err := s.checkEntityAccess(ctx, &models.EntityAccessDTO{
		EntityType: att.EntityType,
		EntityID:   att.EntityID,
		ActorID:    actorID,
		Realm:      realm,
	}, string(access.Read)); err != nil {
		return nil, nil, err
	}

	f, err := os.Open(att.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return att, f, nil
}

func (s *AttachmentService) Upload(ctx context.Context, tx postgres.Tx, dto *models.UploadAttachmentDTO) (*models.Attachment, error) {
	if !allowedEntityTypes[dto.EntityType] {
		return nil, fmt.Errorf("invalid entity type: %s", dto.EntityType)
	}

	if err := s.checkEntityAccess(ctx, &models.EntityAccessDTO{
		EntityType: dto.EntityType,
		EntityID:   dto.EntityID,
		ActorID:    dto.UploadedBy,
		Realm:      dto.Realm,
	}, string(access.Write)); err != nil {
		return nil, err
	}

	ext := filepath.Ext(dto.FileName)
	mimeType := dto.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	base := filepath.Base(dto.FileName[:len(dto.FileName)-len(ext)])
	safeName := fmt.Sprintf("%s_%s%s", uuid.New().String(), base, ext)

	relPath := filepath.Join(dto.EntityType, dto.EntityID.String(), safeName)
	absPath := filepath.Join(s.conf.UploadDir, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	if _, err := io.Copy(dst, dto.File); err != nil {
		dst.Close()
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	dst.Close()

	att := &models.Attachment{
		EntityType: dto.EntityType,
		EntityID:   dto.EntityID,
		FileName:   dto.FileName,
		FilePath:   absPath,
		FileSize:   dto.FileSize,
		MimeType:   mimeType,
		UploadedBy: dto.UploadedBy,
	}

	if err := s.repo.Create(ctx, tx, att); err != nil {
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to save attachment: %w", err)
	}

	return att, nil
}

func (s *AttachmentService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteAttachmentDTO) error {
	att, err := s.repo.GetByID(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to load attachment: %w", err)
	}

	if err := s.checkEntityAccess(ctx, &models.EntityAccessDTO{
		EntityType: att.EntityType,
		EntityID:   att.EntityID,
		ActorID:    dto.ActorID,
		Realm:      dto.Realm,
	}, string(access.Write)); err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}

	if err := s.repo.Delete(ctx, tx, dto.ID); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if err := os.Remove(att.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
