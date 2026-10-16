package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

// StartWSForRealm запускает веб-сокет прослушивания сообщений Mattermost для
// realm, если он ещё не запущен и интеграция активна. Идемпотентно: повторный
// вызов для уже запущенного realm ничего не делает.
func (s *MattermostService) StartWSForRealm(ctx context.Context, realmID uuid.UUID) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	realmKey := realmID.String()

	if _, ok := s.wsClients[realmKey]; ok {
		return nil
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return fmt.Errorf("failed to get mattermost settings: %w", err)
	}
	if !settings.IsActive || settings.BotToken == "" {
		return nil
	}

	botUserID := s.resolveBotUserID(ctx, settings)

	wsClient := mattermost.NewWSClient(s.most.Client.URL(), botUserID,
		func(ctx context.Context, event mattermost.PostedEvent) {
			s.handleWSEvent(ctx, realmID, botUserID, event)
		},
		func(ctx context.Context, event mattermost.DirectAddedEvent) {
			s.handleDirectAddedEvent(ctx, realmID, botUserID, event)
		},
	)

	s.wsClients[realmKey] = wsClient

	go wsClient.Run(ctx, settings.BotToken)

	logger.Info("mattermost WS client started", logger.StringAttr("realm_id", realmKey))
	return nil
}

// resolveBotUserID возвращает id бота, авторитетный для текущего токена.
// Источник истины — токен: колонка bot_user_id могла остаться от прежнего токена
// (в т.ч. после правки руками в БД), а от неё зависит фильтр своих сообщений в
// веб-сокете. С устаревшим id пост бота проходит фильтр, и бот отвечает сам себе
// бесконечно. Расхождение с колонкой залечиваем, чтобы тот же id использовали и
// HTTP-пути. Ошибка запроса запуск не ломает: работаем на сохранённом значении.
func (s *MattermostService) resolveBotUserID(ctx context.Context, settings *models.RealmMattermost) string {
	me, err := s.most.Client.GetMe(settings.BotToken)
	if err != nil {
		logger.Warn("failed to resolve mattermost bot user id, using saved value",
			logger.StringAttr("realm_id", settings.RealmID.String()),
			logger.ErrAttr(err),
		)
		return settings.BotUserID
	}
	if me.ID == settings.BotUserID {
		return me.ID
	}

	logger.Warn("mattermost bot user id differs from saved value, healing",
		logger.StringAttr("realm_id", settings.RealmID.String()),
		logger.StringAttr("saved_bot_user_id", settings.BotUserID),
		logger.StringAttr("actual_bot_user_id", me.ID),
	)

	settings.BotUserID = me.ID
	if err := s.repo.Upsert(ctx, nil, settings); err != nil {
		logger.Error("failed to save healed mattermost bot user id",
			logger.StringAttr("realm_id", settings.RealmID.String()),
			logger.ErrAttr(err),
		)
	}
	return me.ID
}

// StopWSForRealm останавливает веб-сокет указанного realm (если он запущен).
func (s *MattermostService) StopWSForRealm(realmID uuid.UUID) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	realmKey := realmID.String()
	if client, ok := s.wsClients[realmKey]; ok {
		client.Stop()
		delete(s.wsClients, realmKey)
		logger.Info("mattermost WS client stopped", logger.StringAttr("realm_id", realmKey))
	}
}

// StopAllWS останавливает все запущенные веб-сокеты Mattermost (например,
// при завершении приложения).
func (s *MattermostService) StopAllWS() {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	for key, client := range s.wsClients {
		client.Stop()
		delete(s.wsClients, key)
	}
	logger.Info("all mattermost WS clients stopped")
}

// StartAllActiveWS запускает при старте приложения веб-сокеты для всех
// активных интеграций Mattermost. Реализовано в отдельном методе, чтобы
// вынесенная в начале проверка base_url позволяла заранее предупредить
// о неработоспособности интерактивных кнопок, пока сервис ещё поднимается.
func (s *MattermostService) StartAllActiveWS(ctx context.Context) {
	if s.baseURL == "" {
		logger.Warn("http.base_url is not configured, Mattermost interactive features will not work")
	}

	settings, err := s.repo.GetActive(ctx)
	if err != nil {
		logger.Error("failed to get active mattermost settings for startup", logger.ErrAttr(err))
		return
	}

	for _, st := range settings {
		if err := s.StartWSForRealm(ctx, st.RealmID); err != nil {
			logger.Error("failed to start WS for realm on startup",
				logger.StringAttr("realm_id", st.RealmID.String()),
				logger.ErrAttr(err),
			)
		}
	}
}

