package services

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// pluginDeepLink возвращает site-relative ссылку на заявку внутри плагина.
// Абсолютный URL здесь не используется намеренно: site URL сервера MM
// бэкенду неизвестен (см. models.RealmMattermost), а относительный путь
// Mattermost помечает как data-link и открывает через SPA-роутер.
func pluginDeepLink(ticketID uuid.UUID) string {
	return fmt.Sprintf("%s/ticket/%s", models.PluginRoutePrefix, ticketID)
}

func pluginUserName(u models.UserShort) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return u.Username
	}
	return name
}

func (s *MattermostService) pluginAttachmentDTO(a *models.Attachment) *models.PluginAttachment {
	if a == nil {
		return nil
	}
	return &models.PluginAttachment{
		ID:       a.ID,
		FileName: a.FileName,
		FileSize: a.FileSize,
		MimeType: a.MimeType,
	}
}

// resolvePluginSettings определяет настройки интеграции Mattermost по контексту
// запроса плагина. Обычный канал ищется по привязке channel_id, а личный диалог
// с ботом (scope.BotUserID) — по боту реалма, с которым переписывается
// пользователь: такие каналы привязки не имеют. Для диалога дополнительно
// проверяется состав участников, т.к. BotUserID приходит от клиента.
// Положительный результат кэшируется на минуту, чтобы шапка канала не била в
// БД и Mattermost API при каждом запросе.
func (s *MattermostService) resolvePluginSettings(ctx context.Context, scope models.PluginScope) (*models.RealmMattermost, error) {
	cacheKey := pluginScopeCacheKey(scope)
	if settings, ok := s.pluginCache.getSettings(cacheKey); ok {
		return settings, nil
	}
	settings, err := s.resolvePluginSettingsUncached(ctx, scope)
	if err != nil {
		return nil, err
	}
	s.pluginCache.setSettings(cacheKey, settings)
	return settings, nil
}

func (s *MattermostService) resolvePluginSettingsUncached(ctx context.Context, scope models.PluginScope) (*models.RealmMattermost, error) {
	if scope.ChannelID != "" {
		if settings, err := s.repo.GetByChannelID(ctx, scope.ChannelID); err == nil {
			if !settings.IsActive {
				return nil, models.ErrChannelNotBound
			}
			return settings, nil
		}
	}
	if scope.BotUserID != "" {
		settings, err := s.repo.GetByBotUserID(ctx, scope.BotUserID)
		if err != nil {
			return nil, models.ErrChannelNotBound
		}
		if !settings.IsActive {
			return nil, models.ErrChannelNotBound
		}
		if err := s.verifyDMScope(ctx, settings, scope); err != nil {
			return nil, err
		}
		return settings, nil
	}
	return nil, models.ErrChannelNotBound
}

// verifyDMScope подтверждает, что канал из scope — именно личный диалог бота
// реалма с заявителем: в нём состоят ровно эти двое. Проверка защищает от
// подстановки чужого botUserId или userId в запрос плагина.
func (s *MattermostService) verifyDMScope(ctx context.Context, settings *models.RealmMattermost, scope models.PluginScope) error {
	if s.most == nil || s.most.Client == nil {
		return models.ErrChannelNotBound
	}
	members, err := s.most.Client.GetChannelMemberIDs(settings.BotToken, scope.ChannelID, settings.BotUserID, scope.MmUserID)
	if err != nil {
		logger.Warn("failed to verify plugin dm scope", logger.ErrAttr(err))
		return models.ErrChannelNotBound
	}
	if len(members) != 2 {
		return models.ErrChannelNotBound
	}
	return nil
}

