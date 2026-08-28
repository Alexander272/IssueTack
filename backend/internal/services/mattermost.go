package services

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

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
		return s.handleAttachFiles(ctx, settings, input.MmUserID, input.ChannelID, number, input.FileIDs)

	default:
		if len(input.FileIDs) > 0 {
			return s.handleAttachFiles(ctx, settings, input.MmUserID, input.ChannelID, 0, input.FileIDs)
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

// handleAttachFiles прикрепляет файлы Mattermost к заявке пользователя. Если
// указан номер заявки — она ищется и проверяется, что пользователь её создатель
// (иначе файлы не прикрепляются). Без номера файлы прикрепляются к последней
// созданной пользователем заявке из recentTickets, если с момента создания
// прошло не более 30 минут, — так команды вида «№123 + файлы» и просто постинг
// файлов после создания заявки работают единообразно.
func (s *MattermostService) handleAttachFiles(ctx context.Context, settings *models.RealmMattermost, mmUserID, channelID string, ticketNumber int, fileIDs []string) error {
	userID, _, err := s.resolveOrCreateUser(ctx, settings.RealmID, mmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	var ticketID uuid.UUID

	if ticketNumber > 0 {
		tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
			Number:  &ticketNumber,
			RealmID: &settings.RealmID,
			Actor:   &models.Actor{ID: userID},
		})
		if err != nil || len(tickets) == 0 {
			return s.sendDM(settings, mmUserID, fmt.Sprintf("Заявка №%d не найдена", ticketNumber))
		}
		ticket := tickets[0]
		if ticket.Creator.ID != userID {
			return s.sendDM(settings, mmUserID, "Прикреплять файлы может только создатель заявки")
		}
		ticketID = ticket.ID
	} else {
		val, ok := s.recentTickets.Load(mmUserID + ":" + channelID)
		if !ok {
			return s.sendDM(settings, mmUserID, "Не найдена заявка для прикрепления файлов. Отправьте номер заявки (например, №123) вместе с файлами")
		}
		rt := val.(*recentTicket)
		if time.Since(rt.createdAt) > 30*time.Minute {
			return s.sendDM(settings, mmUserID, "Прошло более 30 минут с создания заявки. Укажите номер заявки (например, №123) вместе с файлами")
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

	if attached > 0 {
		msg := fmt.Sprintf("К заявке прикреплено файлов: %d", attached)
		if ticketNumber > 0 {
			msg = fmt.Sprintf("К заявке №%d прикреплено файлов: %d", ticketNumber, attached)
		}
		return s.sendDM(settings, mmUserID, msg)
	}
	return nil
}

func (s *MattermostService) sendDM(settings *models.RealmMattermost, mmUserID, message string) error {
	err := s.most.DM.Send(settings.BotToken, settings.BotUserID, mmUserID, message)
	if err != nil {
		return fmt.Errorf("failed to send direct message: %w", err)
	}
	return nil
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
