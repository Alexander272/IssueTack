package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// PluginContextResult — данные контекста канала для webapp-плагина MM:
// реалм, связанный с каналом, резолвнутый системный пользователь и
// справочники (категории, площадки) для формы создания заявки.
// Bound=false означает, что канал не привязан к реалму (или привязка
// неактивна): нормальное состояние, а не ошибка.
type PluginContextResult struct {
	Bound      bool               `json:"bound"`
	RealmID    uuid.UUID          `json:"realmId"`
	RealmName  string             `json:"realmName"`
	User       PluginUser         `json:"user"`
	Categories []*models.Category `json:"categories"`
	Sites      []*models.Site     `json:"sites"`
}

// PluginUser — краткое представление системного пользователя для плагина.
type PluginUser struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	SiteID   *string   `json:"siteId,omitempty"`
}

// PluginFile — парснутый обработчиком файл из multipart-запроса плагина.
type PluginFile struct {
	FileName string
	FileSize int64
	MimeType string
	Content  io.Reader
}

// PluginCreateTicketInput — данные для создания заявки из плагина MM.
// CategoryID/SiteID равны uuid.Nil, если не выбраны.
type PluginCreateTicketInput struct {
	ChannelID   string
	MmUserID    string
	Title       string
	Description string
	CategoryID  uuid.UUID
	SiteID      uuid.UUID
	Files       []PluginFile
}

// PluginCreateTicketResult — результат создания заявки из плагина.
type PluginCreateTicketResult struct {
	ID     uuid.UUID `json:"id"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	Link   string    `json:"link"`
}

// PluginTicketShort — строка списка «Мои активные заявки» для плагина.
type PluginTicketShort struct {
	ID          uuid.UUID             `json:"id"`
	Number      int                   `json:"number"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      models.TicketStatus   `json:"status"`
	Priority    models.Priority       `json:"priority"`
	CreatedAt   time.Time             `json:"createdAt"`
	Category    *models.CategoryShort `json:"category,omitempty"`
	Site        *models.SiteShort     `json:"site,omitempty"`
	Link        string                `json:"link"`
}