// handleWSEvent диспетчеризует входящее сообщение из веб-сокета Mattermost
// по той же логике, что и личные сообщения (HandleDM). Файлы вместе с
// командой создания заявки откладываются в pendingFiles по ключу
// «канал:пользователь», чтобы привязать их к заявке после отправки диалога —
// Mattermost не связывает файлы с диалогом напрямую.
//
// botUserID — id бота, авторитетный для токена (см. resolveBotUserID): по нему
// повторно отсеиваются собственные посты и системные сообщения. Фильтр есть и в
// WS-клиенте (AcceptPost), здесь — страховка на уровне сервиса: попади такой
// пост в диспетчер, бот ответил бы справкой сам себе и зациклился.
func (s *MattermostService) handleWSEvent(ctx context.Context, realmID uuid.UUID, botUserID string, event mattermost.PostedEvent) {
	if event.Post == nil {
		return
	}

	userID := event.Post.UserId
	channelID := event.ChannelID
	message := event.Post.Message
	fileIDs := event.Post.FileIds

	if userID == "" || userID == botUserID || event.Post.Type != "" {
		return
	}

	hasFiles := len(fileIDs) > 0
	if message == "" && !hasFiles {
		return
	}

	// Лавина входящих сообщений в личном диалоге — признак того, что бот
	// переписывается сам с собой. Гасим канал на время, чтобы цикл не завалил
	// диалог тысячами постов, даже если фильтр своих сообщений пропустил их
	// (например, id бота не удалось получить и колонка содержит устаревшее
	// значение). Ограничиваем только личные диалоги: в канале много входящих
	// сообщений — это обычная работа, а не цикл.
	if event.ChannelType == string(model.ChannelTypeDirect) {
		allowed, tripped := s.wsGuard.allow(realmID.String()+":"+channelID, time.Now())
		if tripped {
			logger.Error("mattermost DM message flood detected, channel muted",
				logger.StringAttr("realm_id", realmID.String()),
				logger.StringAttr("channel_id", channelID),
				logger.IntAttr("limit", inboundDMEventLimit),
			)
			// Лог виден только в логах приложения, а срабатывание ограничителя —
			// аварийный признак петли «бот — сам себе», которую нужно гасить
			// вручную (перечитать настройки/перезапустить интеграцию). Поэтому
			// дополнительно сигнализируем разработчику через error_bot.
			error_bot.Send(nil, "mattermost DM message flood detected, channel muted", map[string]string{
				"realm_id":   realmID.String(),
				"channel_id": channelID,
				"limit":      strconv.Itoa(inboundDMEventLimit),
			})
		}
		if !allowed {
			return
		}
	}

	logger.Debug("mattermost WS event received",
		logger.StringAttr("realm_id", realmID.String()),
		logger.StringAttr("user_id", userID),
	)

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		logger.Error("failed to get realm settings for WS event",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return
	}

	msg := strings.TrimSpace(message)

	ch := &mmChannel{
		Settings:  settings,
		MmUserID:  userID,
		ChannelID: channelID,
	}

	switch {
	case syncCommands.MatchString(msg):
		if err := s.handleSync(ctx, ch, msg); err != nil {
			logger.Error("failed to handle sync from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case createCommands.MatchString(msg):
		if len(event.Post.FileIds) > 0 {
			s.pendingFiles.Store(channelID+":"+userID, event.Post.FileIds)
		}
		if err := s.sendMenu(settings.BotToken, channelID, realmID.String(),
			"Для оформления заявки нажмите на кнопку ниже"); err != nil {
			logger.Error("failed to send bot menu from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case helpCommands.MatchString(msg):
		isAdmin := s.checkIsAdmin(ctx, realmID, userID)
		if err := s.sendHelpMessage(settings.BotToken, channelID, realmID.String(), isAdmin); err != nil {
			logger.Error("failed to send help message from WS", logger.ErrAttr(err))
		}

	case statusCommands.MatchString(msg):
		if err := s.sendStatusMessage(ctx, ch); err != nil {
			logger.Error("failed to send status message from WS", logger.ErrAttr(err))
		}

	case attachCommands.MatchString(msg) && len(event.Post.FileIds) > 0:
		parts := attachCommands.FindStringSubmatch(msg)
		number, _ := strconv.Atoi(parts[1])
		if err := s.handleAttachFiles(ctx, ch, number, event.Post.FileIds, ""); err != nil {
			logger.Error("failed to handle attach from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case len(event.Post.FileIds) > 0 && !attachCommands.MatchString(msg):
		if err := s.handleTextWithFiles(ctx, ch, msg, event.Post.FileIds); err != nil {
			logger.Error("failed to handle text with files from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	case commentCommands.MatchString(msg):
		if err := s.handleComment(ctx, ch, msg); err != nil {
			logger.Error("failed to handle comment from WS",
				logger.StringAttr("user_id", userID),
				logger.ErrAttr(err),
			)
		}

	default:
		if len(event.Post.FileIds) > 0 {
			if err := s.handleAttachFiles(ctx, ch, 0, event.Post.FileIds, ""); err != nil {
				logger.Error("failed to handle auto-attach from WS",
					logger.StringAttr("user_id", userID),
					logger.ErrAttr(err),
				)
			}
			return
		}
		isAdmin := s.checkIsAdmin(ctx, realmID, userID)
		if err := s.sendHelpMessage(settings.BotToken, channelID, realmID.String(), isAdmin); err != nil {
			logger.Error("failed to send help message from WS", logger.ErrAttr(err))
		}
	}
}

// handleDirectAddedEvent реагирует на создание личного диалога бота с
// пользователем: шлёт приветствие один раз. Событие приходит на создание канала,
// поэтому срабатывает, только когда диалог заводит сам пользователь
// (creator_id — человек, teammate_id — бот). Диалог, созданный ботом при
// отправке уведомления (creator_id == bot), пропускаем: приветствие тут было бы
// непрошеным сообщением. Второй триггер — открытие диалога из webapp-плагина
// (PluginGreeting); повторную отправку отсекает отметка в БД.
func (s *MattermostService) handleDirectAddedEvent(ctx context.Context, realmID uuid.UUID, botUserID string, event mattermost.DirectAddedEvent) {
	if event.CreatorID == "" || event.CreatorID == botUserID || event.TeammateID != botUserID {
		return
	}
	if event.ChannelID == "" {
		return
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		logger.Error("failed to get realm settings for direct_added",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return
	}

	user, err := s.resolveOrCreateUser(ctx, settings.RealmID, event.CreatorID, nil)
	if err != nil {
		logger.Error("failed to resolve user for direct_added",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return
	}

	if err := s.sendGreeting(ctx, settings, user.ID, event.ChannelID); err != nil {
		logger.Error("failed to send greeting on direct_added",
			logger.StringAttr("user_id", user.ID.String()),
			logger.ErrAttr(err),
		)
	}
}

const (
	// inboundDMEventLimit — сколько входящих сообщений в личном диалоге с ботом
	// допустимо за окно. Живая переписка укладывается в считанные сообщения в
	// минуту, поэтому 20 событий с большим запасом отделяют обычную работу от
	// цикла «бот — сам себе».
	inboundDMEventLimit = 20
	// inboundDMLimitPeriod — окно подсчёта входящих событий.
	inboundDMLimitPeriod = time.Minute
	// inboundDMMutePeriod — на сколько диалог замолкает после срабатывания.
	inboundDMMutePeriod = 5 * time.Minute
)

// inboundGuard — предохранитель от лавины входящих сообщений в личном диалоге.
// Если бот и Mattermost начнут отвечать друг другу, каждое сообщение бота
// вернётся как входящее; без ограничителя такой цикл заваливает диалог тысячами
// постов (наблюдалось ~4000). Считаем именно входящие события, а не отправленные
// посты: одно сообщение пользователя законно порождает десятки постов (список
// «Мои заявки» бьётся на сообщения), поэтому лимит на исходящие был бы неверен.
type inboundGuard struct {
	mu      sync.Mutex
	windows map[string]*inboundWindow
}

// inboundWindow — состояние одного диалога: счётчик в текущем окне и время, до
// которого диалог замьючен после срабатывания.
type inboundWindow struct {
	windowStart time.Time
	events      int
	mutedUntil  time.Time
}

func newInboundGuard() *inboundGuard {
	return &inboundGuard{windows: make(map[string]*inboundWindow)}
}

// allow сообщает, можно ли обрабатывать событие. tripped == true означает, что
// ограничитель сработал именно сейчас: вызывающий пишет один лог и молчит, пока
// не истечёт мьют. Пустой ключ (неизвестный канал) не ограничивается.
func (g *inboundGuard) allow(key string, now time.Time) (allowed, tripped bool) {
	if key == "" {
		return true, false
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	w := g.windows[key]
	if w == nil {
		w = &inboundWindow{}
		g.windows[key] = w
	}

	if now.Before(w.mutedUntil) {
		return false, false
	}

	if w.windowStart.IsZero() || now.Sub(w.windowStart) >= inboundDMLimitPeriod {
		w.windowStart = now
		w.events = 0
		g.prune(now)
	}

	w.events++
	if w.events > inboundDMEventLimit {
		w.mutedUntil = now.Add(inboundDMMutePeriod)
		w.events = 0
		return false, true
	}
	return true, false
}

// prune убирает записи диалогов, в которых давно ничего не происходило, чтобы
// карта не росла с каждым новым диалогом. Вызывается при перезапуске окна.
func (g *inboundGuard) prune(now time.Time) {
	for key, w := range g.windows {
		if now.After(w.mutedUntil) && now.Sub(w.windowStart) > inboundDMMutePeriod {
			delete(g.windows, key)
		}
	}
}