// PluginGetTicket возвращает заявку по ID с проверкой права чтения.
// Пользователь, не имеющий доступа к заявке, получает ErrPermissionDenied.
func (s *MattermostService) PluginGetTicket(ctx context.Context, scope models.PluginScope, ticketID string) (*models.PluginTicketDetail, error) {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return nil, err
	}
	mmUserID := scope.MmUserID

	id, err := uuid.Parse(ticketID)
	if err != nil {
		return nil, models.ErrInvalidInput
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	ticket, err := s.tickets.GetByID(ctx, &models.GetTicketByIdDTO{
		ID:      id,
		Actor:   &models.Actor{ID: user.ID, Name: user.Username},
		RealmID: settings.RealmID.String(),
	})
	if err != nil {
		return nil, err
	}

	detail := &models.PluginTicketDetail{
		ID:          ticket.ID,
		Title:       ticket.Title,
		Description: ticket.Description,
		Status:      ticket.Status,
		Priority:    ticket.Priority,
		CreatedAt:   ticket.CreatedAt,
		DueDate:     ticket.DueDate,
		Category:    ticket.Category,
		Site:        ticket.Site,
		Creator:     ticket.Creator,
		Owner:       ticket.Owner,
		Assignee:    ticket.Assignee,
	}
	if ticket.TicketNumber != nil {
		detail.Number = *ticket.TicketNumber
	}
	for _, att := range ticket.Attachments {
		if att == nil || att.CommentID != nil {
			continue
		}
		detail.Attachments = append(detail.Attachments, s.pluginAttachmentDTO(att))
	}
	if s.baseURL != "" {
		detail.Link = fmt.Sprintf("%s/tasks/%s", s.baseURL, ticket.ID)
	}
	detail.DeepLink = pluginDeepLink(ticket.ID)

	isOwner := ticket.Owner != nil && ticket.Owner.ID == user.ID
	detail.CanConfirm = isOwner && ticket.Status == models.StatusResolved
	detail.CanReopen = isOwner && ticket.Status == models.StatusResolved
	detail.CanCancel = isOwner && ticket.Status == models.StatusOpen

	return detail, nil
}

// PluginGetTicketLinkContext отдаёт заявку для страницы, открытой по deep-link
// ссылке вида /plug/issuetrack/ticket/<uuid>. В отличие от PluginGetTicket
// канал не принимается: реалм определяется по самой заявке, из него берётся
// привязанный канал, который возвращается клиенту для остальных запросов.
// Проверка прав не дублируется — её выполняет PluginGetTicket внутри.
func (s *MattermostService) PluginGetTicketLinkContext(ctx context.Context, mmUserID, ticketID string) (*models.PluginTicketLinkContext, error) {
	id, err := uuid.Parse(ticketID)
	if err != nil {
		return nil, models.ErrInvalidInput
	}

	realmID, err := s.tickets.GetRealmIDByTicketID(ctx, id)
	if err != nil {
		return nil, err
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return nil, err
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
	}

	detail, err := s.PluginGetTicket(ctx, models.PluginScope{
		ChannelID: settings.ChannelID,
		MmUserID:  mmUserID,
	}, ticketID)
	if err != nil {
		return nil, err
	}

	return &models.PluginTicketLinkContext{
		ChannelID: settings.ChannelID,
		Detail:    detail,
	}, nil
}

// PluginChangeStatus меняет статус заявки из плагина MM. Используется для
// действий владельца: «Подтвердить решение» (resolved → closed), «Вернуть в
// работу» (resolved → in_progress), «Отменить заявку» (активный → cancelled).
// Все правила перехода и прав доступа (canChangeStatus / ownerTransitionAllowed,
// терминальные статусы, закрытие только из resolved и т.д.) применяет
// TicketService.Update.
func (s *MattermostService) PluginChangeStatus(ctx context.Context, scope models.PluginScope, ticketID, status string) error {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return err
	}
	mmUserID := scope.MmUserID

	id, err := uuid.Parse(ticketID)
	if err != nil {
		return models.ErrInvalidInput
	}

	next := models.TicketStatus(status)
	if !next.IsValid() {
		return models.ErrInvalidInput
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	return s.changeTicketStatus(ctx, user, settings.RealmID, id, next)
}

// PluginGetComments возвращает общедоступные комментарии заявки (в порядке
// создания — как диалог). Внутренние комментарии плагину не отдаются.
func (s *MattermostService) PluginGetComments(ctx context.Context, scope models.PluginScope, ticketID string) ([]models.PluginComment, error) {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return nil, err
	}
	mmUserID := scope.MmUserID

	id, err := uuid.Parse(ticketID)
	if err != nil {
		return nil, models.ErrInvalidInput
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	comments, err := s.comments.GetByTicket(ctx, id, user.ID, settings.RealmID.String())
	if err != nil {
		return nil, err
	}

	out := make([]models.PluginComment, 0, len(comments))
	for _, c := range comments {
		if c == nil || c.IsInternal {
			continue
		}
		item := models.PluginComment{
			ID:        c.ID,
			Text:      c.Text,
			CreatedAt: c.CreatedAt,
		}
		if c.User != nil {
			item.User = &models.PluginCommentUser{ID: c.User.ID, Name: pluginUserName(*c.User)}
		}
		for _, att := range c.Attachments {
			if att == nil {
				continue
			}
			item.Attachments = append(item.Attachments, &models.PluginCommentAttachment{
				ID:       att.ID,
				FileName: att.FileName,
				FileSize: att.FileSize,
				MimeType: att.MimeType,
			})
		}
		out = append(out, item)
	}

	return out, nil
}

