package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

var createCommands = regexp.MustCompile(`^(?:заявка|новая|ticket|создать|new)$`)
var syncCommands = regexp.MustCompile(`^(?:синхронизировать|sync)(?:\s+(.+))?$`)
var helpCommands = regexp.MustCompile(`^(?:помощь|help|команды)$`)
var statusCommands = regexp.MustCompile(`^(?:статус|status|мои|заявки|my)$`)
var attachCommands = regexp.MustCompile(`^[#№]?(\d+)$`)
var commentCommands = regexp.MustCompile(`^[#№]?(\d+)\s+(.+)$`)

// MattermostDeps — зависимости фасада Mattermost: сервисы, необходимые
// для оркестрации, и capability-уровень Most для отправки сообщений.
type MattermostDeps struct {
	Repo        repository.Mattermost
	Users       Users
	UserRealms  UserRealms
	Roles       Roles
	Tickets     Tickets
	Groups      Groups
	Categories  Categories
	Sites       Sites
	Attachments Attachments
	Comments    Comments
	EventBus    *events.PolicyEventManager
	Most        *mattermost.Most
	BaseURL     string
}

// MattermostService — тонкий фасад над Mattermost: обрабатывает команды из
// личных сообщений, диалоги создания заявок, интерактивные кнопки и веб-сокеты,
// делегируя низкоуровневую работу с Mattermost пакету pkg/mattermost.
type MattermostService struct {
	repo          repository.Mattermost
	users         Users
	userRealms    UserRealms
	roles         Roles
	tickets       Tickets
	groups        Groups
	categories    Categories
	sites         Sites
	attachments   Attachments
	comments      Comments
	eventBus      *events.PolicyEventManager
	most          *mattermost.Most
	baseURL       string
	wsClients     map[string]*mattermost.WSClient
	wsMu          sync.Mutex
	pendingFiles  sync.Map
	recentTickets sync.Map
}

// NewMattermostService создаёт фасад Mattermost с переданными зависимостями.
func NewMattermostService(deps *MattermostDeps) *MattermostService {
	return &MattermostService{
		repo:        deps.Repo,
		users:       deps.Users,
		userRealms:  deps.UserRealms,
		roles:       deps.Roles,
		tickets:     deps.Tickets,
		groups:      deps.Groups,
		categories:  deps.Categories,
		sites:       deps.Sites,
		attachments: deps.Attachments,
		comments:    deps.Comments,
		eventBus:    deps.EventBus,
		most:        deps.Most,
		baseURL:     deps.BaseURL,
		wsClients:   make(map[string]*mattermost.WSClient),
	}
}

// Mattermost — публичный интерфейс фасада Mattermost, используемый
// HTTP-обработчиками и главной точкой входа.
type Mattermost interface {
	GetSettings(ctx context.Context, realmID uuid.UUID) (*models.RealmMattermost, error)
	SaveSettings(ctx context.Context, realmID uuid.UUID, dto *models.RealmMattermostDTO) error
	DeleteSettings(ctx context.Context, realmID uuid.UUID) error

	HandleDM(ctx context.Context, input *HandleDMInput) error
	HandleDialogOpen(ctx context.Context, triggerID, userID, channelID, buttonPostID string, actionCtx map[string]string) error
	HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error
	HandleInteractiveAction(ctx context.Context, userID, channelID string, context map[string]string) (*model.Post, error)

	StartWSForRealm(ctx context.Context, realmID uuid.UUID) error
	StopWSForRealm(realmID uuid.UUID)
	StopAllWS()
	StartAllActiveWS(ctx context.Context)
}

// HandleDMInput — входящее личное сообщение от пользователя Mattermost
// (уже распарсенное обработчиком) вместе с загруженными файлами.
type HandleDMInput struct {
	MmUserID  string
	BotUserID string
	ChannelID string
	Message   string
	FileIDs   []string
	TriggerID string
}

// GetSettings возвращает настройки интеграции Mattermost для указанного realm.
func (s *MattermostService) GetSettings(ctx context.Context, realmID uuid.UUID) (*models.RealmMattermost, error) {
	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mattermost settings: %w", err)
	}
	return settings, nil
}

