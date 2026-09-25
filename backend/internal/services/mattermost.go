package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	Realms      Realms
	Tickets     Tickets
	Groups      Groups
	Categories  Categories
	Sites       Sites
	Attachments Attachments
	Comments    Comments
	Access      TicketAccessChecker
	EventBus    *events.PolicyEventManager
	Most        *mattermost.Most
	BaseURL     string
}

// mmChannel — контекст переписки пользователя в Mattermost (DM или канал из
// веб-сокета): настройки реалма и адресат (пользователь + канал).
type mmChannel struct {
	Settings  *models.RealmMattermost
	MmUserID  string
	ChannelID string
}

// MattermostService — тонкий фасад над Mattermost: обрабатывает команды из
// личных сообщений, диалоги создания заявок, интерактивные кнопки и веб-сокеты,
// делегируя низкоуровневую работу с Mattermost пакету pkg/mattermost.
type MattermostService struct {
	repo          repository.Mattermost
	users         Users
	userRealms    UserRealms
	roles         Roles
	realms        Realms
	tickets       Tickets
	groups        Groups
	categories    Categories
	sites         Sites
	attachments   Attachments
	comments      Comments
	access        TicketAccessChecker
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
		realms:      deps.Realms,
		tickets:     deps.Tickets,
		groups:      deps.Groups,
		categories:  deps.Categories,
		sites:       deps.Sites,
		attachments: deps.Attachments,
		comments:    deps.Comments,
		access:      deps.Access,
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
	GetSettingsByChannelID(ctx context.Context, channelID string) (*models.RealmMattermost, error)
	SaveSettings(ctx context.Context, realmID uuid.UUID, dto *models.RealmMattermostDTO) error
	DeleteSettings(ctx context.Context, realmID uuid.UUID) error

	HandleDM(ctx context.Context, input *HandleDMInput) error
	HandleDialogOpen(ctx context.Context, input *models.DialogOpenDTO) error
	HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error
	HandleInteractiveAction(ctx context.Context, input *models.InteractiveActionDTO) (*model.Post, error)

	// Плагин MM (webapp + plugin-server → /api/v1/plugin/*)
	PluginContext(ctx context.Context, channelID, mmUserID string) (*PluginContextResult, error)
	PluginCreateTicket(ctx context.Context, input *PluginCreateTicketInput) (*PluginCreateTicketResult, error)
	PluginListMine(ctx context.Context, channelID, mmUserID string) ([]PluginTicketShort, error)
	PluginGetTicket(ctx context.Context, channelID, mmUserID, ticketID string) (*PluginTicketDetail, error)
	PluginGetComments(ctx context.Context, channelID, mmUserID, ticketID string) ([]PluginComment, error)
	PluginCreateComment(ctx context.Context, input *PluginCreateCommentInput) (*PluginComment, error)
	PluginChangeStatus(ctx context.Context, channelID, mmUserID, ticketID, status string) error
	PluginGetAttachmentContent(ctx context.Context, channelID, mmUserID, attachmentID string) (*models.Attachment, io.ReadCloser, error)

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

// generateWebhookSecret создаёт случайный секрет вебхука Mattermost
// (48 hex-символов) для аутентификации входящих webhook-запросов.
func generateWebhookSecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate webhook secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// GetSettingsByChannelID возвращает настройки интеграции Mattermost для
// активного реалма по ID канала (используется для проверки webhook-токена).
func (s *MattermostService) GetSettingsByChannelID(ctx context.Context, channelID string) (*models.RealmMattermost, error) {
	settings, err := s.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mattermost settings: %w", err)
	}
	return settings, nil
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
// (секрет вебхука генерируется один раз и сохраняется между пересохранениями)
// и запускает веб-сокет для realm.
func (s *MattermostService) SaveSettings(ctx context.Context, realmID uuid.UUID, dto *models.RealmMattermostDTO) error {
	botUser, err := s.most.Client.GetMe(dto.BotToken)
	if err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	webhookSecret := ""
	if existing, err := s.repo.GetByRealm(ctx, realmID); err == nil && existing.WebhookSecret != "" {
		webhookSecret = existing.WebhookSecret
	} else {
		webhookSecret, err = generateWebhookSecret()
		if err != nil {
			return err
		}
	}

	settings := &models.RealmMattermost{
		RealmID:       realmID,
		BotToken:      dto.BotToken,
		BotUserID:     botUser.ID,
		ChannelID:     dto.ChannelID,
		WebhookSecret: webhookSecret,
		IsActive:      true,
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

	ch := &mmChannel{
		Settings:  settings,
		MmUserID:  input.MmUserID,
		ChannelID: input.ChannelID,
	}

	switch {
	case syncCommands.MatchString(msg):
		return s.handleSync(ctx, ch, msg)

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
		if _, err := s.resolveOrCreateUser(ctx, settings.RealmID, input.MmUserID, nil); err != nil {
			return fmt.Errorf("failed to resolve user: %w", err)
		}
		return s.sendStatusMessage(ch)

	case attachCommands.MatchString(msg) && len(input.FileIDs) > 0:
		parts := attachCommands.FindStringSubmatch(msg)
		number, _ := strconv.Atoi(parts[1])
		return s.handleAttachFiles(ctx, ch, number, input.FileIDs, "")

	case len(input.FileIDs) > 0 && !attachCommands.MatchString(msg):
		return s.handleTextWithFiles(ctx, ch, msg, input.FileIDs)

	case commentCommands.MatchString(msg):
		return s.handleComment(ctx, ch, msg)

	default:
		if len(input.FileIDs) > 0 {
			return s.handleAttachFiles(ctx, ch, 0, input.FileIDs, "")
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

func (s *MattermostService) sendStatusMessage(ch *mmChannel) error {
	text := "**Ваши активные заявки:**\n_(пока не реализовано)_"

	_, err := s.most.Post.Create(ch.Settings.BotToken, mattermost.CreatePostDTO{
		ChannelID: ch.ChannelID,
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
func (s *MattermostService) handleAttachFiles(ctx context.Context, ch *mmChannel, ticketNumber int, fileIDs []string, commentText string) error {
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	var ticketID uuid.UUID
	numberFromRecent := 0

	if ticketNumber > 0 {
		tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
			Number:  &ticketNumber,
			RealmID: &ch.Settings.RealmID,
			Actor:   &models.Actor{ID: user.ID},
		})
		if err != nil || len(tickets) == 0 {
			s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", ticketNumber))
			return nil
		}
		ticket := tickets[0]
		ticketID = ticket.ID
		if ticket.Creator.ID != user.ID {
			if commentText == "" {
				s.sendDMBestEffort(ch, "Прикреплять файлы может только создатель заявки")
				return nil
			}
			return s.commentAndReply(ctx, ch, ticketID, commentText)
		}
		numberFromRecent = ticketNumber
	} else {
		val, ok := s.recentTickets.Load(ch.MmUserID + ":" + ch.ChannelID)
		if !ok {
			s.sendDMBestEffort(ch, "Не найдена заявка для прикрепления файлов. Отправьте номер заявки (например, №123) вместе с файлами")
			return nil
		}
		rt := val.(*recentTicket)
		if time.Since(rt.createdAt) > 30*time.Minute {
			s.sendDMBestEffort(ch, "Прошло более 30 минут с создания заявки. Укажите номер заявки (например, №123) вместе с файлами")
			return nil
		}
		ticketID = rt.id
	}

	// В случае «файлы + текст» комментарий и файлы создаются атомарно и связываются
	// (файлы привязываются к комментарию через comment_id). Без текста файлы просто
	// прикрепляются к заявке.
	fileDTOs := make([]*models.UploadAttachmentDTO, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		data, err := s.most.Client.DownloadFile(ch.Settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		info, err := s.most.Client.GetFileInfo(ch.Settings.BotToken, fileID)
		if err != nil {
			logger.Warn("failed to get MM file info for attach", logger.StringAttr("file_id", fileID), logger.ErrAttr(err))
			continue
		}
		fileName := info.Name
		if fileName == "" {
			fileName = fileID
		}
		fileDTOs = append(fileDTOs, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   ticketID,
			FileName:   fileName,
			FileSize:   info.Size,
			MimeType:   info.MimeType,
			File:       bytes.NewReader(data),
			UploadedBy: user.ID,
			Realm:      ch.Settings.RealmID.String(),
		})
	}

	attached := 0
	commentCreated := false

	if commentText != "" {
		_, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
			Text:       commentText,
			TicketID:   ticketID,
			IsInternal: false,
			Type:       "",
			UserID:     user.ID,
			Realm:      ch.Settings.RealmID.String(),
			Files:      fileDTOs,
		})
		if err != nil {
			if errors.Is(err, models.ErrPermissionDenied) {
				s.sendDMBestEffort(ch, "Нет прав на комментарий к этой заявке")
				return nil
			}
			logger.Warn("failed to create comment with files", logger.StringAttr("ticket_id", ticketID.String()), logger.ErrAttr(err))
			s.sendDMBestEffort(ch, "Не удалось сохранить текст комментария")
			return nil
		}
		attached = len(fileDTOs)
		commentCreated = true
	} else {
		for _, fileDTO := range fileDTOs {
			att, err := s.tickets.UploadAttachment(ctx, nil, fileDTO)
			if err != nil {
				logger.Warn("failed to upload file as attachment", logger.StringAttr("ticket_id", ticketID.String()), logger.ErrAttr(err))
				continue
			}
			logger.Info("file attached to ticket",
				logger.StringAttr("ticket_id", ticketID.String()),
				logger.StringAttr("attachment_id", att.ID.String()),
				logger.StringAttr("file_name", fileDTO.FileName),
			)
			attached++
		}
	}

	var reply string
	if numberFromRecent > 0 {
		reply = fmt.Sprintf("К заявке №%d прикреплено файлов: %d", numberFromRecent, attached)
	} else {
		reply = fmt.Sprintf("К заявке прикреплено файлов: %d", attached)
	}
	if commentCreated {
		reply += "\nКомментарий добавлен"
	}
	if attached == 0 && len(fileDTOs) > 0 {
		reply = "Файлы не удалось прикрепить к заявке"
	}
	s.sendDMBestEffort(ch, reply)
	return nil
}

// commentAndReply создаёт комментарий к тикету (work-доступ) и отправляет
// пользователю DM с результатом. Используется, когда файлы не прикрепить
// (пользователь не создатель), но комментарий оставить можно.
func (s *MattermostService) commentAndReply(ctx context.Context, ch *mmChannel, ticketID uuid.UUID, commentText string) error {
	if commentText == "" {
		return nil
	}
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}
	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       commentText,
		TicketID:   ticketID,
		IsInternal: false,
		Type:       "",
		UserID:     user.ID,
		Realm:      ch.Settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, "Нет прав комментировать эту заявку")
			return nil
		}
		return fmt.Errorf("failed to create comment: %w", err)
	}
	s.sendDMBestEffort(ch, "Комментарий добавлен. Файлы может прикреплять только создатель заявки")
	return nil
}

