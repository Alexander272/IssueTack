package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

const commentDeleteWindow = 15 * time.Minute

// CommentService — сервис работы с комментариями к тикетам.
type CommentService struct {
	repo         repository.Comments
	ticketAccess TicketAccessChecker
}

// NewCommentService создаёт CommentService.
func NewCommentService(repo repository.Comments, ticketAccess TicketAccessChecker) *CommentService {
	return &CommentService{
		repo:         repo,
		ticketAccess: ticketAccess,
	}
}

// Comments — интерфейс работы с комментариями.
type Comments interface {
	// GetByTicket возвращает комментарии тикета с проверкой доступа.
	GetByTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, realm string) ([]*models.Comment, error)
	// Create создаёт комментарий к тикету.
	Create(ctx context.Context, tx postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error)
	// Delete удаляет комментарий.
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteCommentDTO) error
}

// GetByTicket возвращает комментарии тикета с проверкой доступа и видимостью внутренних комментариев.
func (s *CommentService) GetByTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, realm string) ([]*models.Comment, error) {
	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: ticketID,
		UserID:   userID,
		Action:   "read",
		Realm:    realm,
	}); err != nil {
		return nil, err
	}

	showAllInternal := s.ticketAccess.CheckInternalAssigneeAccess(ctx, &models.AccessCheckDTO{
		TicketID: ticketID,
		UserID:   userID,
		Realm:    realm,
	}) == nil

	data, err := s.repo.GetByTicket(ctx, ticketID, userID, showAllInternal)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	return data, nil
}

// Create создаёт комментарий к тикету с проверкой work-доступа.
func (s *CommentService) Create(ctx context.Context, tx postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error) {
	if err := s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{
		TicketID: dto.TicketID,
		UserID:   dto.UserID,
		Realm:    dto.Realm,
	}); err != nil {
		return nil, err
	}

	comment := &models.Comment{
		Text:       dto.Text,
		UserID:     dto.UserID,
		TicketID:   dto.TicketID,
		IsInternal: dto.IsInternal,
		Type:       dto.Type,
	}

	if err := s.repo.Create(ctx, tx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	return comment, nil
}

// Delete удаляет комментарий; разрешено только автору в течение окна удаления.
func (s *CommentService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteCommentDTO) error {
	comment, err := s.repo.GetByID(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to load comment: %w", err)
	}

	if comment.UserID != dto.ActorID {
		return models.ErrPermissionDenied
	}
	if time.Since(comment.CreatedAt) > commentDeleteWindow {
		return models.ErrCommentExpired
	}

	if err := s.repo.Delete(ctx, tx, dto.ID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	return nil
}
