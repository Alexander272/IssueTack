package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

type recentTicket struct {
	id        uuid.UUID
	createdAt time.Time
}

// HandleDialogOpen открывает диалог создания заявки для пользователя,
// нажавшего кнопку «Создать заявку». Подгружает категории/площадки realm
// и формирует поля диалога; в State передаёт канал и id кнопочного поста,
// чтобы после создания оповестить пользователя и удалить кнопку.
func (s *MattermostService) HandleDialogOpen(ctx context.Context, triggerID, _, channelID, buttonPostID string, actionCtx map[string]string) error {
	if triggerID == "" {
		return fmt.Errorf("missing trigger_id")
	}
	if s.baseURL == "" {
		return fmt.Errorf("failed to open dialog: http.base_url is not configured")
	}

	realmID, err := uuid.Parse(actionCtx["realm_id"])
	if err != nil {
		return fmt.Errorf("invalid realm_id: %w", err)
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return fmt.Errorf("failed to get mattermost settings: %w", err)
	}

	categories, err := s.categories.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	if err != nil {
		logger.Warn("failed to load categories for dialog", logger.ErrAttr(err))
	}

	sites, err := s.sites.Get(ctx, &models.GetSitesDTO{})
	if err != nil {
		logger.Warn("failed to load sites for dialog", logger.ErrAttr(err))
	}

	elements := []mattermost.DialogElement{
		{DisplayName: "Заголовок", Name: "title", Type: "text", MaxLength: 150},
		{DisplayName: "Описание", Name: "description", Type: "textarea", MaxLength: 3000},
	}

	if len(categories) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(categories))
		for _, c := range categories {
			opts = append(opts, &model.PostActionOptions{Text: c.Name, Value: c.ID.String()})
		}
		elements = append(elements, mattermost.DialogElement{
			DisplayName: "Категория", Name: "categoryId", Type: "select", Options: opts,
		})
	}

	if len(sites) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(sites))
		for _, st := range sites {
			opts = append(opts, &model.PostActionOptions{Text: st.Name, Value: st.ID.String()})
		}
		elements = append(elements, mattermost.DialogElement{
			DisplayName: "Площадка", Name: "siteId", Type: "select", Options: opts,
		})
	}

	if err := s.most.Dialog.Open(settings.BotToken, mattermost.OpenRequest{
		TriggerID:   triggerID,
		RealmID:     realmID.String(),
		Title:       "Новая заявка",
		SubmitLabel: "Создать",
		Elements:    elements,
		State:       fmt.Sprintf(`{"channelId":"%s","buttonPostId":"%s"}`, channelID, buttonPostID),
	}); err != nil {
		return fmt.Errorf("failed to open dialog: %w", err)
	}
	return nil
}

// HandleDialogSubmission создаёт заявку из отправленного диалога: резолвит
// пользователя Mattermost в систему, прикрепляет загруженные файлы и удаляет
// исходный кнопочный пост. Отмена диалога обрабатывается отдельно (без создания).
func (s *MattermostService) HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error {
	if submission.Cancelled {
		return nil
	}

	realmID, err := uuid.Parse(submission.CallbackId)
	if err != nil {
		return fmt.Errorf("invalid callback_id: %w", err)
	}

	var (
		creatorID   uuid.UUID
		creatorName string
		siteID      *uuid.UUID
	)
	if rawSite, ok := submission.Submission["siteId"].(string); ok && rawSite != "" {
		if id, err := uuid.Parse(rawSite); err == nil {
			siteID = &id
		}
	}

	creatorID, creatorName, err = s.resolveOrCreateUser(ctx, realmID, submission.UserId, siteID)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	dto := &models.TicketDTO{
		Title:     submission.Submission["title"].(string),
		Status:    models.StatusOpen,
		RealmID:   &realmID,
		CreatorID: creatorID,
		Actor:     &models.Actor{ID: creatorID, Name: creatorName},
	}

	if desc, ok := submission.Submission["description"].(string); ok && desc != "" {
		dto.Description = desc
	}
	if catID, ok := submission.Submission["categoryId"].(string); ok && catID != "" {
		id, err := uuid.Parse(catID)
		if err == nil {
			dto.CategoryID = id

			cat, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: id, RealmID: realmID})
			if err == nil {
				groupID := cat.GroupID
				dto.GroupID = &groupID
				if cat.Priority != "" {
					dto.Priority = cat.Priority
				}
			}
		}
	}

	if dto.GroupID != nil {
		group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
		if err == nil {
			if dto.AssigneeID == nil && group.DefaultAssigneeID != nil {
				dto.AssigneeID = group.DefaultAssigneeID
			}
			if dto.ManagerID == nil && group.ManagerID != nil {
				dto.ManagerID = group.ManagerID
			}
		}
	}
	if siteID != nil {
		dto.SiteID = *siteID
	}

	err = s.tickets.Create(ctx, dto)
	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	logger.Info("ticket created from mattermost",
		logger.StringAttr("ticket_id", dto.ID.String()),
		logger.StringAttr("mm_user", submission.UserId),
	)

	s.recentTickets.Store(submission.UserId+":"+submission.ChannelId, &recentTicket{
		id:        *dto.ID,
		createdAt: time.Now(),
	})

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err == nil {
		s.sendTicketCreatedDM(settings, submission.UserId, dto)
		s.deleteButtonPost(settings, submission)
		s.processPendingFiles(ctx, settings.BotToken, submission, dto)
	}

	return nil
}