// handleTextWithFiles обрабатывает сообщение с файлами и текстом: прикрепляет
// файлы и оставляет текст комментарием к той же заявке. Номер заявки берётся из
// сообщения (если есть, например «№123 ...»), иначе файлы и комментарий идут к
// последней созданной пользователем заявке из recentTickets.
func (s *MattermostService) handleTextWithFiles(ctx context.Context, ch *mmChannel, msg string, fileIDs []string) error {
	number := 0
	text := ""
	if m := commentCommands.FindStringSubmatch(msg); m != nil {
		number, _ = strconv.Atoi(m[1])
		text = strings.TrimSpace(m[2])
	} else {
		text = strings.TrimSpace(msg)
	}
	return s.handleAttachFiles(ctx, ch, number, fileIDs, text)
}

// handleComment создаёт комментарий к заявке из личного сообщения Mattermost.
// Синтаксис: «№123 текст комментария». Пользователь резолвится по mattermost_id,
// комментарий проходит ту же проверку work-доступа, что и написанный в программе.
func (s *MattermostService) handleComment(ctx context.Context, ch *mmChannel, message string) error {
	parts := commentCommands.FindStringSubmatch(message)
	number, _ := strconv.Atoi(parts[1])
	text := strings.TrimSpace(parts[2])

	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		Number:  &number,
		RealmID: &ch.Settings.RealmID,
		Actor:   &models.Actor{ID: user.ID},
	})
	if err != nil || len(tickets) == 0 {
		s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
		return nil
	}
	ticket := tickets[0]

	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:       text,
		TicketID:   ticket.ID,
		IsInternal: false,
		Type:       "",
		UserID:     user.ID,
		Realm:      ch.Settings.RealmID.String(),
	}); err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, fmt.Sprintf("Нет прав комментировать заявку №%d", number))
			return nil
		}
		return fmt.Errorf("failed to create comment from mattermost: %w", err)
	}

	s.sendDMBestEffort(ch, fmt.Sprintf("Комментарий добавлен к заявке №%d", number))
	return nil
}

