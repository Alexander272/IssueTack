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
		eventBus:    deps.EventBus,
		most:        deps.Most,
		baseURL:     deps.BaseURL,
		wsClients:   make(map[string]*mattermost.WSClient),
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
	PluginCreateTicket(ctx context.Context, input *models.PluginCreateTicketInput) (*models.PluginCreateTicketResult, error)
	PluginListMine(ctx context.Context, scope models.PluginScope) ([]models.PluginTicketShort, error)
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

const (
	// myTicketsChunkSize — сколько заявок помещается в одно сообщение бота:
	// список разбивается на сообщения, а не обрезается (молчаливый обрез до 20
	// выглядел для пользователя как «бот потерял заявки»). Карточка несёт
	// описание заявки, поэтому в сообщение влезает меньше заявок.
	myTicketsChunkSize = 5
	// myTicketsMaxCount — предохранитель: список длиннее отдаётся первыми
	// myTicketsMaxCount заявками, чтобы не завалить диалог сотнями постов.
	myTicketsMaxCount = 200
	// myTicketsPostDelay — пауза между сообщениями: Mattermost ограничивает
	// частоту постов, и серия сообщений без пауз частично отбрасывается с 429.
	myTicketsPostDelay = 350 * time.Millisecond
)

// myTicketAction* — действия кнопок в списке «Мои заявки». Правила переходов
// те же, что в плагине (см. PluginChangeStatus).
const (
	actionCancel  = "ticket_cancel"
	actionConfirm = "ticket_confirm"
	actionReopen  = "ticket_reopen"
	// reopenCallbackPrefix — префикс callback_id диалога «Вернуть в работу».
	reopenCallbackPrefix = actionReopen + ":"
	// cancelCallbackPrefix и confirmCallbackPrefix — префиксы callback_id
	// диалогов подтверждения необратимых переходов: отмены и закрытия заявки.
	cancelCallbackPrefix  = actionCancel + ":"
	confirmCallbackPrefix = actionConfirm + ":"
)

// actionTargetStatus — статус, в который переводит заявку действие из списка
// «Мои заявки». Целевой статус берём из самого действия, а не из контекста
// кнопки: контекст приходит от клиента и не должен решать, что делать.
func actionTargetStatus(action string) models.TicketStatus {
	switch action {
	case actionCancel:
		return models.StatusCancelled
	case actionConfirm:
		return models.StatusClosed
	case actionReopen:
		return models.StatusInProgress
	}
	return ""
}

// mmStatusLabels — подписи статусов заявки в карточках бота. Взяты из
// mmPlugin/webapp/src/labels.ts, чтобы список в Mattermost и в плагине называл
// статусы одинаково.
var mmStatusLabels = map[models.TicketStatus]string{
	models.StatusOpen:       "Новая",
	models.StatusInProgress: "В работе",
	models.StatusPending:    "Ожидание",
	models.StatusOnHold:     "Отложена",
	models.StatusResolved:   "Решена",
	models.StatusClosed:     "Закрыта",
	models.StatusCancelled:  "Отменена",
}

// mmStatusColors — цвет полосы карточки по статусу (значения из labels.ts плагина).
var mmStatusColors = map[models.TicketStatus]string{
	models.StatusOpen:       "#01579B",
	models.StatusInProgress: "#E65100",
	models.StatusPending:    "#F57F17",
	models.StatusOnHold:     "#4A148C",
	models.StatusResolved:   "#1B5E20",
	models.StatusClosed:     "#024A02",
	models.StatusCancelled:  "#B71C1C",
}

// myActiveTickets собирает заявки, которые показываем пользователю по кнопке
// «Мои заявки» и по команде «мои»: активные, где он автор ИЛИ заказчик —
// тот же набор, что у вкладки «Мои заявки» в плагине, чтобы список не расходился
// между клиентами. Resolved оставлен, чтобы по таким заявкам можно было
// подтвердить решение или вернуть в работу.
func (s *MattermostService) myActiveTickets(ctx context.Context, realmID uuid.UUID, user *models.UserData) ([]*models.Ticket, error) {
	mode := "created_or_owned"
	tickets, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		RealmID: &realmID,
		Statuses: []models.TicketStatus{
			models.StatusOpen,
			models.StatusInProgress,
			models.StatusPending,
			models.StatusOnHold,
			models.StatusResolved,
		},
		Actor: &models.Actor{ID: user.ID, Name: user.Username},
		Mode:  &mode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}
	return tickets, nil
}