// HandleInteractiveAction обрабатывает нажатия интерактивных кнопок Mattermost
// и возвращает пост-ответ (или nil, если ответ не нужен). Для «view_ticket»
// формирует кнопку «Открыть», ведущую на страницу заявки во фронтенде.
func (s *MattermostService) HandleInteractiveAction(ctx context.Context, userID, channelID string, actionContext map[string]string) (*model.Post, error) {
	if s.baseURL == "" {
		return nil, fmt.Errorf("failed to handle action: http.base_url is not configured")
	}

	action := actionContext["action"]

	switch action {
	case "view_ticket":
		ticketID := actionContext["ticket_id"]
		realmID := actionContext["realm_id"]
		return s.most.Post.Reply(
			fmt.Sprintf("Откройте заявку: /tasks/%s", ticketID),
			&mattermost.InteractiveButton{
				Text:  "Открыть",
				Style: "primary",
				URL:   fmt.Sprintf("%s/api/v1/mattermost/action", s.baseURL),
				Context: map[string]string{
					"action":    "open_link",
					"ticket_id": ticketID,
					"realm_id":  realmID,
				},
			},
		), nil

	default:
		return nil, nil
	}
}

// sendTicketCreatedDM отправляет пользователю личное сообщение с подтверждением
// создания заявки и ссылкой на неё. Ошибки отправки не фатальны — заявка уже
// создана, поэтому проблема лишь логируется.
func (s *MattermostService) sendTicketCreatedDM(settings *models.RealmMattermost, mmUserID string, dto *models.TicketDTO) {
	if settings.BotToken == "" {
		return
	}
	msg := fmt.Sprintf("Заявка №%d создана.\nЗаголовок: %s",
		dto.TicketNumber, dto.Title)
	if s.baseURL != "" {
		msg += fmt.Sprintf("\nОткрыть: %s/tasks/%s", s.baseURL, dto.ID.String())
	}
	msg += "\nОтправьте файлы следующим сообщением в течение 30 минут — они прикрепятся автоматически к этой заявке. Или можете указать номер заявки (например, №123) вместе с файлами."
	if err := s.most.DM.Send(settings.BotToken, settings.BotUserID, mmUserID, msg); err != nil {
		logger.Warn("failed to send ticket created DM", logger.ErrAttr(err))
	}
}

// deleteButtonPost удаляет исходный пост с кнопкой «Создать заявку» после того,
// как диалог был успешно отправлен. Состояние (id поста кнопки) передаётся
// в диалог заранее через State; если id отсутствует, пост не трогаем.
func (s *MattermostService) deleteButtonPost(settings *models.RealmMattermost, submission *model.SubmitDialogRequest) {
	if settings.BotToken == "" || submission.State == "" {
		return
	}
	var state struct {
		ButtonPostID string `json:"buttonPostId"`
	}
	if err := json.Unmarshal([]byte(submission.State), &state); err != nil || state.ButtonPostID == "" {
		return
	}
	if err := s.most.Post.Delete(settings.BotToken, state.ButtonPostID); err != nil {
		logger.Warn("failed to delete button post", logger.ErrAttr(err))
	}
}

// processPendingFiles прикрепляет файлы, загруженные пользователем вместе с
// сообщением-командой, к только что созданной заявке. Ключ — канал+пользователь,
// т.к. Mattermost не связывает файлы загруженные до диалога напрямую. Каждый
// файл скачивается отдельно, ошибки не прерывают обработку остальных.
func (s *MattermostService) processPendingFiles(ctx context.Context, botToken string, submission *model.SubmitDialogRequest, dto *models.TicketDTO) {
	key := submission.ChannelId + ":" + submission.UserId
	fileIDsVal, ok := s.pendingFiles.LoadAndDelete(key)
	if !ok {
		logger.Debug("no pending files found", logger.StringAttr("key", key))
		return
	}
	fileIDs, ok := fileIDsVal.([]string)
	if !ok || len(fileIDs) == 0 {
		return
	}

	logger.Info("processing pending files",
		logger.StringAttr("ticket_id", dto.ID.String()),
		logger.IntAttr("count", len(fileIDs)),
	)

	for _, fileID := range fileIDs {
		data, err := s.most.Client.DownloadFile(botToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}

		info, err := s.most.Client.GetFileInfo(botToken, fileID)
		if err != nil {
			logger.Warn("failed to get MM file info",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}

		fileName := info.Name
		if fileName == "" {
			fileName = fileID
		}

		att, err := s.attachments.Upload(ctx, nil, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   *dto.ID,
			FileName:   fileName,
			FileSize:   info.Size,
			MimeType:   info.MimeType,
			File:       bytes.NewReader(data),
			UploadedBy: dto.CreatorID,
			Realm:      dto.RealmID.String(),
		})
		if err != nil {
			logger.Warn("failed to upload MM file as attachment",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}
		logger.Info("attachment created from MM file",
			logger.StringAttr("ticket_id", dto.ID.String()),
			logger.StringAttr("attachment_id", att.ID.String()),
			logger.StringAttr("file_name", fileName),
		)
	}
}
