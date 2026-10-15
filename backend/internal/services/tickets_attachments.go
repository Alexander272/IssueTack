package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// UploadAttachment сохраняет вложение через AttachmentService и, если вложение не
// привязано к комментарию (CommentID == nil), рассылает уведомление «Новое вложение»
// исполнителю и подписанным. Файлы из комментариев отдельно не уведомляют — они уже
// покрываются TicketCommented. Уведомление выполняется best-effort: его сбой не
// превращает успешную загрузку файла в ошибку, но сигнализируется в лог.
func (s *TicketService) UploadAttachment(ctx context.Context, tx postgres.Tx, dto *models.UploadAttachmentDTO) (*models.Attachment, error) {
	att, err := s.attachments.Upload(ctx, tx, dto)
	if err != nil {
		return nil, err
	}
	if dto.CommentID != nil {
		return att, nil
	}

	ticketID, err := s.attachmentTicketID(ctx, dto.EntityType, dto.EntityID)
	if err != nil {
		return att, nil // доступа к тикету нет — уведомление всё равно некому рассылать
	}
	ticket, err := s.GetSummary(ctx, ticketID)
	if err != nil {
		logger.Warn("failed to load ticket for attachment notification",
			logger.StringAttr("entity_id", dto.EntityID.String()), logger.ErrAttr(err))
		return att, nil
	}
	if err := s.notifications.AttachmentAdded(ctx, ticket, dto.UploadedBy); err != nil {
		logger.Warn("failed to notify about attachment",
			logger.StringAttr("entity_id", dto.EntityID.String()), logger.ErrAttr(err))
	}
	return att, nil
}

// attachmentTicketID определяет ID родительского тикета сущности вложения: для тикета —
// сам entityID, для подзадачи — ticketID родительского тикета.
func (s *TicketService) attachmentTicketID(ctx context.Context, entityType string, entityID uuid.UUID) (uuid.UUID, error) {
	switch entityType {
	case "ticket":
		return entityID, nil
	case "subtask":
		sub, err := s.subtasks.GetRawByID(ctx, &models.GetSubtaskDTO{ID: entityID})
		if err != nil {
			return uuid.Nil, err
		}
		return sub.TicketID, nil
	default:
		return uuid.Nil, fmt.Errorf("unknown entity type: %s", entityType)
	}
}