// myTicketsChunk — одно сообщение со списком заявок пользователя: заголовок
// и карточки с действиями.
type myTicketsChunk struct {
	message string
	cards   []mattermost.Attachment
}

// myTicketsChunks разбивает заявки пользователя на сообщения по
// myTicketsChunkSize — показываем все, а не первые 20.
func (s *MattermostService) myTicketsChunks(tickets []*models.Ticket, ownerID uuid.UUID) []myTicketsChunk {
	if len(tickets) == 0 {
		return []myTicketsChunk{{message: "У вас нет активных заявок."}}
	}
	if len(tickets) > myTicketsMaxCount {
		tickets = tickets[:myTicketsMaxCount]
	}

	chunks := make([]myTicketsChunk, 0, (len(tickets)+myTicketsChunkSize-1)/myTicketsChunkSize)
	for from := 0; from < len(tickets); from += myTicketsChunkSize {
		chunk, ok := s.myTicketsChunkAt(tickets, from, ownerID)
		if !ok {
			break
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

// myTicketsChunkAt собирает одно сообщение со срезом заявок [from, from+размер).
// Смещение from кладут в контекст кнопок карточек: после смены статуса бот
// перерисовывает тем же смещением тот же срез.
func (s *MattermostService) myTicketsChunkAt(tickets []*models.Ticket, from int, ownerID uuid.UUID) (myTicketsChunk, bool) {
	if from < 0 || from >= len(tickets) {
		return myTicketsChunk{}, false
	}
	end := min(from+myTicketsChunkSize, len(tickets))
	return s.myTicketsChunk(tickets, from, end, myTicketsHeader(from, end, len(tickets)), ownerID), true
}

// myTicketsChunkAfterEdit пересобирает сообщение списка после действия над
// заявкой, нажатой в срезе [from, ...).
//
//	removed=true  — заявка ушла из активного списка (отмена, подтверждение
//	                решения): показываем ровно тот же срез без этой карточки.
//	                Всё, что было ниже удалённой заявки, сдвинулось на один
//	                элемент, поэтому срез нужно взять на элемент короче, иначе
//	                в него подтянется лишняя карточка из следующего сообщения.
//	removed=false — заявка осталась в списке (возврат в работу): перерисовываем
//	                срез целиком, карточка получит новый статус и кнопки.
//
// Заголовок всегда считается по состоянию ДО действия, поэтому он не «поедет»
// после удаления карточки. ok=false означает, что в сообщении не осталось ни
// одной карточки — такое сообщение надо удалить.
func (s *MattermostService) myTicketsChunkAfterEdit(tickets []*models.Ticket, from int, removed bool, ownerID uuid.UUID) (myTicketsChunk, bool) {
	if from < 0 || from > len(tickets) {
		return myTicketsChunk{}, false
	}

	total := len(tickets)
	if removed {
		total++ // убираем из тотала заявку, которая выпала из списка
	}
	oldEnd := min(from+myTicketsChunkSize, total)

	end := oldEnd
	if removed {
		end--
	}
	end = min(end, len(tickets))
	if from >= end {
		return myTicketsChunk{}, false
	}
	return s.myTicketsChunk(tickets, from, end, myTicketsHeader(from, oldEnd, total), ownerID), true
}

// myTicketsChunk превращает срез заявок в карточки поста с готовым заголовком.
func (s *MattermostService) myTicketsChunk(tickets []*models.Ticket, from, end int, message string, ownerID uuid.UUID) myTicketsChunk {
	cards := make([]mattermost.Attachment, 0, end-from)
	for _, t := range tickets[from:end] {
		cards = append(cards, s.myTicketCard(t, ownerID, from))
	}
	return myTicketsChunk{message: message, cards: cards}
}

// myTicketsHeader — заголовок сообщения со срезом заявок. Первое сообщение
// показывает общее количество, остальные — свой диапазон.
func myTicketsHeader(from, end, total int) string {
	if from == 0 {
		return fmt.Sprintf("**Ваши активные заявки** (%d):", total)
	}
	return fmt.Sprintf("Заявки %d–%d из %d:", from+1, end, total)
}

// myTicketCard собирает карточку заявки списка. Заголовок ведёт на заявку
// (title_link) — отдельной кнопки «Открыть заявку» нет, вместо неё у заявки
// владельца появляются действия по статусу.
func (s *MattermostService) myTicketCard(t *models.Ticket, ownerID uuid.UUID, from int) mattermost.Attachment {
	title := t.Title
	if t.TicketNumber != nil {
		title = fmt.Sprintf("№%d — %s", *t.TicketNumber, t.Title)
	}

	fields := []mattermost.AttachmentField{
		{Title: "Статус", Value: mmStatusLabels[t.Status], Short: true},
		{Title: "Создана", Value: t.CreatedAt.Format("02.01.2006"), Short: true},
	}
	if t.Site != nil && t.Site.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Площадка", Value: t.Site.Name, Short: true})
	}

	card := mattermost.Attachment{
		Title:     title,
		TitleLink: s.taskLink(t.ID),
		Text:      cardSummary(t.Description),
		Color:     mmStatusColors[t.Status],
		Fields:    fields,
		Buttons:   s.myTicketActionButtons(t, ownerID, from),
	}
	return card
}

// cardSummary — описание заявки для карточки в посте. Текст схлопывается в одну
// строку (карточка — это строчка в ленте, а не абзац) и подрезается, чтобы одна
// заявка с длинным описанием не растянула весь список.
const cardSummaryLimit = 200

func cardSummary(text string) string {
	summary := strings.Join(strings.Fields(text), " ")
	if len([]rune(summary)) <= cardSummaryLimit {
		return summary
	}
	return string([]rune(summary)[:cardSummaryLimit]) + "…"
}

// myTicketActionButtons — действия владельца заявки в списке «Мои заявки».
// Правила те же, что в плагине: отменить можно только новую заявку,
// подтвердить решение и вернуть в работу — только решённую. Заявки, где
// пользователь не владелец, остаются без кнопок.
func (s *MattermostService) myTicketActionButtons(t *models.Ticket, ownerID uuid.UUID, from int) []mattermost.InteractiveButton {
	if t.Owner == nil || t.Owner.ID != ownerID {
		return nil
	}

	btnCtx := func(action string, status models.TicketStatus) map[string]string {
		return map[string]string{
			"action":    action,
			"ticket_id": t.ID.String(),
			"status":    string(status),
			"from":      strconv.Itoa(from),
		}
	}

	switch t.Status {
	case models.StatusOpen:
		return []mattermost.InteractiveButton{{
			Text: "Отменить заявку", Style: "danger",
			URL: s.actionURL(), Context: btnCtx(actionCancel, models.StatusCancelled),
			ID: ticketActionID(t.ID, actionCancel),
		}}
	case models.StatusResolved:
		return []mattermost.InteractiveButton{
			{
				Text: "Подтвердить решение", Style: "primary",
				URL: s.actionURL(), Context: btnCtx(actionConfirm, models.StatusClosed),
				ID: ticketActionID(t.ID, actionConfirm),
			},
			{
				Text: "Вернуть в работу",
				URL:  s.actionURL(), Context: btnCtx(actionReopen, models.StatusInProgress),
				ID: ticketActionID(t.ID, actionReopen),
			},
		}
	default:
		return nil
	}
}

// ticketActionID — action_id кнопки заявки. Он должен быть уникален в посте
// (сервер ищет действие по id в attachments поста и берёт первое совпадение, так
// что общий id у карточек увёл бы нажатие на чужую заявку) и состоять только из
// [A-Za-z0-9] — роут /posts/{id}/actions/{action_id} дефисы и подчёркивания не
// принимает, поэтому uuid режем до hex-префикса, а из имени действия выкидываем
// всё лишнее. Привязка к заявке, а не позиция в срезе: id не «едет», когда
// карточка уезжает из списка.
func ticketActionID(ticketID uuid.UUID, action string) string {
	suffix := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		}
		return -1
	}, strings.TrimPrefix(action, "ticket_"))
	if suffix == "" {
		suffix = "act"
	}
	return "t" + strings.ReplaceAll(ticketID.String(), "-", "")[:12] + suffix
}