func (s *MattermostService) sendDM(ch *mmChannel, message string) error {
	err := s.most.DM.Send(ch.Settings.BotToken, ch.Settings.BotUserID, ch.MmUserID, message)
	if err != nil {
		return fmt.Errorf("failed to send direct message: %w", err)
	}
	return nil
}

// sendDMBestEffort отправляет DM пользователю, но не является фатальной для
// уже совершённой операции (комментарий/вложения/синк уже в БД). Ошибка
// логируется и сигнализируется разработчику (error_bot), чтобы вебхук не
// отдавал 500 и Mattermost не ретраил операцию (иначе были бы дубликаты).
func (s *MattermostService) sendDMBestEffort(ch *mmChannel, message string) {
	if err := s.sendDM(ch, message); err != nil {
		bestEffortError("failed to send direct message", err, map[string]string{"mm_user_id": ch.MmUserID})
	}
}

func (s *MattermostService) checkIsAdmin(ctx context.Context, realmID uuid.UUID, mmUserID string) bool {
	user, err := s.users.GetByMattermostID(ctx, mmUserID)
	if err != nil {
		return false
	}
	return s.isRealmSupervisor(ctx, user.ID, realmID)
}

// isRealmSupervisor определяет, является ли системный пользователь «начальником области»
// в реалме (realm-wide пермишены управления областью). Делегируется единому объекту доступа.
func (s *MattermostService) isRealmSupervisor(ctx context.Context, userID uuid.UUID, realmID uuid.UUID) bool {
	ok, err := s.access.IsRealmSupervisor(ctx, userID, realmID.String())
	if err != nil {
		logger.Error("failed to check realm supervisor",
			logger.StringAttr("user_id", userID.String()),
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return false
	}
	return ok
}
