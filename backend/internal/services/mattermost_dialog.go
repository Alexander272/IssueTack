package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

type recentTicket struct {
	id        uuid.UUID
	createdAt time.Time
}

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

	elements := []model.DialogElement{
		{
			DisplayName: "Заголовок",
			Name:        "title",
			Type:        "text",
			MaxLength:   150,
		},
		{
			DisplayName: "Описание",
			Name:        "description",
			Type:        "textarea",
			MaxLength:   3000,
		},
	}

	if len(categories) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(categories))
		for _, c := range categories {
			opts = append(opts, &model.PostActionOptions{Text: c.Name, Value: c.ID.String()})
		}
		elements = append(elements, model.DialogElement{
			DisplayName: "Категория",
			Name:        "categoryId",
			Type:        "select",
			Options:     opts,
		})
	}

	if len(sites) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(sites))
		for _, st := range sites {
			opts = append(opts, &model.PostActionOptions{Text: st.Name, Value: st.ID.String()})
		}
		elements = append(elements, model.DialogElement{
			DisplayName: "Площадка",
			Name:        "siteId",
			Type:        "select",
			Options:     opts,
		})
	}

	if err := s.client.OpenDialog(settings.BotToken, &model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       fmt.Sprintf("%s/api/v1/mattermost/dialog/%s", s.baseURL, realmID),
		Dialog: model.Dialog{
			CallbackId:     realmID.String(),
			Title:          "Новая заявка",
			Elements:       elements,
			SubmitLabel:    "Создать",
			NotifyOnCancel: true,
			State:          fmt.Sprintf(`{"channelId":"%s","buttonPostId":"%s"}`, channelID, buttonPostID),
		},
	}); err != nil {
		return fmt.Errorf("failed to open dialog: %w", err)
	}
	return nil
}

func (s *MattermostService) HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error {
	if submission.Cancelled {
		return nil
	}

	realmID, err := uuid.Parse(submission.CallbackId)
	if err != nil {
		return fmt.Errorf("invalid callback_id: %w", err)
	}

	creatorID, creatorName, err := s.resolveOrCreateUser(ctx, realmID, submission.UserId)
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
	if siteID, ok := submission.Submission["siteId"].(string); ok && siteID != "" {
		id, err := uuid.Parse(siteID)
		if err == nil {
			dto.SiteID = id
		}
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
		s.updateButtonPost(settings, submission, dto)
		s.processPendingFiles(ctx, settings.BotToken, submission, dto)
	}

	return nil
}

func (s *MattermostService) HandleInteractiveAction(ctx context.Context, userID, channelID string, actionContext map[string]string) (*model.Post, error) {
	if s.baseURL == "" {
		return nil, fmt.Errorf("failed to handle action: http.base_url is not configured")
	}

	action := actionContext["action"]

	switch action {
	case "view_ticket":
		ticketID := actionContext["ticket_id"]
		realmID := actionContext["realm_id"]
		return &model.Post{
			Message: fmt.Sprintf("Откройте заявку: /tasks/%s", ticketID),
			Props: model.StringInterface{
				"attachments": []model.StringInterface{{
					"text": fmt.Sprintf("Заявка #%s", ticketID[:8]),
					"actions": []model.StringInterface{{
						"name":  "Открыть",
						"type":  "button",
						"style": "primary",
						"integration": model.StringInterface{
							"url": fmt.Sprintf("%s/api/v1/mattermost/action", s.baseURL),
							"context": map[string]string{
								"action":    "open_link",
								"ticket_id": ticketID,
								"realm_id":  realmID,
							},
						},
					}},
				}},
			},
		}, nil

	default:
		return nil, nil
	}
}

func (s *MattermostService) sendTicketCreatedDM(settings *models.RealmMattermost, mmUserID string, dto *models.TicketDTO) {
	if settings.BotToken == "" {
		return
	}
	msg := fmt.Sprintf("Заявка #%s создана.\nЗаголовок: %s\nСтатус: %s",
		dto.ID.String()[:8], dto.Title, dto.Status)
	if s.baseURL != "" {
		msg += fmt.Sprintf("\nОткрыть: %s/tasks/%s", s.baseURL, dto.ID.String())
	}
	if _, err := s.client.SendDM(settings.BotToken, settings.BotUserID, mmUserID, msg); err != nil {
		logger.Warn("failed to send ticket created DM", logger.ErrAttr(err))
	}
}

func (s *MattermostService) updateButtonPost(settings *models.RealmMattermost, submission *model.SubmitDialogRequest, dto *models.TicketDTO) {
	if settings.BotToken == "" || submission.State == "" {
		return
	}
	var state struct {
		ChannelID    string `json:"channelId"`
		ButtonPostID string `json:"buttonPostId"`
	}
	if err := json.Unmarshal([]byte(submission.State), &state); err != nil || state.ButtonPostID == "" {
		return
	}
	link := ""
	if s.baseURL != "" {
		link = fmt.Sprintf("\nОткрыть: %s/tasks/%s", s.baseURL, dto.ID.String())
	}
	msg := fmt.Sprintf("Заявка #%s создана.%s", dto.ID.String()[:8], link)
	patch := &model.PostPatch{
		Message: &msg,
	}
	if err := s.client.UpdatePost(settings.BotToken, state.ButtonPostID, patch); err != nil {
		logger.Warn("failed to update button post", logger.ErrAttr(err))
	}
}

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
		data, err := s.client.DownloadFile(botToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}

		info, err := s.client.GetFileInfo(botToken, fileID)
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