// taskLink — абсолютная ссылка на заявку в веб-приложении. Абсолютная нужна для
// мобильного клиента Mattermost: site-relative адрес здесь не откроется.
func (s *MattermostService) taskLink(ticketID uuid.UUID) string {
	if s.baseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/tasks/%s", s.baseURL, ticketID.String())
}

// sendStatusMessage отвечает на команду «мои | статус | заявки» списком активных
// заявок карточками — тем же содержимым, что и кнопка «Мои заявки», но обычным
// сообщением в канале. Длинный список уходит несколькими сообщениями.
func (s *MattermostService) sendStatusMessage(ctx context.Context, ch *mmChannel) error {
	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	tickets, err := s.myActiveTickets(ctx, ch.Settings.RealmID, user)
	if err != nil {
		return err
	}

	return s.sendMyTicketsChunks(ctx, ch.Settings.BotToken, ch.ChannelID, s.myTicketsChunks(tickets, user.ID))
}

// sendMyTicketsChunks отправляет список заявок одним или несколькими сообщениями
// с паузой между ними. Ошибка одного сообщения не отменяет остальные: список
// на 27 заявок не должен пропадать из-за одного отказа.
func (s *MattermostService) sendMyTicketsChunks(ctx context.Context, botToken, channelID string, chunks []myTicketsChunk) error {
	var firstErr error
	for i, chunk := range chunks {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(myTicketsPostDelay):
			}
		}
		if _, err := s.most.Post.Create(botToken, mattermost.CreatePostDTO{
			ChannelID:   channelID,
			Message:     chunk.message,
			Attachments: chunk.cards,
		}); err != nil {
			logger.Warn("failed to send my tickets message",
				logger.StringAttr("channel_id", channelID),
				logger.IntAttr("part", i+1),
				logger.IntAttr("parts", len(chunks)),
				logger.ErrAttr(err),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to send status message: %w", err)
			}
		}
	}
	return firstErr
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