// SaveSettings проверяет валидность bot-токена, сохраняет настройки интеграции
// и запускает веб-сокет для realm.
func (s *MattermostService) SaveSettings(ctx context.Context, realmID uuid.UUID, dto *models.RealmMattermostDTO) error {
	botUser, err := s.most.Client.GetMe(dto.BotToken)
	if err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	settings := &models.RealmMattermost{
		RealmID:   realmID,
		BotToken:  dto.BotToken,
		BotUserID: botUser.ID,
		ChannelID: dto.ChannelID,
		IsActive:  true,
	}
	if err := s.repo.Upsert(ctx, nil, settings); err != nil {
		return fmt.Errorf("failed to save mattermost settings: %w", err)
	}

	if err := s.StartWSForRealm(ctx, realmID); err != nil {
		logger.Error("failed to start WS after save",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
	}

	return nil
}

// DeleteSettings останавливает веб-сокет и удаляет настройки интеграции для realm.
func (s *MattermostService) DeleteSettings(ctx context.Context, realmID uuid.UUID) error {
	s.StopWSForRealm(realmID)
	if err := s.repo.Delete(ctx, nil, realmID); err != nil {
		return fmt.Errorf("failed to delete mattermost settings: %w", err)
	}
	return nil
}

// HandleDM обрабатывает личное сообщение пользователя: диспетчеризует его
// на создание заявки, статус, помощь или синхронизацию.
func (s *MattermostService) HandleDM(ctx context.Context, input *HandleDMInput) error {
	var settings *models.RealmMattermost
	var err error

	if input.BotUserID != "" {
		settings, err = s.repo.GetByBotUserID(ctx, input.BotUserID)
	} else {
		settings, err = s.repo.GetByChannelID(ctx, input.ChannelID)
	}
	if err != nil {
		return fmt.Errorf("failed to find realm settings: %w", err)
	}

	msg := strings.TrimSpace(input.Message)

	switch {
	case syncCommands.MatchString(msg):
		return s.handleSync(ctx, settings, input.MmUserID, msg)

	case createCommands.MatchString(msg):
		if len(input.FileIDs) > 0 {
			key := input.ChannelID + ":" + input.MmUserID
			s.pendingFiles.Store(key, input.FileIDs)
			logger.Info("stored pending files for create command",
				logger.StringAttr("key", key),
				logger.IntAttr("count", len(input.FileIDs)),
			)
		}
		return s.sendCreateButton(settings.BotToken, input.ChannelID, settings.RealmID.String())

	case helpCommands.MatchString(msg):
		isAdmin := s.checkIsAdmin(ctx, settings.RealmID, input.MmUserID)
		return s.sendHelpMessage(settings.BotToken, input.ChannelID, isAdmin)

	case statusCommands.MatchString(msg):
		userID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, input.MmUserID, nil)
		if err != nil {
			return fmt.Errorf("failed to resolve user: %w", err)
		}
		return s.sendStatusMessage(ctx, settings, userID, input.ChannelID)

	case attachCommands.MatchString(msg) && len(input.FileIDs) > 0:
		parts := attachCommands.FindStringSubmatch(msg)
		number, _ := strconv.Atoi(parts[1])
		return s.handleAttachFiles(ctx, settings, input.MmUserID, input.ChannelID, number, input.FileIDs, "")

	case len(input.FileIDs) > 0 && !attachCommands.MatchString(msg):
		return s.handleTextWithFiles(ctx, settings, input.MmUserID, input.ChannelID, msg, input.FileIDs)

	case commentCommands.MatchString(msg):
		return s.handleComment(ctx, settings, input.MmUserID, input.ChannelID, msg)

	default:
		if len(input.FileIDs) > 0 {
			return s.handleAttachFiles(ctx, settings, input.MmUserID, input.ChannelID, 0, input.FileIDs, "")
		}
		isAdmin := s.checkIsAdmin(ctx, settings.RealmID, input.MmUserID)
		return s.sendHelpMessage(settings.BotToken, input.ChannelID, isAdmin)
	}
}

