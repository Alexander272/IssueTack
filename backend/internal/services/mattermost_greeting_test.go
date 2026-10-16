package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestSendGreeting_FirstTime: первая отправка помечает пользователя и уходит
// личным сообщением со скриншотом кнопки «Создать заявку».
func TestSendGreeting_FirstTime(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(true, nil)

	require.NoError(t, svc.sendGreeting(context.Background(), realmSettings(realmID), userID, "dm1"))

	require.Len(t, posts, 1)
	assert.Equal(t, "dm1", posts[0].channelID)
	assert.Contains(t, posts[0].message, "На связи чат-бот IT-поддержки")
	assert.Contains(t, posts[0].message, "«Создать заявку»")
	assert.Equal(t, []string{"f1"}, posts[0].fileIDs, "к приветствию приложен скриншот")
}

// TestSendGreeting_AlreadyGreeted: повторно приветствие не шлётся — отметка в БД
// уже стоит, второй триггер (например, открытие диалога после direct_added)
// должен молчать.
func TestSendGreeting_AlreadyGreeted(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(false, nil)

	require.NoError(t, svc.sendGreeting(context.Background(), realmSettings(realmID), userID, "dm1"))

	assert.Empty(t, posts, "второй раз приветствие не отправляем")
}

// TestSendGreeting_ReleasesClaimOnFailure: если отправка не удалась, пометка
// снимается — иначе пользователь остался бы без приветствия навсегда.
func TestSendGreeting_ReleasesClaimOnFailure(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmFixtureService(t, &mmCapture{posts: &posts, failPosts: true}, nil)
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(true, nil)
	repo.On("ReleaseGreeting", mock.Anything, realmID, userID).Return(nil)

	err := svc.sendGreeting(context.Background(), realmSettings(realmID), userID, "dm1")

	require.Error(t, err, "ошибка отправки пробрасывается вызывающему")
	assert.Empty(t, posts)
	repo.AssertCalled(t, "ReleaseGreeting", mock.Anything, realmID, userID)
}

// TestPluginGreeting_SendsOncePerUser: открытие диалога из плагина шлёт
// приветствие один раз; повторный вызов (переключение каналов) молчит.
func TestPluginGreeting_SendsOncePerUser(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, repo, users, userRealms, _ := dmService(t, &posts)
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(realmSettings(realmID), nil)
	expectExistingUser(users, userRealms, userID, realmID)
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(true, nil).Once()
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(false, nil).Once()

	require.NoError(t, svc.PluginGreeting(context.Background(), pluginScopeFixture("ch1")))
	require.NoError(t, svc.PluginGreeting(context.Background(), pluginScopeFixture("ch1")))

	assert.Len(t, posts, 1, "приветствие приходит один раз")
}

// TestPluginGreeting_UnboundChannelNoop: канал, не привязанный к реалму, — не
// ошибка, приветствия просто нет.
func TestPluginGreeting_UnboundChannelNoop(t *testing.T) {
	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, assert.AnError)

	require.NoError(t, svc.PluginGreeting(context.Background(), pluginScopeFixture("ch1")))

	assert.Empty(t, posts)
	repo.AssertNotCalled(t, "ClaimGreeting", mock.Anything, mock.Anything, mock.Anything)
}

// TestHandleDirectAddedEvent_GreetsUserInitiatedDM: пользователь сам создал
// диалог с ботом (creator — человек, teammate — бот) — шлём приветствие.
func TestHandleDirectAddedEvent_GreetsUserInitiatedDM(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, repo, users, userRealms, _ := dmService(t, &posts)
	repo.On("GetByRealm", mock.Anything, realmID).Return(realmSettings(realmID), nil)
	expectExistingUser(users, userRealms, userID, realmID)
	repo.On("ClaimGreeting", mock.Anything, realmID, userID).Return(true, nil)

	svc.handleDirectAddedEvent(context.Background(), realmID, "bot-id", mattermost.DirectAddedEvent{
		ChannelID:  "dm1",
		CreatorID:  "mm1",
		TeammateID: "bot-id",
	})

	require.Len(t, posts, 1)
	assert.Equal(t, "dm1", posts[0].channelID)
}

// TestHandleDirectAddedEvent_IgnoresBotInitiatedDM: диалог создал сам бот
// (рассылка уведомления) — приветствие было бы непрошеным сообщением.
func TestHandleDirectAddedEvent_IgnoresBotInitiatedDM(t *testing.T) {
	realmID := uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)

	svc.handleDirectAddedEvent(context.Background(), realmID, "bot-id", mattermost.DirectAddedEvent{
		ChannelID:  "dm1",
		CreatorID:  "bot-id",
		TeammateID: "mm1",
	})

	assert.Empty(t, posts)
	repo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "ClaimGreeting", mock.Anything, mock.Anything, mock.Anything)
}

// TestHandleDirectAddedEvent_IgnoresForeignDM: второго участника диалога нет
// среди ботов реалма — это не диалог с нашим ботом.
func TestHandleDirectAddedEvent_IgnoresForeignDM(t *testing.T) {
	realmID := uuid.New()

	var posts []capturedPost
	svc, repo, _, _, _ := dmService(t, &posts)

	svc.handleDirectAddedEvent(context.Background(), realmID, "bot-id", mattermost.DirectAddedEvent{
		ChannelID:  "dm1",
		CreatorID:  "mm1",
		TeammateID: "other-bot",
	})

	assert.Empty(t, posts)
	repo.AssertNotCalled(t, "ClaimGreeting", mock.Anything, mock.Anything, mock.Anything)
}