// handleTicketCard отвечает на голый номер заявки («№123») карточкой: статус,
// приоритет, участники, срок и действия владельца.
//
// Право на просмотр здесь обязательно и обеспечивается двумя шагами. Поиск по
// номеру идёт через Tickets.Get, который без групп актора вернул бы заявку
// любого реалма с таким номером, поэтому найденный ID досматривается через
// ticketDetail → Tickets.GetByID — он гоняет атрибутную модель доступа и вернёт
// ErrPermissionDenied на чужую заявку. Отказ показываем тем же «не найдена», что и
// для несуществующего номера: сообщать о существовании чужой заявки нельзя.
func (s *MattermostService) handleTicketCard(ctx context.Context, ch *mmChannel, message string) error {
	parts := attachCommands.FindStringSubmatch(message)
	number, _ := strconv.Atoi(parts[1])

	user, err := s.resolveOrCreateUser(ctx, ch.Settings.RealmID, ch.MmUserID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	found, _, err := s.tickets.Get(ctx, &models.TicketFilter{
		Number:  &number,
		RealmID: &ch.Settings.RealmID,
		Actor:   &models.Actor{ID: user.ID, Name: user.Username},
	})
	if err != nil {
		return fmt.Errorf("failed to find ticket by number: %w", err)
	}
	if len(found) == 0 {
		s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
		return nil
	}

	detail, err := s.ticketDetail(ctx, ch.Settings, ch.MmUserID, found[0].ID)
	if err != nil {
		if errors.Is(err, models.ErrPermissionDenied) {
			s.sendDMBestEffort(ch, fmt.Sprintf("Заявка №%d не найдена", number))
			return nil
		}
		return fmt.Errorf("failed to get ticket detail: %w", err)
	}

	if _, err := s.most.Post.Create(ch.Settings.BotToken, mattermost.CreatePostDTO{
		ChannelID:   ch.ChannelID,
		Message:     fmt.Sprintf("Заявка №%d", detail.Number),
		Attachments: []mattermost.Attachment{s.ticketCard(detail)},
	}); err != nil {
		return fmt.Errorf("failed to send ticket card: %w", err)
	}
	return nil
}

// cardDetailLimit — лимит описания в карточке заявки. Выше cardSummary (200) для
// одной заявки: карточка одна, растянуть ленту она не может, а обрезанное
// описание бесполезно — за ним идут за полным текстом в веб-приложении.
const cardDetailLimit = 1000

var mmPriorityLabels = map[models.Priority]string{
	models.PriorityLow:    "Низкий",
	models.PriorityMedium: "Средний",
	models.PriorityHigh:   "Высокий",
	models.PriorityUrgent: "Срочный",
}

// ticketCard собирает карточку заявки для ответа на «№N». Заголовок ведёт на
// заявку через title_link: отдельного поля url у вложения нет, клиенты рисуют
// ссылкой только заголовок. Кнопки — те же, что у карточки в списке «Мои
// заявки», но без post_id: applyListEdit правку списка не делает, а по
// standalone-карточке её и нечего править, поэтому действие уходит в диалог
// подтверждения и меняет статус через общий TicketService.Update.
func (s *MattermostService) ticketCard(detail *models.PluginTicketDetail) mattermost.Attachment {
	title := detail.Title
	fields := []mattermost.AttachmentField{
		{Title: "Статус", Value: mmStatusLabels[detail.Status], Short: true},
		{Title: "Приоритет", Value: mmPriorityLabels[detail.Priority], Short: true},
	}
	if detail.Category != nil && detail.Category.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Категория", Value: detail.Category.Name, Short: true})
	}
	if detail.Site != nil && detail.Site.Name != "" {
		fields = append(fields, mattermost.AttachmentField{Title: "Площадка", Value: detail.Site.Name, Short: true})
	}
	fields = append(fields, mattermost.AttachmentField{
		Title: "Создана", Value: detail.CreatedAt.Format("02.01.2006 15:04"), Short: true,
	})
	if detail.DueDate != nil {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Срок", Value: detail.DueDate.Format("02.01.2006"), Short: true,
		})
	}
	if detail.Assignee != nil && detail.Assignee.Username != "" {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Исполнитель", Value: detail.Assignee.Username, Short: true,
		})
	}
	if detail.Owner != nil && detail.Owner.Username != "" {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Заказчик", Value: detail.Owner.Username, Short: true,
		})
	}
	if len(detail.Attachments) > 0 {
		fields = append(fields, mattermost.AttachmentField{
			Title: "Вложения", Value: strconv.Itoa(len(detail.Attachments)), Short: true,
		})
	}

	card := mattermost.Attachment{
		Title:   title,
		Text:    detailText(detail.Description),
		Color:   mmStatusColors[detail.Status],
		Fields:  fields,
		Buttons: s.detailActionButtons(detail),
	}
	if detail.Link != "" {
		card.TitleLink = detail.Link
	}
	return card
}

