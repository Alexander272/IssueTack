package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

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

// readableMode — режим поиска заявки по номеру в боте (см. TicketService.Get):
// возвращает заявки, доступные пользователю на чтение по атрибутам — он автор,
// заказчик, исполнитель или участник/менеджер группы тикета. Обычный списочный
// scope отсекал бы создателя, который сам в группу тикета не входит.
const readableMode = "readable"

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
	Policies    AccessPolicies
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
	policies      AccessPolicies
	eventBus      *events.PolicyEventManager
	most          *mattermost.Most
	baseURL       string
	wsClients     map[string]*mattermost.WSClient
	wsMu          sync.Mutex
	wsGuard       *inboundGuard
	pendingFiles  sync.Map
	recentTickets sync.Map
	pluginCache   *pluginContextCache
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
		policies:    deps.Policies,
		eventBus:    deps.EventBus,
		most:        deps.Most,
		baseURL:     deps.BaseURL,
		wsClients:   make(map[string]*mattermost.WSClient),
		wsGuard:     newInboundGuard(),
		pluginCache: newPluginContextCache(),
	}
}

// Mattermost — публичный интерфейс фасада Mattermost, используемый
// HTTP-обработчиками и главной точкой входа.
type Mattermost interface {
	GetSettings(ctx context.Context, realmID uuid.UUID) (*models.RealmMattermost, error)
	GetSettingsByChannelID(ctx context.Context, channelID string) (*models.RealmMattermost, error)
	SaveSettings(ctx context.Context, realmID uuid.UUID, dto *models.RealmMattermostDTO) error
	DeleteSettings(ctx context.Context, realmID uuid.UUID) error

	// SyncRealmUsers импортирует пользователей Mattermost в realm: сопоставляет их
	// с системными по mattermost_id/email/username/ФИО и создаёт недостающих.
	// Доступно только администратору realm и только при активной интеграции.
	SyncRealmUsers(ctx context.Context, realmID uuid.UUID, actor *models.Actor, teamNames []string) (*SyncUsersResult, error)

	HandleDM(ctx context.Context, input *models.HandleDMInput) error
	HandleDialogOpen(ctx context.Context, input *models.DialogOpenDTO) error
	HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error
	HandleInteractiveAction(ctx context.Context, input *models.InteractiveActionDTO) (*ActionResult, error)

	// Плагин MM (webapp + plugin-server → /api/v1/plugin/*).
	// Scope несёт канал, Mattermost-пользователя и бота реалма для личного диалога.
	PluginContext(ctx context.Context, scope models.PluginScope) (*models.PluginContextResult, error)
	PluginGreeting(ctx context.Context, scope models.PluginScope) error
	// PluginDM отдаёт личный диалог пользователя с ботом реалма — по нему иконка
	// в App Bar открывает модалку из любого канала.
	PluginDM(ctx context.Context, mmUserID string) (*models.PluginDMResult, error)
	PluginCreateTicket(ctx context.Context, input *models.PluginCreateTicketInput) (*models.PluginCreateTicketResult, error)
	PluginListMine(ctx context.Context, query *models.PluginTicketListQuery) (*models.PluginTicketListResult, error)
	PluginGetTicket(ctx context.Context, scope models.PluginScope, ticketID string) (*models.PluginTicketDetail, error)
	PluginGetTicketLinkContext(ctx context.Context, mmUserID, ticketID string) (*models.PluginTicketLinkContext, error)
	PluginGetComments(ctx context.Context, scope models.PluginScope, ticketID string) ([]models.PluginComment, error)
	PluginCreateComment(ctx context.Context, input *models.PluginCreateCommentInput) (*models.PluginComment, error)
	PluginChangeStatus(ctx context.Context, scope models.PluginScope, ticketID, status string) error
	PluginGetAttachmentContent(ctx context.Context, scope models.PluginScope, attachmentID string) (*models.Attachment, io.ReadCloser, error)

	StartWSForRealm(ctx context.Context, realmID uuid.UUID) error
	StopWSForRealm(realmID uuid.UUID)
	StopAllWS()
	StartAllActiveWS(ctx context.Context)
}