// PluginTicketDetail — компактная карточка заявки для показа в модалке
// плагина MM (без подзадач/вложений/комментариев).
type PluginTicketDetail struct {
	ID          uuid.UUID             `json:"id"`
	Number      int                   `json:"number"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      models.TicketStatus   `json:"status"`
	Priority    models.Priority       `json:"priority"`
	CreatedAt   time.Time             `json:"createdAt"`
	DueDate     *time.Time            `json:"dueDate,omitempty"`
	Category    *models.CategoryShort `json:"category,omitempty"`
	Site        *models.SiteShort     `json:"site,omitempty"`
	Creator     models.UserShort      `json:"creator"`
	Owner       *models.UserShort     `json:"owner,omitempty"`
	Assignee    *models.UserShort     `json:"assignee,omitempty"`
	Link        string                `json:"link"`
	// Attachments — вложенные к заявке файлы (не к комментариям).
	Attachments []*PluginAttachment `json:"attachments,omitempty"`
	// Флаги действий текущего пользователя (формулы те же, что в InfoBar
	// веб-приложения): CanConfirm/CanReopen — владелец и статус resolved,
	// CanCancel — владелец и статус open (отменять можно только новые заявки).
	CanConfirm bool `json:"canConfirm,omitempty"`
	CanReopen  bool `json:"canReopen,omitempty"`
	CanCancel  bool `json:"canCancel,omitempty"`
}

// PluginAttachment — вложение заявки для плагина. Ссылка на скачивание строится
// в webapp-плагине из id (через plugin-proxy, минуя авторизацию программы).
type PluginAttachment struct {
	ID       uuid.UUID `json:"id"`
	FileName string    `json:"fileName"`
	FileSize int64     `json:"fileSize"`
	MimeType string    `json:"mimeType"`
}

// PluginComment — общедоступный комментарий к заявке для плагина.
type PluginComment struct {
	ID          uuid.UUID                 `json:"id"`
	Text        string                    `json:"text"`
	CreatedAt   time.Time                 `json:"createdAt"`
	User        *PluginCommentUser        `json:"user"`
	Attachments []*PluginCommentAttachment `json:"attachments,omitempty"`
}

type PluginCommentUser struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PluginCommentAttachment struct {
	ID       uuid.UUID `json:"id"`
	FileName string    `json:"fileName"`
	FileSize int64     `json:"fileSize"`
	MimeType string    `json:"mimeType"`
}

// PluginCreateCommentInput — входящие данные комментария из плагина.
type PluginCreateCommentInput struct {
	ChannelID string
	MmUserID  string
	TicketID  string
	Text      string
	Files     []PluginFile
}

func pluginUserName(u models.UserShort) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return u.Username
	}
	return name
}

func (s *MattermostService) pluginAttachmentDTO(a *models.Attachment) *PluginAttachment {
	if a == nil {
		return nil
	}
	return &PluginAttachment{
		ID:       a.ID,
		FileName: a.FileName,
		FileSize: a.FileSize,
		MimeType: a.MimeType,
	}
}

// PluginGetTicket возвращает заявку по ID с проверкой права чтения.
// Пользователь, не имеющий доступа к заявке, получает ErrPermissionDenied.
func (s *MattermostService) PluginGetTicket(ctx context.Context, channelID, mmUserID, ticketID string) (*PluginTicketDetail, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
	}

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

	detail := &PluginTicketDetail{
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

	isOwner := ticket.Owner != nil && ticket.Owner.ID == user.ID
	detail.CanConfirm = isOwner && ticket.Status == models.StatusResolved
	detail.CanReopen = isOwner && ticket.Status == models.StatusResolved
	detail.CanCancel = isOwner && ticket.Status == models.StatusOpen

	return detail, nil
}

// PluginChangeStatus меняет статус заявки из плагина MM. Используется для
// действий владельца: «Подтвердить решение» (resolved → closed), «Вернуть в
// работу» (resolved → in_progress), «Отменить заявку» (активный → cancelled).
// Все правила перехода и прав доступа (canChangeStatus / ownerTransitionAllowed,
// терминальные статусы, закрытие только из resolved и т.д.) применяет
// TicketService.Update.
func (s *MattermostService) PluginChangeStatus(ctx context.Context, channelID, mmUserID, ticketID, status string) error {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return models.ErrChannelNotBound
	}

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

	dto := &models.TicketDTO{
		ID:      &id,
		Status:  next,
		Actor:   &models.Actor{ID: user.ID, Name: user.Username},
		RealmID: &settings.RealmID,
	}
	dto.MarkProvided("status")

	return s.tickets.Update(ctx, dto)
}

// PluginGetComments возвращает общедоступные комментарии заявки (в порядке
// создания — как диалог). Внутренние комментарии плагину не отдаются.
func (s *MattermostService) PluginGetComments(ctx context.Context, channelID, mmUserID, ticketID string) ([]PluginComment, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
	}

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

	out := make([]PluginComment, 0, len(comments))
	for _, c := range comments {
		if c == nil || c.IsInternal {
			continue
		}
		item := PluginComment{
			ID:        c.ID,
			Text:      c.Text,
			CreatedAt: c.CreatedAt,
		}
		if c.User != nil {
			item.User = &PluginCommentUser{ID: c.User.ID, Name: pluginUserName(*c.User)}
		}
		for _, att := range c.Attachments {
			if att == nil {
				continue
			}
			item.Attachments = append(item.Attachments, &PluginCommentAttachment{
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
func (s *MattermostService) PluginCreateComment(ctx context.Context, input *PluginCreateCommentInput) (*PluginComment, error) {
	settings, err := s.repo.GetByChannelID(ctx, input.ChannelID)
	if err != nil {
		return nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
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

	return &PluginComment{
		ID:        comment.ID,
		Text:      text,
		CreatedAt: comment.CreatedAt,
		User:      &PluginCommentUser{ID: user.ID, Name: pluginUserName(models.UserShort{ID: user.ID, Username: user.Username, FirstName: user.FirstName, LastName: user.LastName})},
	}, nil
}

// PluginGetAttachmentContent возвращает вложение заявки и поток его файла
// (доступ проверяется внутри AttachmentService.GetContent).
func (s *MattermostService) PluginGetAttachmentContent(ctx context.Context, channelID, mmUserID, attachmentID string) (*models.Attachment, io.ReadCloser, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, nil, models.ErrChannelNotBound
	}

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

// PluginContext возвращает контекст канала плагина: реалм по каналу,
// справочники и пользователя (с автосозданием при первом обращении).
// Канал, не привязанный ни к одному реалму (или с неактивной привязкой), —
// не ошибка: возвращается ответ с Bound=false, чтобы webapp просто скрыл иконку.
func (s *MattermostService) PluginContext(ctx context.Context, channelID, mmUserID string) (*PluginContextResult, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return &PluginContextResult{Bound: false}, nil
	}
	if !settings.IsActive {
		return &PluginContextResult{Bound: false}, nil
	}

	realm, err := s.realms.GetByID(ctx, &models.GetRealmByIdDTO{ID: settings.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get realm: %w", err)
	}

	categories, err := s.categories.Get(ctx, &models.GetCategoriesDTO{RealmID: settings.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	sites, err := s.sites.Get(ctx, &models.GetSitesDTO{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sites: %w", err)
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	return &PluginContextResult{
		Bound:      true,
		RealmID:    settings.RealmID,
		RealmName:  realm.Name,
		User:       PluginUser{ID: user.ID, Username: user.Username, SiteID: user.SiteID},
		Categories: categories,
		Sites:      sites,
	}, nil
}

// PluginCreateTicket создаёт заявку из формы плагина (аналог
// HandleDialogSubmission) и прикрепляет загруженные файлы. Возвращает результат
// с ID, номером и ссылкой на страницу заявки.
func (s *MattermostService) PluginCreateTicket(ctx context.Context, input *PluginCreateTicketInput) (*PluginCreateTicketResult, error) {
	settings, err := s.repo.GetByChannelID(ctx, input.ChannelID)
	if err != nil {
		return nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
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

	result := &PluginCreateTicketResult{
		ID:    *dto.ID,
		Title: dto.Title,
	}
	if dto.TicketNumber > 0 {
		result.Number = dto.TicketNumber
	}
	if s.baseURL != "" {
		result.Link = fmt.Sprintf("%s/tasks/%s", s.baseURL, dto.ID)
	}

	return result, nil
}

// PluginListMine возвращает активные заявки пользователя в реалме канала —
// созданные им и ещё не завершённые (open/in_progress/pending/on_hold).
func (s *MattermostService) PluginListMine(ctx context.Context, channelID, mmUserID string) ([]PluginTicketShort, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, models.ErrChannelNotBound
	}
	if !settings.IsActive {
		return nil, models.ErrChannelNotBound
	}

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

	list := make([]PluginTicketShort, 0, len(tickets))
	for _, t := range tickets {
		item := PluginTicketShort{
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
		list = append(list, item)
	}

	return list, nil
}