// detailActionButtons — действия владельца на карточке заявки. Правила те же, что
// в плагине: отмена только для новой заявки, подтверждение решения и возврат в
// работу — только для решённой. Заявка не владельца остаётся без кнопок.
func (s *MattermostService) detailActionButtons(detail *models.PluginTicketDetail) []mattermost.InteractiveButton {
	if detail.Owner == nil {
		return nil
	}
	btnCtx := func(action string, status models.TicketStatus) map[string]string {
		return map[string]string{
			"action":    action,
			"ticket_id": detail.ID.String(),
			"status":    string(status),
		}
	}

	switch detail.Status {
	case models.StatusOpen:
		if !detail.CanCancel {
			return nil
		}
		return []mattermost.InteractiveButton{{
			Text: "Отменить заявку", Style: "danger",
			URL: s.actionURL(), Context: btnCtx(actionCancel, models.StatusCancelled),
			ID: ticketActionID(detail.ID, actionCancel),
		}}
	case models.StatusResolved:
		var buttons []mattermost.InteractiveButton
		if detail.CanConfirm {
			buttons = append(buttons, mattermost.InteractiveButton{
				Text: "Подтвердить решение", Style: "primary",
				URL: s.actionURL(), Context: btnCtx(actionConfirm, models.StatusClosed),
				ID: ticketActionID(detail.ID, actionConfirm),
			})
		}
		if detail.CanReopen {
			buttons = append(buttons, mattermost.InteractiveButton{
				Text: "Вернуть в работу",
				URL:  s.actionURL(), Context: btnCtx(actionReopen, models.StatusInProgress),
				ID: ticketActionID(detail.ID, actionReopen),
			})
		}
		return buttons
	default:
		return nil
	}
}

// detailText — описание заявки для карточки: переносы схлопнуты в абзацы, а не
// в одну строку (здесь, в отличие от списка, описание — содержание карточки),
// и подрезано до cardDetailLimit.
func detailText(text string) string {
	paragraphs := strings.Split(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\n")
	kept := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	out := strings.Join(kept, "\n")
	if len([]rune(out)) <= cardDetailLimit {
		return out
	}
	return string([]rune(out)[:cardDetailLimit]) + "…"
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
