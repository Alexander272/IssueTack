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
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

var allowedEntityTypes = map[string]bool{
	"ticket":  true,
	"subtask": true,
}

// AttachmentService — сервис управления вложениями (загрузка, чтение, удаление) с проверкой доступа к сущности.
// Уведомление о новом вложении рассылает TicketService.UploadAttachment — сервис-владелец агрегата,
// чтобы вложения не зависели от тикетов (цикл tickets → attachments).
type AttachmentService struct {
	repo         repository.Attachments
	conf         *config.FileServerConfig
	ticketAccess TicketAccessChecker
	subtasks     Subtasks
}

// NewAttachmentService создаёт AttachmentService.
func NewAttachmentService(repo repository.Attachments, conf *config.FileServerConfig, ticketAccess TicketAccessChecker, subtasks Subtasks) *AttachmentService {
	return &AttachmentService{
		repo:         repo,
		conf:         conf,
		ticketAccess: ticketAccess,
		subtasks:     subtasks,
	}
}

// checkEntityAccess проверяет право действия action на сущность вложения. Для тикета проверка
// идёт напрямую; для подзадачи сначала загружается её тикет, т.к. доступ к вложению подзадачи
// определяется доступом к родительскому тикету. Для write используется CheckWorkAccess
// (исполнитель тоже может работать с тикетом), для остальных действий — CheckAccess.
func (s *AttachmentService) checkEntityAccess(ctx context.Context, dto *models.EntityAccessDTO, action string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	switch dto.EntityType {
	case "ticket":
		if action == string(access.Write) {
			return s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: dto.EntityID, UserID: dto.ActorID})
		}
		return s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: dto.EntityID, UserID: dto.ActorID, Action: action})
	case "subtask":
		sub, err := s.subtasks.GetRawByID(ctx, &models.GetSubtaskDTO{ID: dto.EntityID})
		if err != nil {
			return fmt.Errorf("failed to load subtask for access check: %w", err)
		}
		if action == string(access.Write) {
			return s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: sub.TicketID, UserID: dto.ActorID})
		}
		return s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: sub.TicketID, UserID: dto.ActorID, Action: action})
	}
	return fmt.Errorf("unknown entity type: %s", dto.EntityType)
}

// Attachments — интерфейс работы с вложениями.
type Attachments interface {
	// GetByEntity возвращает список вложений сущности (тикета/подзадачи).
	GetByEntity(ctx context.Context, dto *models.EntityAccessDTO) ([]*models.Attachment, error)
	// GetContent возвращает вложение и поток чтения его файла.
	GetContent(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error)
	// Upload сохраняет файл вложения и регистрирует его в системе.
	Upload(ctx context.Context, tx postgres.Tx, dto *models.UploadAttachmentDTO) (*models.Attachment, error)
	// Delete удаляет вложение и связанный файл с диска.
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteAttachmentDTO) error
	// DeleteByEntity удаляет записи вложений сущности (тикета/подзадачи) в рамках
	// переданной транзакции. Внутренний вызов при каскадном удалении родительской
	// записи: доступ не проверяется, авторизацию обеспечивает вызывающий.
	// Файлы не трогает — см. RemoveEntityDir.
	DeleteByEntity(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID) error
	// RemoveEntityDir удаляет с диска директорию вложений сущности
	// (entity_type/entity_id). Вызывается строго после успешного коммита удаления
	// записей вложений. Ошибки не возвращаются — только Warn-лог: файловая система
	// не участвует в транзакции, и её сбой не должен отменять завершённую операцию.
	RemoveEntityDir(entityType string, entityID uuid.UUID)
	// GetForComments возвращает вложения комментариев тикета, сгруппированные по
	// comment_id. Ожидается, что вызов осуществляет CommentService, уже проверивший
	// право чтения тикета и видимость внутренних комментариев.
	GetForComments(ctx context.Context, ticketID uuid.UUID, showInternal bool) (map[uuid.UUID][]*models.Attachment, error)
}