func (s *MattermostService) sendCreateButton(botToken, channelID, realmID string) error {
	if s.baseURL == "" {
		return fmt.Errorf("failed to send create button: http.base_url is not configured")
	}

	_, err := s.most.Post.Create(botToken, mattermost.CreatePostDTO{
		ChannelID: channelID,
		Message:   "Для оформления заявки нажмите на кнопку ниже",
		Button: &mattermost.InteractiveButton{
			Text:  "Создать заявку",
			Style: "primary",
			URL:   fmt.Sprintf("%s/api/v1/mattermost/dialog/open", s.baseURL),
			Context: map[string]string{
				"realm_id": realmID,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send create button: %w", err)
	}
	return nil
}

func (s *MattermostService) sendHelpMessage(botToken, channelID string, isAdmin bool) error {
	text := `**Доступные команды:**

• **заявка | новая | создать** — создать новую заявку
• **статус | мои | заявки** — мои активные заявки
• **№123 текст** — добавить комментарий к заявке №123
• **№123 + файл(ы)** — прикрепить файлы к заявке №123
• **файл(ы) + текст** — прикрепить файлы и оставить комментарий к последней заявке (или по номеру)
• **помощь** — показать эту справку`

	if isAdmin {
		text += "\n• **синхронизировать [команда1,команда2]** — синхронизация пользователей"
	}

	_, err := s.most.Post.Create(botToken, mattermost.CreatePostDTO{
		ChannelID: channelID,
		Message:   text,
	})
	if err != nil {
		return fmt.Errorf("failed to send help message: %w", err)
	}
	return nil
}

func (s *MattermostService) sendStatusMessage(_ context.Context, settings *models.RealmMattermost, _ uuid.UUID, channelID string) error {
	text := "**Ваши активные заявки:**\n_(пока не реализовано)_"

	_, err := s.most.Post.Create(settings.BotToken, mattermost.CreatePostDTO{
		ChannelID: channelID,
		Message:   text,
	})
	if err != nil {
		return fmt.Errorf("failed to send status message: %w", err)
	}
	return nil
}

// handleAttachFiles прикрепляет файлы Mattermost к заявке пользователя и, если
// передан commentText (текст сообщения, а не только номер), оставляет его
// комментарием к той же заявке. Если указан номер заявки — файлы ищутся по нему,
// и файлы прикрепляются, только если пользователь её создатель (иначе файлы не
// прикрепляются; комментарий всё же создаётся при наличии work-доступа). Без
// номера файлы прикрепляются к последней созданной пользователем заявке из
// recentTickets, если с момента создания прошло не более 30 минут.
func (s *MattermostService) handleAttachFiles(ctx context.Context, settings *models.RealmMattermost, mmUserID, channelID string, ticketNumber int, fileIDs []string, commentText string) error {
	userID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	var ticketID uuid.UUID
	numberFromRecent := 0

	if ticketNumber > 0 {
		tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
			Number:  &ticketNumber,
			RealmID: &settings.RealmID,
			Actor:   &models.Actor{ID: userID},
		})
		if err != nil || len(tickets) == 0 {
			s.sendDMBestEffort(ctx, settings, mmUserID, fmt.Sprintf("Заявка №%d не найдена", ticketNumber))
			return nil
		}
		ticket := tickets[0]
		ticketID = ticket.ID
		if ticket.Creator.ID != userID {
			if commentText == "" {
				s.sendDMBestEffort(ctx, settings, mmUserID, "Прикреплять файлы может только создатель заявки")
				return nil
			}
			return s.commentAndReply(ctx, settings, mmUserID, ticketID, commentText)
		}
		numberFromRecent = ticketNumber
	} else {
		val, ok := s.recentTickets.Load(mmUserID + ":" + channelID)
		if !ok {
			s.sendDMBestEffort(ctx, settings, mmUserID, "Не найдена заявка для прикрепления файлов. Отправьте номер заявки (например, №123) вместе с файлами")
			return nil
		}
		rt := val.(*recentTicket)
		if time.Since(rt.createdAt) > 30*time.Minute {
			s.sendDMBestEffort(ctx, settings, mmUserID, "Прошло более 30 минут с создания заявки. Укажите номер заявки (например, №123) вместе с файлами")
			return nil
		}
		ticketID = rt.id
	}

	attached := 0
	for _, fileID := range fileIDs {
		data, err := s.most.Client.DownloadFile(settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		info, err := s.most.Client.GetFileInfo(settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to get MM file info for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		fileName := info.Name
		if fileName == "" {
			fileName = fileID
		}
		att, err := s.attachments.Upload(ctx, nil, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   ticketID,
			FileName:   fileName,
			FileSize:   info.Size,
			MimeType:   info.MimeType,
			File:       bytes.NewReader(data),
			UploadedBy: userID,
			Realm:      settings.RealmID.String(),
		})
		if err != nil {
			logger.Warn("failed to upload file as attachment", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		logger.Info("file attached to ticket",
			logger.StringAttr("ticket_id", ticketID.String()),
			logger.StringAttr("attachment_id", att.ID.String()),
			logger.StringAttr("file_name", fileName),
		)
		attached++
	}

	var reply string
	if numberFromRecent > 0 {
		reply = fmt.Sprintf("К заявке №%d прикреплено файлов: %d", numberFromRecent, attached)
	} else {
		reply = fmt.Sprintf("К заявке прикреплено файлов: %d", attached)
	}
	if commentText != "" {
		if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
			Text:       commentText,
			TicketID:   ticketID,
			IsInternal: false,
			Type:       "",
			UserID:     userID,
			Realm:      settings.RealmID.String(),
		}); err != nil {
			if errors.Is(err, models.ErrPermissionDenied) {
				reply += "\nНет прав на комментарий к этой заявке"
			} else {
				logger.Warn("failed to create comment with files", logger.StringAttr("ticket_id", ticketID.String()), logger.ErrAttr(err))
				reply += "\nНе удалось сохранить текст комментария"
			}
		} else {
			reply += "\nКомментарий добавлен"
		}
	}
	s.sendDMBestEffort(ctx, settings, mmUserID, reply)
	return nil
}

// commentAndReply создаёт комментарий к тикету (work-доступ) и отправляет
// пользователю DM с результатом. Используется, когда файлы не прикрепить
// (пользователь не создатель), но комментарий оставить можно.
func (s *MattermostService) commentAndReply(ctx context.Context, settings *models.RealmMattermost, mmUserID string, ticketID uuid.UUID, commentText string) error {
	if commentText == "" {
		return nil
	}
	userID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}
	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       commentText,
		TicketID:   ticketID,
		IsInternal: false,
		Type:       "",
		UserID:     userID,
		Realm:      settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ctx, settings, mmUserID, "Нет прав комментировать эту заявку")
			return nil
		}
		return fmt.Errorf("failed to create comment: %w", err)
	}
	s.sendDMBestEffort(ctx, settings, mmUserID, "Комментарий добавлен. Файлы может прикреплять только создатель заявки")
	return nil
}

