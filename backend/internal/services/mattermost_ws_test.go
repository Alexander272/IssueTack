package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// realmSettings — настройки интеграции, как их ждут WS-обработчики: активная
// интеграция с бот-токеном и id бота.
func realmSettings(realmID uuid.UUID) *models.RealmMattermost {
	return &models.RealmMattermost{
		RealmID:   realmID,
		BotToken:  "bot-token",
		BotUserID: "bot-id",
		IsActive:  true,
	}
}

// -------------------- ограничитель лавины входящих сообщений --------------------

// TestInboundGuard_LimitsFlood проверяет основной сценарий предохранителя: до
// лимита события проходят, двадцать первое включает мьют (и ровно один лог),
// мьют держится всё окно, а после него диалог оживает.
func TestInboundGuard_LimitsFlood(t *testing.T) {
	guard := newInboundGuard()
	now := time.Now()

	for i := 1; i <= inboundDMEventLimit; i++ {
		allowed, tripped := guard.allow("realm:dm1", now)
		require.True(t, allowed, "событие %d должно пройти", i)
		require.False(t, tripped)
	}

	allowed, tripped := guard.allow("realm:dm1", now)
	assert.False(t, allowed, "двадцать первое событие — уже лавина")
	assert.True(t, tripped, "о срабатывании предупреждаем один раз")

	for _, offset := range []time.Duration{time.Second, time.Minute, inboundDMMutePeriod - time.Second} {
		allowed, tripped = guard.allow("realm:dm1", now.Add(offset))
		assert.False(t, allowed, "пока действует мьют, событий не обрабатываем")
		assert.False(t, tripped, "повторного лога на каждое событие нет")
	}

	allowed, tripped = guard.allow("realm:dm1", now.Add(inboundDMMutePeriod+time.Second))
	assert.True(t, allowed, "после мьюта диалог снова работает")
	assert.False(t, tripped)
}

// TestInboundGuard_ResetsWindow закрепляет, что счётчик живёт в окне: медленная
// переписка (по событию в несколько минут) ограничитель не включает.
func TestInboundGuard_ResetsWindow(t *testing.T) {
	guard := newInboundGuard()
	now := time.Now()

	for i := 0; i < inboundDMEventLimit; i++ {
		allowed, _ := guard.allow("realm:dm1", now)
		require.True(t, allowed)
	}

	allowed, tripped := guard.allow("realm:dm1", now.Add(inboundDMLimitPeriod+time.Second))
	assert.True(t, allowed, "окно истекло — счётчик начинается заново")
	assert.False(t, tripped)
}

// TestInboundGuard_IsolatesChannels: предохранитель привязан к диалогу, соседей
// он не задевает.
func TestInboundGuard_IsolatesChannels(t *testing.T) {
	guard := newInboundGuard()
	now := time.Now()

	for i := 0; i < inboundDMEventLimit; i++ {
		guard.allow("realm:dm1", now)
	}

	allowed, tripped := guard.allow("realm:dm2", now)
	assert.True(t, allowed)
	assert.False(t, tripped)
}

// TestInboundGuard_EmptyKeyAllowed: событие без канала (неизвестный ключ)
// ограничивать нечем — оно проходит.
func TestInboundGuard_EmptyKeyAllowed(t *testing.T) {
	guard := newInboundGuard()

	for i := 0; i < inboundDMEventLimit*2; i++ {
		allowed, tripped := guard.allow("", time.Now())
		require.True(t, allowed)
		require.False(t, tripped)
	}
}

// -------------------- handleWSEvent: фильтры и ограничитель --------------------

// TestHandleWSEvent_IgnoresNonUserPosts — страховка на уровне сервиса: даже если
// пост проскочил фильтр WS-клиента, собственные посты бота и системные сообщения
// Mattermost не должны доходить до диспетчера (иначе бот отвечает сам себе).
func TestHandleWSEvent_IgnoresNonUserPosts(t *testing.T) {
	realmID := uuid.New()

	tests := []struct {
		name string
		post *model.Post
	}{
		{
			name: "собственный пост бота",
			post: &model.Post{UserId: "bot-id", ChannelId: "dm1", Message: "привет"},
		},
		{
			name: "системное сообщение",
			post: &model.Post{UserId: "mm1", ChannelId: "dm1", Type: model.PostTypeAddToChannel, Message: "добавил бота в канал"},
		},
		{
			name: "пост без автора",
			post: &model.Post{ChannelId: "dm1", Message: "привет"},
		},
		{
			name: "пустое сообщение",
			post: &model.Post{UserId: "mm1", ChannelId: "dm1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var posts []capturedPost
			svc, repo, _, _, _ := dmService(t, &posts)

			svc.handleWSEvent(context.Background(), realmID, "bot-id", mattermost.PostedEvent{
				ChannelID:   tt.post.ChannelId,
				ChannelType: string(model.ChannelTypeDirect),
				Post:        tt.post,
			})

			assert.Empty(t, posts, "на такой пост бот не отвечает")
			repo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
		})
	}
}