// PluginCreateComment создаёт общедоступный комментарий к заявке (проверка
// work-доступа и файлы — через CommentService.Create, атомарно). Требуется
// текст или хотя бы один файл.
func (s *MattermostService) PluginCreateComment(ctx context.Context, input *models.PluginCreateCommentInput) (*models.PluginComment, error) {
	settings, err := s.resolvePluginSettings(ctx, input.PluginScope)
	if err != nil {
		return nil, err
	}

	ticketID, err := uuid.Parse(input.TicketID)
	if err != nil {
		return nil, models.ErrInvalidInput
	}

	text := strings.TrimSpace(input.Text)
	if text == "" && len(input.Files) == 0 {
		return nil, models.ErrInvalidInput
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, input.MmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	files := make([]*models.UploadAttachmentDTO, 0, len(input.Files))
	for i := range input.Files {
		f := input.Files[i]
		files = append(files, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   ticketID,
			FileName:   f.FileName,
			FileSize:   f.FileSize,
			MimeType:   f.MimeType,
			File:       f.Content,
			UploadedBy: user.ID,
			Realm:      settings.RealmID.String(),
		})
	}

	comment, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       text,
		TicketID:   ticketID,
		IsInternal: false,
		Type:       "",
		UserID:     user.ID,
		Realm:      settings.RealmID.String(),
		Files:      files,
	})
	if err != nil {
		return nil, err
	}

	return &models.PluginComment{
		ID:        comment.ID,
		Text:      text,
		CreatedAt: comment.CreatedAt,
		User:      &models.PluginCommentUser{ID: user.ID, Name: pluginUserName(models.UserShort{ID: user.ID, Username: user.Username, FirstName: user.FirstName, LastName: user.LastName})},
	}, nil
}

// PluginGetAttachmentContent возвращает вложение заявки и поток его файла
// (доступ проверяется внутри AttachmentService.GetContent).
func (s *MattermostService) PluginGetAttachmentContent(ctx context.Context, scope models.PluginScope, attachmentID string) (*models.Attachment, io.ReadCloser, error) {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return nil, nil, err
	}
	mmUserID := scope.MmUserID

	id, err := uuid.Parse(attachmentID)
	if err != nil {
		return nil, nil, models.ErrInvalidInput
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	return s.attachments.GetContent(ctx, id, user.ID, settings.RealmID.String())
}

// PluginContext возвращает контекст канала плагина: реалм (по привязке канала
// либо по боту личного диалога), справочники и пользователя (с автосозданием
// при первом обращении). Канал, не привязанный ни к одному реалму (или с
// неактивной привязкой), — не ошибка: возвращается ответ с Bound=false, чтобы
// webapp просто скрыл иконку.
func (s *MattermostService) PluginContext(ctx context.Context, scope models.PluginScope) (*models.PluginContextResult, error) {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return &models.PluginContextResult{Bound: false}, nil
	}

	realmName, categories, sites, err := s.pluginRealmData(ctx, settings.RealmID)
	if err != nil {
		return nil, err
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, scope.MmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	return &models.PluginContextResult{
		Bound:      true,
		RealmID:    settings.RealmID,
		RealmName:  realmName,
		User:       models.PluginUser{ID: user.ID, Username: user.Username, SiteID: user.SiteID},
		Categories: categories,
		Sites:      sites,
	}, nil
}

// pluginRealmData отдаёт название реалма и его общие справочники (категории,
// площадки). Они одинаковы для всех пользователей реалма, поэтому кэшируются,
// чтобы всплеск /plugin/context не перечитывал их из БД на каждый запрос.
func (s *MattermostService) pluginRealmData(ctx context.Context, realmID uuid.UUID) (string, []*models.Category, []*models.Site, error) {
	if ent, ok := s.pluginCache.getRealm(realmID); ok {
		return ent.realmName, ent.categories, ent.sites, nil
	}

	realm, err := s.realms.GetByID(ctx, &models.GetRealmByIdDTO{ID: realmID})
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get realm: %w", err)
	}

	categories, err := s.categories.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get categories: %w", err)
	}

	sites, err := s.sites.Get(ctx, &models.GetSitesDTO{})
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get sites: %w", err)
	}

	s.pluginCache.setRealm(realmID, realm.Name, categories, sites)
	return realm.Name, categories, sites, nil
}