// handleTextWithFiles обрабатывает сообщение с файлами и текстом: прикрепляет
// файлы и оставляет текст комментарием к той же заявке. Номер заявки берётся из
// сообщения (если есть, например «№123 ...»), иначе файлы и комментарий идут к
// последней созданной пользователем заявке из recentTickets.
func (s *MattermostService) handleTextWithFiles(ctx context.Context, settings *models.RealmMattermost, mmUserID, channelID, msg string, fileIDs []string) error {
	number := 0
	text := ""
	if m := commentCommands.FindStringSubmatch(msg); m != nil {
		number, _ = strconv.Atoi(m[1])
		text = strings.TrimSpace(m[2])
	} else {
		text = strings.TrimSpace(msg)
	}
	return s.handleAttachFiles(ctx, settings, mmUserID, channelID, number, fileIDs, text)
}

// handleComment создаёт комментарий к заявке из личного сообщения Mattermost.
// Синтаксис: «№123 текст комментария». Пользователь резолвится по mattermost_id,
// комментарий проходит ту же проверку work-доступа, что и написанный в программе.
func (s *MattermostService) handleComment(ctx context.Context, settings *models.RealmMattermost, mmUserID, channelID, message string) error {
	parts := commentCommands.FindStringSubmatch(message)
	number, _ := strconv.Atoi(parts[1])
	text := strings.TrimSpace(parts[2])

	userID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		Number:  &number,
		RealmID: &settings.RealmID,
		Actor:   &models.Actor{ID: userID},
	})
	if err != nil || len(tickets) == 0 {
		s.sendDMBestEffort(ctx, settings, mmUserID, fmt.Sprintf("Заявка №%d не найдена", number))
		return nil
	}
	ticket := tickets[0]

	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       text,
		TicketID:   ticket.ID,
		IsInternal: false,
		Type:       "",
		UserID:     userID,
		Realm:      settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ctx, settings, mmUserID, fmt.Sprintf("Нет прав комментировать заявку №%d", number))
			return nil
		}
		return fmt.Errorf("failed to create comment from mattermost: %w", err)
	}

	s.sendDMBestEffort(ctx, settings, mmUserID, fmt.Sprintf("Комментарий добавлен к заявке №%d", number))
	return nil
}

func (s *MattermostService) sendDM(settings *models.RealmMattermost, mmUserID, message string) error {
	err := s.most.DM.Send(settings.BotToken, settings.BotUserID, mmUserID, message)
	if err != nil {
		return fmt.Errorf("failed to send direct message: %w", err)
	}
	return nil
}

// sendDMBestEffort отправляет DM пользователю, но не является фатальной для
// уже совершённой операции (комментарий/вложения/синк уже в БД). Ошибка
// логируется и сигнализируется разработчику (error_bot), чтобы вебхук не
// отдавал 500 и Mattermost не ретраил операцию (иначе были бы дубликаты).
func (s *MattermostService) sendDMBestEffort(ctx context.Context, settings *models.RealmMattermost, mmUserID, message string) {
	if err := s.sendDM(settings, mmUserID, message); err != nil {
		bestEffortError("failed to send direct message", err, map[string]string{"mm_user_id": mmUserID})
	}
}

func (s *MattermostService) checkIsAdmin(ctx context.Context, realmID uuid.UUID, mmUserID string) bool {
	user, err := s.users.GetByMattermostID(ctx, mmUserID)
	if err != nil {
		return false
	}
	ur, err := s.userRealms.GetByUserAndRealm(ctx, user.ID, realmID)
	if err != nil || ur == nil || ur.Role == nil {
		return false
	}
	return ur.Role.Slug == "admin"
}