// TestHandleWSEvent_MutesFloodedDirectMessage — предохранитель на уровне сервиса:
// лавина входящих в личном диалоге (признак цикла «бот — сам себе») гасит канал,
// а в обычном канале лимита нет — там много входящих сообщений это норма.
func TestHandleWSEvent_MutesFloodedDirectMessage(t *testing.T) {
	realmID := uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)
	repo.On("GetByRealm", mock.Anything, realmID).Return(realmSettings(realmID), nil)

	event := func(channelID, channelType string) mattermost.PostedEvent {
		return mattermost.PostedEvent{
			ChannelID:   channelID,
			ChannelType: channelType,
			Post:        &model.Post{UserId: "mm1", ChannelId: channelID, Message: "заявка"},
		}
	}

	for i := 0; i < inboundDMEventLimit; i++ {
		svc.handleWSEvent(context.Background(), realmID, "bot-id", event("dm1", string(model.ChannelTypeDirect)))
	}
	require.Len(t, posts, inboundDMEventLimit, "до лимита бот отвечает на каждое сообщение")

	svc.handleWSEvent(context.Background(), realmID, "bot-id", event("dm1", string(model.ChannelTypeDirect)))
	assert.Len(t, posts, inboundDMEventLimit, "после лимита диалог замолкает")

	for i := 0; i < inboundDMEventLimit*2; i++ {
		svc.handleWSEvent(context.Background(), realmID, "bot-id", event("ch2", string(model.ChannelTypeOpen)))
	}
	assert.Len(t, posts, inboundDMEventLimit*3, "в открытом канале ограничителя нет")
}

// TestHandleWSEvent_FloodNotifiesErrorBot — при срабатывании ограничителя одной
// записи в логе мало: аварию нужно увидеть разработчику, поэтому дополнительно
// уходит уведомление в error_bot (настроенный через ERR_URL). Лог и уведомление
// дополняют друг друга, а не заменяют.
func TestHandleWSEvent_FloodNotifiesErrorBot(t *testing.T) {
	realmID := uuid.New()

	notified := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		select {
		case notified <- string(body):
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("ERR_URL", server.URL)

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)
	repo.On("GetByRealm", mock.Anything, realmID).Return(realmSettings(realmID), nil)

	event := mattermost.PostedEvent{
		ChannelID:   "dm1",
		ChannelType: string(model.ChannelTypeDirect),
		Post:        &model.Post{UserId: "mm1", ChannelId: "dm1", Message: "заявка"},
	}
	for i := 0; i < inboundDMEventLimit+1; i++ {
		svc.handleWSEvent(context.Background(), realmID, "bot-id", event)
	}

	select {
	case body := <-notified:
		assert.Contains(t, body, "mattermost DM message flood detected, channel muted")
		assert.Contains(t, body, realmID.String())
		assert.Contains(t, body, "dm1")
	case <-time.After(time.Second):
		t.Fatal("о срабатывании ограничителя не отправлено уведомление в error_bot")
	}
}

// -------------------- resolveBotUserID --------------------

// TestResolveBotUserID_HealsDivergentSavedValue: id бота берётся из токена, а
// расхождение с колонкой залечивается — иначе фильтр своих сообщений пропускает
// посты бота и он отвечает сам себе.
func TestResolveBotUserID_HealsDivergentSavedValue(t *testing.T) {
	realmID := uuid.New()
	capture := &mmCapture{posts: &[]capturedPost{}, botID: "actual-bot-id"}
	svc, repo, _, _, _ := dmFixtureService(t, capture, nil)

	settings := &models.RealmMattermost{
		RealmID:   realmID,
		BotToken:  "bot-token",
		BotUserID: "stale-bot-id",
		IsActive:  true,
	}
	repo.On("Upsert", mock.Anything, nil, mock.MatchedBy(func(dto *models.RealmMattermost) bool {
		return dto.RealmID == realmID && dto.BotUserID == "actual-bot-id" && dto.BotToken == "bot-token"
	})).Return(nil)

	got := svc.resolveBotUserID(context.Background(), settings)

	assert.Equal(t, "actual-bot-id", got, "источник истины — токен")
	assert.Equal(t, "actual-bot-id", settings.BotUserID, "колонка залечена в самой модели")
	repo.AssertCalled(t, "Upsert", mock.Anything, nil, mock.Anything)
}

// TestResolveBotUserID_KeepsMatchingValue: при совпадении лишней записи в БД нет.
func TestResolveBotUserID_KeepsMatchingValue(t *testing.T) {
	realmID := uuid.New()
	capture := &mmCapture{posts: &[]capturedPost{}, botID: "bot-id"}
	svc, repo, _, _, _ := dmFixtureService(t, capture, nil)

	settings := realmSettings(realmID)

	assert.Equal(t, "bot-id", svc.resolveBotUserID(context.Background(), settings))
	repo.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything, mock.Anything)
}

// TestResolveBotUserID_FallsBackOnError: недоступный Mattermost не мешает запуску
// — работаем на сохранённом значении, ничего не перезаписывая.
func TestResolveBotUserID_FallsBackOnError(t *testing.T) {
	realmID := uuid.New()
	// botID не задан — заглушка отвечает ошибкой, как недоступный Mattermost.
	capture := &mmCapture{posts: &[]capturedPost{}}
	svc, repo, _, _, _ := dmFixtureService(t, capture, nil)

	settings := &models.RealmMattermost{
		RealmID:   realmID,
		BotToken:  "bot-token",
		BotUserID: "saved-bot-id",
		IsActive:  true,
	}

	assert.Equal(t, "saved-bot-id", svc.resolveBotUserID(context.Background(), settings))
	assert.Equal(t, "saved-bot-id", settings.BotUserID)
	repo.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything, mock.Anything)
}