// GetByEntity возвращает список вложений сущности (тикета/подзадачи) с проверкой доступа на чтение.
// Для тикета вложения, привязанные к внутренним комментариям, скрываются от
// пользователей, не имеющих доступа к внутренним комментариям. Файлы из публичных
// комментариев и самостоятельные вложения видны всем, кому доступен тикет.
func (s *AttachmentService) GetByEntity(ctx context.Context, dto *models.EntityAccessDTO) ([]*models.Attachment, error) {
	if err := s.checkEntityAccess(ctx, dto, string(access.Read)); err != nil {
		return nil, err
	}
	data, err := s.repo.GetByEntity(ctx, dto.EntityType, dto.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	if dto.EntityType != "ticket" {
		return data, nil
	}

	showInternal := s.ticketAccess.CheckInternalAssigneeAccess(ctx, &models.AccessCheckDTO{
		TicketID: dto.EntityID,
		UserID:   dto.ActorID,
	}) == nil
	if showInternal {
		return data, nil
	}

	internalCommentIDs, _, err := s.repo.GetByComments(ctx, dto.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment attachments: %w", err)
	}

	filtered := data[:0]
	for _, att := range data {
		if att.CommentID != nil && internalCommentIDs[*att.CommentID] {
			continue
		}
		filtered = append(filtered, att)
	}
	if filtered == nil {
		filtered = []*models.Attachment{}
	}
	return filtered, nil
}

// GetForComments возвращает вложения комментариев тикета, сгруппированные по
// comment_id. Внутренние комментарии включаются только если showInternal.
func (s *AttachmentService) GetForComments(ctx context.Context, ticketID uuid.UUID, showInternal bool) (map[uuid.UUID][]*models.Attachment, error) {
	internalComments, atts, err := s.repo.GetByComments(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment attachments: %w", err)
	}

	result := map[uuid.UUID][]*models.Attachment{}
	for _, att := range atts {
		if att.CommentID == nil {
			continue
		}
		if internalComments[*att.CommentID] && !showInternal {
			continue
		}
		result[*att.CommentID] = append(result[*att.CommentID], att)
	}
	return result, nil
}

// GetContent возвращает вложение и поток чтения его файла с проверкой доступа на чтение.
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

// Upload сохраняет файл вложения на диск и создаёт запись о вложении с проверкой права на запись.
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

	var written int64
	if s.conf.MaxSize > 0 {
		written, err = io.Copy(dst, io.LimitReader(dto.File, s.conf.MaxSize+1))
	} else {
		written, err = io.Copy(dst, dto.File)
	}
	if err != nil {
		dst.Close()
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	if s.conf.MaxSize > 0 && written > s.conf.MaxSize {
		dst.Close()
		os.Remove(absPath)
		return nil, fmt.Errorf("%w: максимум %d байт", models.ErrFileTooLarge, s.conf.MaxSize)
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
		CommentID:  dto.CommentID,
	}

	if err := s.repo.Create(ctx, tx, att); err != nil {
		os.Remove(absPath)
		return nil, fmt.Errorf("failed to save attachment: %w", err)
	}

	return att, nil
}

// Delete удаляет вложение (запись в БД и файл с диска) с проверкой права на запись.
//
// Вызывается без транзакции (tx == nil), поэтому удаление строки коммитится сразу
// после repo.Delete и файл можно снимать только после этого. Если транзакция всё же
// передана, файлы не трогаются: момент коммита вызывающему неизвестен, а удаление до
// него необратимо теряет файл при откате (см. RemoveEntityDir).
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

	// Строка уже удалена и (при tx == nil) закоммичена: сбой очистки диска не должен
	// отменять операцию, поэтому только логируем. Несуществующий файл — не ошибка.
	if tx != nil {
		logger.Warn("attachment deleted within caller transaction, file kept",
			logger.StringAttr("attachment_id", dto.ID.String()),
			logger.StringAttr("file_path", att.FilePath),
		)
		return nil
	}
	if err := os.Remove(att.FilePath); err != nil && !os.IsNotExist(err) {
		logger.Warn("failed to remove attachment file",
			logger.StringAttr("attachment_id", dto.ID.String()),
			logger.StringAttr("file_path", att.FilePath),
			logger.ErrAttr(err),
		)
	}
	return nil
}

// DeleteByEntity удаляет записи вложений сущности в рамках переданной транзакции.
// Применяется при каскадном удалении тикета (вместе с подзадачами), чтобы не
// оставлять «осиротевшие» записи. Вызывается внутри транзакции удаления тикета,
// без проверки доступа — она выполнена выше, при проверке Delete-права.
//
// Файлы с диска здесь НЕ удаляются: файловая система не участвует в транзакции,
// поэтому удаление до коммита необратимо теряет их при откате (в тикете после этого
// вызова идут ещё чтение подзадач и удаление строки). Вызывающий накапливает
// директории и чистит их после успешного коммита — см. RemoveEntityDir.
func (s *AttachmentService) DeleteByEntity(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID) error {
	if !allowedEntityTypes[entityType] {
		return fmt.Errorf("invalid entity type: %s", entityType)
	}
	if err := s.repo.DeleteByEntity(ctx, tx, entityType, entityID); err != nil {
		return fmt.Errorf("failed to delete attachments: %w", err)
	}
	return nil
}

// RemoveEntityDir удаляет с диска директорию вложений сущности. Вызывается строго
// ПОСЛЕ успешного коммита удаления записей (см. DeleteByEntity).
//
// Ошибки удаления не возвращаются: файловая система не участвует в транзакции, и
// срыв очистки не должен превращать уже завершившуюся операцию в ошибку ответа —
// остаётся осиротевший файл, который уберёт сборщик мусора. Наружу уходит Warn-лог.
func (s *AttachmentService) RemoveEntityDir(entityType string, entityID uuid.UUID) {
	if !allowedEntityTypes[entityType] {
		return
	}
	dir := filepath.Join(s.conf.UploadDir, entityType, entityID.String())
	if err := os.RemoveAll(dir); err != nil {
		logger.Warn("failed to remove attachment directory",
			logger.StringAttr("entity_type", entityType),
			logger.StringAttr("entity_id", entityID.String()),
			logger.ErrAttr(err),
		)
	}
}