// PluginCreateTicket создаёт заявку из формы плагина (аналог
// HandleDialogSubmission) и прикрепляет загруженные файлы. Возвращает результат
// с ID, номером и ссылкой на страницу заявки.
func (s *MattermostService) PluginCreateTicket(ctx context.Context, input *models.PluginCreateTicketInput) (*models.PluginCreateTicketResult, error) {
	settings, err := s.resolvePluginSettings(ctx, input.PluginScope)
	if err != nil {
		return nil, err
	}

	var siteID *uuid.UUID
	if input.SiteID != uuid.Nil {
		siteID = &input.SiteID
	}

	creator, err := s.resolveOrCreateUser(ctx, settings.RealmID, input.MmUserID, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	dto := &models.TicketDTO{
		Title:     input.Title,
		Status:    models.StatusOpen,
		RealmID:   &settings.RealmID,
		CreatorID: creator.ID,
		Actor:     &models.Actor{ID: creator.ID, Name: creator.Username},
	}
	if input.Description != "" {
		dto.Description = input.Description
	}

	if input.CategoryID != uuid.Nil {
		dto.CategoryID = input.CategoryID
		cat, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: input.CategoryID, RealmID: settings.RealmID})
		if err != nil {
			return nil, fmt.Errorf("invalid category: %w", err)
		}
		groupID := cat.GroupID
		dto.GroupID = &groupID
		if cat.Priority != "" {
			dto.Priority = cat.Priority
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

	if err := s.tickets.Create(ctx, dto); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	logger.Info("ticket created from mm plugin",
		logger.StringAttr("ticket_id", dto.ID.String()),
		logger.StringAttr("mm_user", input.MmUserID),
		logger.StringAttr("channel_id", input.ChannelID),
		logger.IntAttr("files", len(input.Files)),
	)

	for _, f := range input.Files {
		att, err := s.tickets.UploadAttachment(ctx, nil, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   *dto.ID,
			FileName:   f.FileName,
			FileSize:   f.FileSize,
			MimeType:   f.MimeType,
			File:       f.Content,
			UploadedBy: dto.CreatorID,
			Realm:      dto.RealmID.String(),
		})
		if err != nil {
			logger.Warn("failed to upload plugin attachment",
				logger.StringAttr("ticket_id", dto.ID.String()),
				logger.StringAttr("file_name", f.FileName),
				logger.ErrAttr(err),
			)
			continue
		}
		logger.Info("attachment created from plugin file",
			logger.StringAttr("ticket_id", dto.ID.String()),
			logger.StringAttr("attachment_id", att.ID.String()),
			logger.StringAttr("file_name", f.FileName),
		)
	}

	result := &models.PluginCreateTicketResult{
		ID:    *dto.ID,
		Title: dto.Title,
	}
	if dto.TicketNumber > 0 {
		result.Number = dto.TicketNumber
	}
	if s.baseURL != "" {
		result.Link = fmt.Sprintf("%s/tasks/%s", s.baseURL, dto.ID)
	}
	result.DeepLink = pluginDeepLink(*dto.ID)

	return result, nil
}

// PluginListMine возвращает активные заявки пользователя в реалме канала —
// созданные им и ещё не завершённые (open/in_progress/pending/on_hold).
func (s *MattermostService) PluginListMine(ctx context.Context, scope models.PluginScope) ([]models.PluginTicketShort, error) {
	settings, err := s.resolvePluginSettings(ctx, scope)
	if err != nil {
		return nil, err
	}
	mmUserID := scope.MmUserID

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	statuses := []models.TicketStatus{
		models.StatusOpen,
		models.StatusInProgress,
		models.StatusPending,
		models.StatusOnHold,
		// resolved показываем владельцу, чтобы можно было «подтвердить решение»
		// или «вернуть в работу» прямо из плагина.
		models.StatusResolved,
	}
	mode := "created_or_owned"

	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		RealmID:  &settings.RealmID,
		Statuses: statuses,
		Actor:    &models.Actor{ID: user.ID, Name: user.Username},
		Mode:     &mode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}

	list := make([]models.PluginTicketShort, 0, len(tickets))
	for _, t := range tickets {
		item := models.PluginTicketShort{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			Priority:    t.Priority,
			CreatedAt:   t.CreatedAt,
			Category:    t.Category,
			Site:        t.Site,
		}
		if t.TicketNumber != nil {
			item.Number = *t.TicketNumber
		}
		if s.baseURL != "" {
			item.Link = fmt.Sprintf("%s/tasks/%s", s.baseURL, t.ID)
		}
		item.DeepLink = pluginDeepLink(t.ID)
		list = append(list, item)
	}

	return list, nil
}
