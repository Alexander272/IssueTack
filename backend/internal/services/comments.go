package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

const commentDeleteWindow = 15 * time.Minute

// mattermostSender — минимальный интерфейс отправки личного сообщения
// Mattermost. Реализуется типом *mattermost.Most (через встраиваемый DM).
type mattermostSender interface {
	Send(botToken, botUserID, targetUserID, message string) error
}

// CommentService — сервис работы с комментариями к тикетам.
type CommentService struct {
	repo          repository.Comments
	ticketAccess  TicketAccessChecker
	tickets       repository.Tickets
	users         Users
	mmRepo        repository.Mattermost
	mmSender      mattermostSender
	notifications Notifications
	txManager     TransactionManager
}

// NewCommentService создаёт CommentService.
func NewCommentService(repo repository.Comments, ticketAccess TicketAccessChecker, tickets repository.Tickets, users Users, mmRepo repository.Mattermost, mmSender mattermostSender, notifications Notifications, txManager TransactionManager) *CommentService {
	return &CommentService{
		repo:          repo,
		ticketAccess:  ticketAccess,
		tickets:       tickets,
		users:         users,
		mmRepo:        mmRepo,
		mmSender:      mmSender,
		notifications: notifications,
		txManager:     txManager,
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
// Для активных тикетов допускаются как внешние, так и внутренние комментарии
// (при work-доступе). Для «замороженных» (resolved/closed/cancelled) внешние
// комментарии запрещены — допустимы только внутренние, и только от исполнителя/
// менеджера группы (CheckInternalAssigneeAccess).
// Если транзакция не передана, открывает собственную (owner-tx), чтобы
// запись комментария была атомарной; side-эффекты (Mattermost, уведомления)
// выполняются уже после фикса транзакции.
func (s *CommentService) Create(ctx context.Context, tx postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error) {
	check := &models.AccessCheckDTO{
		TicketID: dto.TicketID,
		UserID:   dto.UserID,
		Realm:    dto.Realm,
	}

	ticket, err := s.tickets.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return nil, fmt.Errorf("failed to load ticket for comment access: %w", err)
	}

	if isTicketInactive(ticket.Status) {
		// «Замороженная» заявка: только внутренние комментарии, внешние запрещены.
		if !dto.IsInternal {
			return nil, models.ErrTicketFrozen
		}
		if err := s.ticketAccess.CheckInternalAssigneeAccess(ctx, check); err != nil {
			return nil, err
		}
	} else {
		if err := s.ticketAccess.CheckWorkAccess(ctx, check); err != nil {
			return nil, err
		}
	}

	var createErr error
	comment := &models.Comment{
		Text:       dto.Text,
		UserID:     dto.UserID,
		TicketID:   dto.TicketID,
		IsInternal: dto.IsInternal,
		Type:       dto.Type,
	}

	if tx == nil {
		createErr = s.txManager.WithinTransaction(ctx, func(newTx postgres.Tx) error {
			return s.repo.Create(ctx, newTx, comment)
		})
	} else {
		createErr = s.repo.Create(ctx, tx, comment)
	}
	if createErr != nil {
		return nil, fmt.Errorf("failed to create comment: %w", createErr)
	}

	// Side-эффекты выполняются только после фикса транзакции и не влияют на сам
	// факт создания комментария — их сбои только логируются.
	s.notifyOwnerViaMattermost(ctx, comment)
	if s.notifications != nil {
		if err := s.notifications.TicketCommented(ctx, comment.TicketID, comment.UserID); err != nil {
			logger.Warn("failed to notify about comment", logger.StringAttr("ticket_id", comment.TicketID.String()), logger.ErrAttr(err))
		}
	}
	return comment, nil
}

// notifyOwnerViaMattermost дублирует публичный (не внутренний) комментарий из
// программы владельцу заявки в личном сообщении Mattermost. Сбои интеграции не
// влияют на сам факт создания комментария — они только логируются.
func (s *CommentService) notifyOwnerViaMattermost(ctx context.Context, comment *models.Comment) {
	if comment.IsInternal {
		return
	}
	if s.mmSender == nil || s.users == nil || s.mmRepo == nil {
		return
	}

	ticket, err := s.tickets.GetByID(ctx, &models.GetTicketByIdDTO{ID: comment.TicketID})
	if err != nil {
		logger.Warn("failed to load ticket for mattermost comment notify", logger.StringAttr("ticket_id", comment.TicketID.String()), logger.ErrAttr(err))
		return
	}
	if ticket.Owner == nil || ticket.RealmID == nil {
		return
	}
	if ticket.Owner.ID == comment.UserID {
		return
	}

	ownerUser, err := s.users.GetByID(ctx, ticket.Owner.ID)
	if err != nil || ownerUser.MattermostID == nil || *ownerUser.MattermostID == "" {
		return
	}

	settings, err := s.mmRepo.GetByRealm(ctx, *ticket.RealmID)
	if err != nil || !settings.IsActive {
		return
	}

	title := ticket.Title
	var number string
	if ticket.TicketNumber != nil {
		number = fmt.Sprintf("№%d", *ticket.TicketNumber)
	} else {
		number = "заявка"
	}

	text := fmt.Sprintf("**%s: %s**\n\n%s", number, title, comment.Text)
	if err := s.mmSender.Send(settings.BotToken, settings.BotUserID, *ownerUser.MattermostID, text); err != nil {
		logger.Warn("failed to send mattermost comment notify", logger.StringAttr("ticket_id", comment.TicketID.String()), logger.ErrAttr(err))
	}
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