// ActionResult — ответ бота на нажатие интерактивной кнопки Mattermost.
// Ephemeral показывает текст только нажавшему — так сообщение об ошибке не
// засоряет общий список заявок. Перерисовать пост нажатием нельзя: все
// необратимые действия в списке заявок идут через диалог, а ответ на его
// отправку не умеет обновлять посты.
type ActionResult struct {
	Ephemeral string
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

	// Перезапуск: StartWSForRealm идемпотентен, поэтому уже запущенный сокет
	// продолжил бы работу со старым токеном до перезапуска приложения.
	s.StopWSForRealm(realmID)

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
func (s *MattermostService) HandleDM(ctx context.Context, input *models.HandleDMInput) error {
	// Пустое сообщение без явного вызова команды — не вопрос боту. Раньше на
	// такой пост (в т.ч. на системное сообщение Mattermost, доехавшее по
	// HTTP-пути) бот отвечал справкой, то есть писал тому, кто ничего не писал.
	// Пустой вызов команды (у slash-команды есть trigger_id) остаётся вопросом:
	// человек осознанно её вызвал и ждёт ответа.
	if strings.TrimSpace(input.Message) == "" && len(input.FileIDs) == 0 && input.TriggerID == "" {
		return nil
	}

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

	// Автор — сам бот: на собственные сообщения бот не отвечает. Иначе при
	// расхождении bot_user_id (остался от прежнего токена или правился руками в
	// БД) каждое сообщение бота возвращалось как входящее и порождало новое —
	// цикл «справка на справку». Веб-сокет отсеивает такие посты фильтром
	// AcceptPost по id, авторитетному для токена; здесь — та же защита для
	// HTTP-путей.
	if input.MmUserID == settings.BotUserID {
		return nil
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
		return s.sendMenu(settings.BotToken, input.ChannelID, settings.RealmID.String(),
			"Для оформления заявки нажмите на кнопку ниже")

	case helpCommands.MatchString(msg):
		isAdmin := s.checkIsAdmin(ctx, settings.RealmID, input.MmUserID)
		return s.sendHelpMessage(settings.BotToken, input.ChannelID, settings.RealmID.String(), isAdmin)

	case statusCommands.MatchString(msg):
		return s.sendStatusMessage(ctx, ch)

	case attachCommands.MatchString(msg) && len(input.FileIDs) > 0:
		parts := attachCommands.FindStringSubmatch(msg)
		number, _ := strconv.Atoi(parts[1])
		return s.handleAttachFiles(ctx, ch, number, input.FileIDs, "")

	case len(input.FileIDs) > 0 && !attachCommands.MatchString(msg):
		return s.handleTextWithFiles(ctx, ch, msg, input.FileIDs)

	case commentCommands.MatchString(msg):
		return s.handleComment(ctx, ch, msg)

	// Голое «№123» без текста и без файлов — карточка заявки. Кейс стоит после
	// attachCommands (тот с файлами) и commentCommands (с текстом), иначе он
	// перехватил бы их.
	case attachCommands.MatchString(msg):
		return s.handleTicketCard(ctx, ch, msg)

	default:
		if len(input.FileIDs) > 0 {
			return s.handleAttachFiles(ctx, ch, 0, input.FileIDs, "")
		}
		isAdmin := s.checkIsAdmin(ctx, settings.RealmID, input.MmUserID)
		return s.sendHelpMessage(settings.BotToken, input.ChannelID, settings.RealmID.String(), isAdmin)
	}
}

// sendMenu отправляет сообщение с кнопками-командами бота. Текст задаёт вызывающий:
// справка приветствия, «помощь» или приглашение оформить заявку. Без настроенного
// http.base_url кнопки построить нельзя (в них уходят адреса наших хендлеров), поэтому
// сообщение уходит без них, а не падает.
func (s *MattermostService) sendMenu(botToken, channelID, realmID, message string) error {
	_, err := s.most.Post.Create(botToken, mattermost.CreatePostDTO{
		ChannelID: channelID,
		Message:   message,
		Buttons:   s.menuButtons(realmID),
	})
	if err != nil {
		return fmt.Errorf("failed to send bot menu: %w", err)
	}
	return nil
}

// menuButtons — кнопки бота вместо команд в чате: «Создать заявку» открывает
// интерактивный диалог, «Мои заявки» возвращает список карточками. Обе кнопки
// нативны — Mattermost рисует их и в веб-клиенте, и в мобильном приложении,
// где плагин недоступен.
func (s *MattermostService) menuButtons(realmID string) []mattermost.InteractiveButton {
	if s.baseURL == "" {
		return nil
	}
	return []mattermost.InteractiveButton{
		{
			Text:  "Создать заявку",
			Style: "primary",
			URL:   fmt.Sprintf("%s/api/v1/mattermost/dialog/open", s.baseURL),
			Context: map[string]string{
				"realm_id": realmID,
			},
		},
		{
			Text: "Мои заявки",
			URL:  s.actionURL(),
			Context: map[string]string{
				"action":   "my_tickets",
				"realm_id": realmID,
			},
		},
	}
}

// actionURL — адрес обработчика нажатий интерактивных кнопок Mattermost.
func (s *MattermostService) actionURL() string {
	return fmt.Sprintf("%s/api/v1/mattermost/action", s.baseURL)
}

func (s *MattermostService) sendHelpMessage(botToken, channelID, realmID string, isAdmin bool) error {
	text := `**Доступные команды:**

• **заявка | новая | создать** — создать новую заявку
• **статус | мои | заявки** — мои активные заявки
• **№123 текст** — добавить комментарий к заявке №123
• **№123 + файл(ы)** — прикрепить файлы к заявке №123
• **файл(ы) + текст** — прикрепить файлы и оставить комментарий к последней заявке (или по номеру)
• **помощь** — показать эту справку

Можно не печатать команды — используйте кнопки ниже.`

	if isAdmin {
		text += "\n• **синхронизировать [команда1,команда2]** — синхронизация пользователей"
	}

	if err := s.sendMenu(botToken, channelID, realmID, text); err != nil {
		return fmt.Errorf("failed to send help message: %w", err)
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
