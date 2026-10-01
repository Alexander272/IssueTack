package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testBaseURL = "https://issuetrack.example.com"

// capturedPost — пост, который бот отправил в Mattermost.
type capturedPost struct {
	channelID string
	message   string
	props     model.StringInterface
}

// mmCapture перехватывает обращения бота к Mattermost: посты, открытие диалогов,
// удаление и правку сообщений, а также список пользователей для синка.
type mmCapture struct {
	posts          *[]capturedPost
	dialogs        []model.OpenDialogRequest
	deletedPostIDs []string
	patches        map[string]capturedPost
	// users отдаётся на GET /api/v4/users (синк всех пользователей сервера),
	// teamUsers — на тот же запрос с фильтром in_team (синк по командам).
	users     []*model.User
	teamUsers map[string][]*model.User
}

// mmServer поднимает Mattermost-заглушку: сохраняет отправленные посты и
// открытые диалоги, запоминает удалённые и правленые сообщения, отвечает так,
// как их ждёт model.Client4.
func mmServer(t *testing.T, capture *mmCapture) *httptest.Server {
	t.Helper()
	if capture.patches == nil {
		capture.patches = map[string]capturedPost{}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v4/posts":
			assert.Equal(t, http.MethodPost, r.Method)
			var sent model.Post
			require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
			*capture.posts = append(*capture.posts, capturedPost{
				channelID: sent.ChannelId,
				message:   sent.Message,
				props:     sent.Props,
			})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"p1","channel_id":"` + sent.ChannelId + `"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v4/posts/") && strings.HasSuffix(r.URL.Path, "/patch"):
			assert.Equal(t, http.MethodPut, r.Method)
			postID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v4/posts/"), "/patch")
			var patch model.PostPatch
			require.NoError(t, json.NewDecoder(r.Body).Decode(&patch))
			patched := capturedPost{}
			if patch.Message != nil {
				patched.message = *patch.Message
			}
			if patch.Props != nil {
				patched.props = *patch.Props
			}
			capture.patches[postID] = patched
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + postID + `"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v4/posts/"):
			assert.Equal(t, http.MethodDelete, r.Method)
			capture.deletedPostIDs = append(capture.deletedPostIDs,
				strings.TrimPrefix(r.URL.Path, "/api/v4/posts/"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"OK"}`))

		case r.URL.Path == "/api/v4/channels/direct":
			// Бот открывает DM-канал перед отправкой личного сообщения.
			var sent []string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"dm1"}`))

		case r.URL.Path == "/api/v4/actions/dialogs/open":
			var sent model.OpenDialogRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
			capture.dialogs = append(capture.dialogs, sent)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"errors":null}`))

		case r.URL.Path == "/api/v4/users":
			// Синк пользователей. Запрос постраничный: клиент идёт по страницам,
			// пока не отдаст меньше per_page записей, поэтому на первой странице
			// отдаём весь список, на остальных — пусто.
			page := r.URL.Query().Get("page")
			list := capture.users
			if teamID := r.URL.Query().Get("in_team"); teamID != "" {
				list = capture.teamUsers[teamID]
			}
			if page != "0" {
				list = nil
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mustJSON(t, list)))

		case strings.HasPrefix(r.URL.Path, "/api/v4/teams/name/"):
			// Синк по конкретным командам: имя команды используется как её id,
			// чтобы GetUsersInTeam(in_team=...) совпал с ключом teamUsers.
			name := strings.TrimPrefix(r.URL.Path, "/api/v4/teams/name/")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mustJSON(t, &model.Team{Id: name, Name: name})))

		default:
			require.Failf(t, "unexpected mattermost call", "%s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// mustJSON кодирует значение для ответа Mattermost-заглушки.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

// postsServer перехватывает POST /api/v4/posts и пишет отправленные посты в
// captured. Ответ сервера — минимальный созданный пост, как его ждёт model.Client4.
func postsServer(t *testing.T, captured *[]capturedPost) *httptest.Server {
	t.Helper()
	return mmServer(t, &mmCapture{posts: captured})
}

// postObjects достаёт массив объектов из props/карточки. После похода через
// JSON значения приходят как []any, поэтому приводим к общему виду.
func postObjects(t *testing.T, raw any) []model.StringInterface {
	t.Helper()
	var items []any
	switch v := raw.(type) {
	case []model.StringInterface:
		items = make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
	case []any:
		items = v
	default:
		require.Failf(t, "unexpected type of list", "%T", raw)
	}

	out := make([]model.StringInterface, 0, len(items))
	for _, item := range items {
		switch obj := item.(type) {
		case model.StringInterface:
			out = append(out, obj)
		case map[string]any:
			out = append(out, model.StringInterface(obj))
		default:
			require.Failf(t, "unexpected type of item", "%T", item)
		}
	}
	return out
}

// postCards достаёт карточки поста (пропуская ведущую карточку с рядом кнопок).
func postCards(t *testing.T, props model.StringInterface) []model.StringInterface {
	t.Helper()
	raw, ok := props["attachments"]
	require.True(t, ok, "post must contain attachments")

	cards := make([]model.StringInterface, 0)
	for _, a := range postObjects(t, raw) {
		if _, isActionRow := a["actions"]; isActionRow && len(a) == 1 {
			continue
		}
		cards = append(cards, a)
	}
	return cards
}

// postActionRow достаёт ряд кнопок поста.
func postActionRow(t *testing.T, props model.StringInterface) []model.StringInterface {
	t.Helper()
	raw, ok := props["attachments"]
	require.True(t, ok, "post must contain attachments")

	attachments := postObjects(t, raw)
	require.NotEmpty(t, attachments, "post must have a leading action row")
	require.Equal(t, 1, len(attachments[0]), "action row must not mix with card fields")

	actions, ok := attachments[0]["actions"]
	require.True(t, ok, "action row must have actions")
	return postObjects(t, actions)
}

// postCardActions достаёт ряд кнопок карточки (интерактивных и ссылок).
func postCardActions(t *testing.T, card model.StringInterface) []model.StringInterface {
	t.Helper()
	raw, ok := card["actions"]
	if !ok {
		return nil
	}
	return postObjects(t, raw)
}

// stringMap приводит значение к map[string]any независимо от того, прошло ли
// оно через JSON (после похода по сети именованный тип теряется).
func stringMap(t *testing.T, raw any, what string) map[string]any {
	t.Helper()
	switch v := raw.(type) {
	case model.StringInterface:
		return v
	case map[string]any:
		return v
	}
	require.Failf(t, "unexpected value", "%s must be an object, got %T", what, raw)
	return nil
}

// actionContext достаёт integration.context кнопки.
func actionContext(t *testing.T, action model.StringInterface) map[string]string {
	t.Helper()
	integration := stringMap(t, action["integration"], "integration")
	raw := stringMap(t, integration["context"], "context")

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		text, ok := value.(string)
		require.Truef(t, ok, "context value %q must be a string, got %T", key, value)
		out[key] = text
	}
	return out
}

// cardActionsByName перечисляет кнопки карточки как «имя: действие», чтобы
// проверять набор кнопок списка «Мои заявки» одной строкой.
func cardActionsByName(t *testing.T, card model.StringInterface) []string {
	t.Helper()
	names := make([]string, 0)
	for _, action := range postCardActions(t, card) {
		name, _ := action["name"].(string)
		actionType, _ := action["type"].(string)
		switch actionType {
		case "button":
			names = append(names, name+":"+actionContext(t, action)["action"])
		default:
			names = append(names, name+":"+actionType)
		}
	}
	return names
}

// cardActionIDs собирает action_id всех действий карточки — Mattermost ищет
// действие по id в attachments поста и берёт первое совпадение, поэтому id
// обязан быть непустым и уникальным в пределах поста.
func cardActionIDs(t *testing.T, card model.StringInterface) []string {
	t.Helper()
	ids := make([]string, 0)
	for _, action := range postCardActions(t, card) {
		id, _ := action["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// dmService собирает сервис с перехватом постов Mattermost.
func dmService(t *testing.T, captured *[]capturedPost) (*MattermostService, *MockMattermostRepo, *MockUserService, *MockUserRealmsService, *MockTicketsService) {
	t.Helper()
	return dmFixtureService(t, &mmCapture{posts: captured}, nil)
}

// dmFixtureService собирает сервис с перехватом постов и диалогов Mattermost.
// comments можно передать nil, если тест не проверяет комментарии.
func dmFixtureService(t *testing.T, capture *mmCapture, comments *MockCommentsService) (*MattermostService, *MockMattermostRepo, *MockUserService, *MockUserRealmsService, *MockTicketsService) {
	t.Helper()
	repo, users, userRealms, tickets := pluginMocks()
	srv := mmServer(t, capture)

	svc := NewMattermostService(&MattermostDeps{
		Repo:       repo,
		Users:      users,
		UserRealms: userRealms,
		Tickets:    tickets,
		Comments:   comments,
		Most:       mattermost.NewMost(mattermost.MostConfig{ServerURL: srv.URL, BaseURL: testBaseURL}),
		BaseURL:    testBaseURL,
	})
	return svc, repo, users, userRealms, tickets
}

func TestMenuButtons(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})

	buttons := svc.menuButtons(uuid.New().String())
	require.Len(t, buttons, 2, "меню бота — ровно две кнопки")

	assert.Equal(t, "Создать заявку", buttons[0].Text)
	assert.Equal(t, "primary", buttons[0].Style)
	assert.Equal(t, testBaseURL+"/api/v1/mattermost/dialog/open", buttons[0].URL)
	assert.NotContains(t, buttons[0].Context, "action", "кнопка создания открывает диалог, а не action")

	assert.Equal(t, "Мои заявки", buttons[1].Text)
	assert.Equal(t, testBaseURL+"/api/v1/mattermost/action", buttons[1].URL)
	assert.Equal(t, "my_tickets", buttons[1].Context["action"])
	assert.NotEmpty(t, buttons[1].Context["realm_id"])
}

func TestMenuButtons_WithoutBaseURL(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{})
	assert.Empty(t, svc.menuButtons(uuid.New().String()), "без base_url кнопки построить нельзя")
}

func TestSendHelpMessage_AttachesMenuButtons(t *testing.T) {
	var posts []capturedPost
	svc, _, _, _, _ := dmService(t, &posts)

	require.NoError(t, svc.sendHelpMessage("bot-token", "ch1", uuid.New().String(), false))
	require.Len(t, posts, 1)

	post := posts[0]
	assert.Equal(t, "ch1", post.channelID)
	assert.Contains(t, post.message, "Можно не печатать команды")

	actions := postActionRow(t, post.props)
	require.Len(t, actions, 2)
	assert.Equal(t, "Создать заявку", actions[0]["name"])
	assert.Equal(t, "button", actions[0]["type"])
	assert.Equal(t, "Мои заявки", actions[1]["name"])
}

func TestSendMenu_ButtonsForTicketCommand(t *testing.T) {
	var posts []capturedPost
	svc, _, _, _, _ := dmService(t, &posts)

	require.NoError(t, svc.sendMenu("bot-token", "ch1", uuid.New().String(),
		"Для оформления заявки нажмите на кнопку ниже"))
	require.Len(t, posts, 1)

	assert.Contains(t, posts[0].message, "Для оформления заявки")
	actions := postActionRow(t, posts[0].props)
	require.Len(t, actions, 2, "у приглашения оформить заявку то же меню из двух кнопок")
	assert.Equal(t, "Создать заявку", actions[0]["name"])
	assert.Equal(t, "Мои заявки", actions[1]["name"])
}

func TestSendStatusMessage_PostsTicketCards(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()
	ticketID := uuid.New()
	num := 7

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)

	created := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	tickets.On("Get", mock.Anything, mock.MatchedBy(func(f *models.TicketFilter) bool {
		if f.Mode == nil || *f.Mode != "created_or_owned" {
			return false
		}
		if f.Actor == nil || f.Actor.ID != userID {
			return false
		}
		return len(f.Statuses) == 5
	})).Return([]*models.Ticket{{
		ID:           ticketID,
		Title:        "Сломалась дверь",
		Status:       models.StatusOpen,
		CreatedAt:    created,
		TicketNumber: &num,
		Site:         &models.SiteShort{Name: "Офис на Пушкина"},
		Owner:        &models.UserShort{ID: userID},
	}}, 1, nil)

	ch := &mmChannel{
		Settings:  &models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"},
		MmUserID:  "mm1",
		ChannelID: "ch1",
	}
	require.NoError(t, svc.sendStatusMessage(context.Background(), ch))
	require.Len(t, posts, 1)

	post := posts[0]
	assert.Equal(t, "ch1", post.channelID)
	assert.Equal(t, "**Ваши активные заявки** (1):", post.message)

	cards := postCards(t, post.props)
	require.Len(t, cards, 1)

	card := cards[0]
	assert.Equal(t, "№7 — Сломалась дверь", card["title"])
	assert.Equal(t, mmStatusColors[models.StatusOpen], card["color"])
	assert.Equal(t, testBaseURL+"/tasks/"+ticketID.String(), card["title_link"], "заголовок ведёт на заявку")

	fields := postObjects(t, card["fields"])
	require.Len(t, fields, 3)
	assert.Equal(t, "Статус", fields[0]["title"])
	assert.Equal(t, "Новая", fields[0]["value"])
	assert.Equal(t, "Создана", fields[1]["title"])
	assert.Equal(t, "05.09.2026", fields[1]["value"])
	assert.Equal(t, "Площадка", fields[2]["title"])
	assert.Equal(t, "Офис на Пушкина", fields[2]["value"])

	// Отдельной кнопки «Открыть заявку» больше нет — вместо неё действие владельца.
	assert.Equal(t, []string{"Отменить заявку:ticket_cancel"}, cardActionsByName(t, card))
	assert.Equal(t, []string{ticketActionID(ticketID, actionCancel)}, cardActionIDs(t, card))
}

func TestSendStatusMessage_WithoutTickets(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{}, 0, nil)

	ch := &mmChannel{
		Settings:  &models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"},
		MmUserID:  "mm1",
		ChannelID: "ch1",
	}
	require.NoError(t, svc.sendStatusMessage(context.Background(), ch))
	require.Len(t, posts, 1)

	assert.Equal(t, "У вас нет активных заявок.", posts[0].message)
	assert.Nil(t, posts[0].props, "пустой список не должен слать пустые карточки")
}

func TestSendStatusMessage_TicketsQueryFails(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("Get", mock.Anything, mock.Anything).Return(nil, 0, models.ErrNotFound)

	ch := &mmChannel{
		Settings:  &models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"},
		MmUserID:  "mm1",
		ChannelID: "ch1",
	}
	assert.Error(t, svc.sendStatusMessage(context.Background(), ch))
	assert.Empty(t, posts)
}

// Нажатие «Мои заявки» отвечает отдельным сообщением с карточками, а не
// {"update": post} — иначе пост с кнопками заменился бы списком.
func TestHandleInteractiveAction_MyTickets(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()
	ticketID := uuid.New()

	var posts []capturedPost
	svc, repo, users, userRealms, tickets := dmService(t, &posts)
	repo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"}, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{{
		ID: ticketID, Title: "Заявка", Status: models.StatusInProgress, CreatedAt: time.Now(),
	}}, 1, nil)

	result, err := svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID:    "mm1",
		ChannelID: "ch1",
		Context:   map[string]string{"action": "my_tickets", "realm_id": realmID.String()},
	})

	require.NoError(t, err)
	assert.Nil(t, result, "ответ отдельным постом — пост с кнопками должен остаться")
	require.Len(t, posts, 1)
	assert.Equal(t, "ch1", posts[0].channelID)
	cards := postCards(t, posts[0].props)
	require.Len(t, cards, 1)
	assert.Equal(t, "Заявка", cards[0]["title"])
	assert.Equal(t, testBaseURL+"/tasks/"+ticketID.String(), cards[0]["title_link"])
}

func TestHandleInteractiveAction_MyTickets_InvalidRealm(t *testing.T) {
	var posts []capturedPost
	svc, _, _, _, _ := dmService(t, &posts)

	_, err := svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID:    "mm1",
		ChannelID: "ch1",
		Context:   map[string]string{"action": "my_tickets", "realm_id": "not-a-uuid"},
	})
	assert.Error(t, err)
	assert.Empty(t, posts)
}

// statusActionFixture — сервис с realm/пользователем и заявкой в заданном
// статусе, готовый к нажатию кнопки карточки.
type statusActionFixture struct {
	svc      *MattermostService
	repo     *MockMattermostRepo
	users    *MockUserService
	realms   *MockUserRealmsService
	tickets  *MockTicketsService
	comments *MockCommentsService
	realmID  uuid.UUID
	userID   uuid.UUID
	ticketID uuid.UUID
}

func newStatusActionFixture(t *testing.T, status models.TicketStatus) (*statusActionFixture, *mmCapture) {
	t.Helper()
	capture := &mmCapture{posts: &[]capturedPost{}}
	comments := new(MockCommentsService)
	svc, repo, users, userRealms, tickets := dmFixtureService(t, capture, comments)

	fixture := &statusActionFixture{
		svc: svc, repo: repo, users: users, realms: userRealms, tickets: tickets,
		comments: comments, realmID: uuid.New(), userID: uuid.New(), ticketID: uuid.New(),
	}
	repo.On("GetByRealm", mock.Anything, fixture.realmID).Return(
		&models.RealmMattermost{RealmID: fixture.realmID, BotToken: "bot-token"}, nil)
	expectExistingUser(users, userRealms, fixture.userID, fixture.realmID)
	tickets.On("GetRealmIDByTicketID", mock.Anything, fixture.ticketID).Return(fixture.realmID, nil)
	return fixture, capture
}

// expectTicketSummary готовит «сырую» заявку — она нужна диалогам над заявкой
// (подтверждение отмены, подтверждение решения, возврат в работу) для номера и
// заголовка в тексте диалога.
func (f *statusActionFixture) expectTicketSummary(t *testing.T, status models.TicketStatus) {
	t.Helper()
	f.tickets.On("GetSummary", mock.Anything, f.ticketID).Return(&models.Ticket{
		ID: f.ticketID, Status: status,
	}, nil)
}

// pressAction нажимает кнопку карточки заявки в первом срезе списка.
func (f *statusActionFixture) pressAction(t *testing.T, action string, status models.TicketStatus) (*ActionResult, error) {
	t.Helper()
	return f.pressActionAt(t, 0, action, status)
}

// pressActionAt нажимает кнопку карточки заявки в срезе, который начинается
// с позиции from.
func (f *statusActionFixture) pressActionAt(t *testing.T, from int, action string, status models.TicketStatus) (*ActionResult, error) {
	t.Helper()
	return f.svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID:    "mm1",
		ChannelID: "ch1",
		PostID:    "p1",
		TriggerID: "trigger1",
		Context: map[string]string{
			"action":    action,
			"ticket_id": f.ticketID.String(),
			"status":    string(status),
			"from":      strconv.Itoa(from),
		},
	})
}

// expectStatusUpdate проверяет, что сервис просит сервис тикетов сменить статус.
func (f *statusActionFixture) expectStatusUpdate(t *testing.T, status models.TicketStatus) {
	t.Helper()
	f.tickets.On("Update", mock.Anything, mock.MatchedBy(func(dto *models.TicketDTO) bool {
		return dto.ID != nil && *dto.ID == f.ticketID &&
			dto.Status == status &&
			dto.Actor != nil && dto.Actor.ID == f.userID &&
			dto.RealmID != nil && *dto.RealmID == f.realmID &&
			dto.Provided["status"]
	})).Return(nil).Once()
}

// Нажатие «Отменить заявку» ничего не меняет: бот только спрашивает
// подтверждение — cancelled терминальный и без диалога промах по кнопке
// отменял бы заявку навсегда.
func TestHandleInteractiveAction_CancelOpensConfirmDialog(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	num := 42
	fixture.tickets.On("GetSummary", mock.Anything, fixture.ticketID).Return(&models.Ticket{
		ID: fixture.ticketID, Status: models.StatusOpen, TicketNumber: &num, Title: "Не открывается 1С",
	}, nil)

	result, err := fixture.pressAction(t, actionCancel, models.StatusCancelled)
	require.NoError(t, err)
	assert.Nil(t, result, "ответ на нажатие не нужен: открыт диалог")
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	assert.Empty(t, capture.patches)
	assert.Empty(t, capture.deletedPostIDs)

	require.Len(t, capture.dialogs, 1)
	sent := capture.dialogs[0]
	dialog := sent.Dialog
	assert.Equal(t, cancelCallbackPrefix+fixture.ticketID.String(), dialog.CallbackId)
	assert.Equal(t, "trigger1", sent.TriggerId)
	assert.Equal(t, "Отменить заявку", dialog.Title)
	assert.Equal(t, "Отменить заявку", dialog.SubmitLabel)
	assert.Empty(t, dialog.Elements, "в диалоге подтверждения нет полей")
	assert.Equal(t, "Заявка №42. Не открывается 1С. Отменить заявку? Действие необратимо.", dialog.IntroductionText)
	// post_id и смещение нужны, чтобы после подтверждения убрать карточку:
	// в SubmitDialogRequest их нет, а ответ диалога посты не обновляет.
	assert.JSONEq(t, `{"from":0,"post_id":"p1"}`, dialog.State)
}

// Подтверждение решения — тоже необратимый переход (closed терминальный),
// поэтому у него свой диалог со своим текстом и submit-подписью.
func TestHandleInteractiveAction_ConfirmOpensConfirmDialog(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)
	fixture.tickets.On("GetSummary", mock.Anything, fixture.ticketID).Return(&models.Ticket{
		ID: fixture.ticketID, Status: models.StatusResolved, Title: "Ошибка печати",
	}, nil)

	result, err := fixture.pressActionAt(t, 10, actionConfirm, models.StatusClosed)
	require.NoError(t, err)
	assert.Nil(t, result)
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	require.Len(t, capture.dialogs, 1)
	dialog := capture.dialogs[0].Dialog
	assert.Equal(t, confirmCallbackPrefix+fixture.ticketID.String(), dialog.CallbackId)
	assert.Equal(t, "Подтвердить решение", dialog.Title)
	assert.Equal(t, "Закрыть заявку", dialog.SubmitLabel)
	assert.Empty(t, dialog.Elements)
	assert.Equal(t, "Заявка «Ошибка печати». Подтвердить решение и закрыть заявку? Действие необратимо.", dialog.IntroductionText)
	assert.JSONEq(t, `{"from":10,"post_id":"p1"}`, dialog.State)
}

// Без trigger_id Mattermost не сможет открыть диалог — сообщаем об этом
// нажавшему, а не падаем с 500.
func TestHandleInteractiveAction_CancelWithoutTriggerID(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)

	result, err := fixture.svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID: "mm1",
		Context: map[string]string{
			"action":    actionCancel,
			"ticket_id": fixture.ticketID.String(),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Ephemeral, "trigger_id")
	assert.Empty(t, capture.dialogs)
}

func TestHandleInteractiveAction_RejectsInvalidTicketID(t *testing.T) {
	fixture, _ := newStatusActionFixture(t, models.StatusOpen)

	_, err := fixture.svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID:  "mm1",
		Context: map[string]string{"action": actionCancel, "ticket_id": "not-a-uuid"},
	})
	assert.Error(t, err)
}

// submitCancelDialog отправляет подтверждение диалога отмены/закрытия.
func (f *statusActionFixture) submitCancelDialog(t *testing.T, action, state string) error {
	t.Helper()
	return f.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		CallbackId: action + ":" + f.ticketID.String(),
		UserId:     "mm1",
		ChannelId:  "ch1",
		State:      state,
	})
}

// Отмена убирает карточку из сообщения: остальные заявки остаются на своих
// местах, а сосед из следующего сообщения в список не подтягивается.
func TestHandleDialogSubmission_Cancel(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)
	// После отмены заявки в списке осталось 12 — из 13 был срез на 5 карточек.
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return(
		ticketFixtures(0, 12, fixture.userID, models.StatusOpen), 12, nil)

	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, `{"from":0,"post_id":"p1"}`))
	fixture.tickets.AssertExpectations(t)
	assert.Empty(t, capture.deletedPostIDs)

	patched, ok := capture.patches["p1"]
	require.True(t, ok, "карточка должна убраться из своего сообщения")
	assert.Equal(t, "**Ваши активные заявки** (13):", patched.message,
		"заголовок считается по состоянию до действия и не должен «поехать»")
	cards := postCards(t, patched.props)
	require.Len(t, cards, 4, "из среза на 5 карточек уходит ровно одна")
	assert.Equal(t, "№1 — Заявка 1", cards[0]["title"])
	assert.Equal(t, "№4 — Заявка 4", cards[3]["title"])

	// Ответ диалога не умеет писать нажавшему, поэтому итог — обычный пост.
	require.Len(t, *capture.posts, 1)
	assert.Equal(t, "Заявка отменена.", (*capture.posts)[0].message)
	assert.Equal(t, "ch1", (*capture.posts)[0].channelID)
}

// Целевой статус берём из действия, а не из контекста кнопки: подделанное
// значение в контекте не должно решать, что произойдёт с заявкой.
func TestHandleDialogSubmission_CancelIgnoresContextStatus(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectTicketSummary(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return(
		ticketFixtures(0, 1, fixture.userID, models.StatusOpen), 1, nil)

	_, err := fixture.pressAction(t, actionCancel, models.StatusClosed)
	require.NoError(t, err)
	require.Len(t, capture.dialogs, 1)
	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, `{"from":0,"post_id":"p1"}`))
	fixture.tickets.AssertExpectations(t)
}

// Если карточка была единственной в сообщении, сообщение удаляется целиком —
// иначе в диалоге осталось бы пустое «**Ваши активные заявки** (1):».
func TestHandleDialogSubmission_CancelDeletesPostWhenLastCardGone(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)
	// Заявка была единственной во всём списке, после отмены не осталось ничего.
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{}, 0, nil)

	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, `{"from":0,"post_id":"p1"}`))
	assert.Empty(t, capture.patches)
	assert.Equal(t, []string{"p1"}, capture.deletedPostIDs)
	assert.Equal(t, "Заявка отменена.", (*capture.posts)[0].message)
}

// Исчезнувший последний частичный срез: сообщение удаляется, список не
// прыгает к началу — остальные сообщения остаются как были.
func TestHandleDialogSubmission_CancelDeletesEmptyLastPartialPost(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)
	// 15 заявок осталось, последний срез начинался с 16-й — он исчез целиком.
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return(
		ticketFixtures(0, 15, fixture.userID, models.StatusOpen), 15, nil)

	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, `{"from":15,"post_id":"p1"}`))
	assert.Empty(t, capture.patches)
	assert.Equal(t, []string{"p1"}, capture.deletedPostIDs)
}

// Битое смещение в State не должно приводить к удалению сообщения:
// перерисовываем начало списка.
func TestHandleDialogSubmission_CancelKeepsPostOnBrokenOffset(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return(
		ticketFixtures(0, 12, fixture.userID, models.StatusOpen), 12, nil)

	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, `{"from":99,"post_id":"p1"}`))
	assert.Empty(t, capture.deletedPostIDs)
	patched, ok := capture.patches["p1"]
	require.True(t, ok)
	assert.Len(t, postCards(t, patched.props), 5,
		"при неизвестном смещении показываем целый первый срез")
}

// Подтверждение решения тоже уводит заявку из списка активных: из среза
// 11–15 из 16 остаётся четыре карточки.
func TestHandleDialogSubmission_Confirm(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)
	fixture.expectStatusUpdate(t, models.StatusClosed)
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return(
		ticketFixtures(0, 15, fixture.userID, models.StatusResolved), 15, nil)

	require.NoError(t, fixture.submitCancelDialog(t, actionConfirm, `{"from":10,"post_id":"p1"}`))
	fixture.tickets.AssertExpectations(t)

	patched, ok := capture.patches["p1"]
	require.True(t, ok)
	assert.Equal(t, "Заявки 11–15 из 16:", patched.message)
	assert.Len(t, postCards(t, patched.props), 4)
	require.Len(t, *capture.posts, 1)
	assert.Equal(t, "Заявка закрыта.", (*capture.posts)[0].message)
}

// Пока диалог открыт, заявку могли закрыть или снять права: доменная ошибка
// возвращается в диалог, список не трогаем.
func TestHandleDialogSubmission_CancelRejected(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.tickets.On("Update", mock.Anything, mock.Anything).Return(models.ErrTicketFrozen)

	err := fixture.submitCancelDialog(t, actionCancel, `{"from":0,"post_id":"p1"}`)
	assert.ErrorIs(t, err, models.ErrTicketFrozen)
	assert.Empty(t, capture.patches)
	assert.Empty(t, capture.deletedPostIDs)
	assert.Empty(t, *capture.posts)
}

// Закрытый диалог подтверждения ничего не меняет.
func TestHandleDialogSubmission_CancelCancelled(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)

	require.NoError(t, fixture.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		Cancelled:  true,
		CallbackId: cancelCallbackPrefix + fixture.ticketID.String(),
		UserId:     "mm1",
	}))
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	assert.Empty(t, *capture.posts)
	assert.Empty(t, capture.patches)
}

// Без post_id в State перерисовать список нечем: статус меняем, сообщение не
// трогаем — оно обновится при следующем «Мои заявки».
func TestHandleDialogSubmission_CancelWithoutListPost(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusOpen)
	fixture.expectStatusUpdate(t, models.StatusCancelled)

	require.NoError(t, fixture.submitCancelDialog(t, actionCancel, ""))
	assert.Empty(t, capture.patches)
	fixture.tickets.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestHandleInteractiveAction_ReopenOpensReasonDialog(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)
	num := 42
	fixture.tickets.On("GetSummary", mock.Anything, fixture.ticketID).Return(&models.Ticket{
		ID: fixture.ticketID, Status: models.StatusResolved, TicketNumber: &num,
	}, nil)
	result, err := fixture.pressAction(t, actionReopen, models.StatusInProgress)
	require.NoError(t, err)
	assert.Nil(t, result, "диалог открывается, пост не перерисовываем")
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	require.Len(t, capture.dialogs, 1)
	dialog := capture.dialogs[0].Dialog
	assert.Equal(t, reopenCallbackPrefix+fixture.ticketID.String(), dialog.CallbackId)
	assert.Equal(t, "trigger1", capture.dialogs[0].TriggerId)
	assert.Contains(t, dialog.IntroductionText, "№42")
	// post_id нужен, чтобы после сабмита обновить карточку в списке: в
	// SubmitDialogRequest его нет.
	assert.JSONEq(t, `{"from":0,"post_id":"p1"}`, dialog.State)
	require.Len(t, dialog.Elements, 1)
	assert.Equal(t, "reason", dialog.Elements[0].Name)
	assert.Equal(t, "textarea", dialog.Elements[0].Type)
}

// Без trigger_id Mattermost не сможет открыть диалог — сообщаем об этом
// нажавшему, а не падаем с 500.
func TestHandleInteractiveAction_ReopenWithoutTriggerID(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)

	result, err := fixture.svc.HandleInteractiveAction(context.Background(), &models.InteractiveActionDTO{
		UserID: "mm1",
		Context: map[string]string{
			"action":    actionReopen,
			"ticket_id": fixture.ticketID.String(),
			"status":    string(models.StatusInProgress),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Ephemeral, "trigger_id")
	assert.Empty(t, capture.dialogs)
}

// Диалог причины меняет статус в работу и сохраняет причину комментарием —
// статус сначала (на «решённой» заявке внешний комментарий запрещён).
// После возврата в работу карточка в списке обновляется на месте: у неё новый
// статус и больше нет кнопок «решённой» заявки.
func TestHandleDialogSubmission_Reopen(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)
	fixture.expectStatusUpdate(t, models.StatusInProgress)
	fixture.comments.On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(dto *models.CreateCommentDTO) bool {
		return dto.TicketID == fixture.ticketID &&
			dto.UserID == fixture.userID &&
			!dto.IsInternal &&
			dto.Realm == fixture.realmID.String() &&
			strings.Contains(dto.Text, "не устранена")
	})).Return(&models.Comment{}, nil)
	fixture.tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{
		{ID: fixture.ticketID, Title: "Заявка", Status: models.StatusInProgress, CreatedAt: time.Now()},
	}, 1, nil)

	err := fixture.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		CallbackId: reopenCallbackPrefix + fixture.ticketID.String(),
		UserId:     "mm1",
		ChannelId:  "ch1",
		State:      `{"from":0,"post_id":"p1"}`,
		Submission: map[string]any{"reason": "  не устранена  "},
	})
	require.NoError(t, err)
	fixture.tickets.AssertExpectations(t)
	fixture.comments.AssertExpectations(t)
	require.Len(t, *capture.posts, 1)
	assert.Contains(t, (*capture.posts)[0].message, "возвращена в работу")

	patched, ok := capture.patches["p1"]
	require.True(t, ok, "карточка в списке должна обновиться на месте")
	assert.Equal(t, "**Ваши активные заявки** (1):", patched.message)
	cards := postCards(t, patched.props)
	require.Len(t, cards, 1)
	fields := postObjects(t, cards[0]["fields"])
	assert.Equal(t, "В работе", fields[0]["value"])
	assert.Empty(t, cardActionsByName(t, cards[0]), "у заявки в работе действий нет")
}

// Без post_id в State перерисовать список нечем: статус меняем, сообщение не
// трогаем — оно обновится при следующем «Мои заявки».
func TestHandleDialogSubmission_ReopenWithoutListPost(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)
	fixture.expectStatusUpdate(t, models.StatusInProgress)
	fixture.comments.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(&models.Comment{}, nil)

	err := fixture.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		CallbackId: reopenCallbackPrefix + fixture.ticketID.String(),
		UserId:     "mm1",
		ChannelId:  "ch1",
		Submission: map[string]any{"reason": "не устранена"},
	})
	require.NoError(t, err)
	assert.Empty(t, capture.patches)
	fixture.tickets.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestHandleDialogSubmission_ReopenWithoutReason(t *testing.T) {
	fixture, _ := newStatusActionFixture(t, models.StatusResolved)

	err := fixture.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		CallbackId: reopenCallbackPrefix + fixture.ticketID.String(),
		UserId:     "mm1",
		Submission: map[string]any{"reason": "   "},
	})
	require.Error(t, err)
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandleDialogSubmission_ReopenCancelled(t *testing.T) {
	fixture, capture := newStatusActionFixture(t, models.StatusResolved)

	require.NoError(t, fixture.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		Cancelled:  true,
		CallbackId: reopenCallbackPrefix + fixture.ticketID.String(),
		UserId:     "mm1",
	}))
	fixture.tickets.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	assert.Empty(t, *capture.posts)
	assert.Empty(t, capture.patches)
}

// ticketFixture — заявка для проверки разбиения списка на сообщения.
func ticketFixture(i int, status models.TicketStatus, ownerID uuid.UUID) *models.Ticket {
	num := i + 1
	return &models.Ticket{
		ID:           uuid.New(),
		Title:        fmt.Sprintf("Заявка %d", num),
		Status:       status,
		CreatedAt:    time.Now(),
		TicketNumber: &num,
		Owner:        &models.UserShort{ID: ownerID},
	}
}

// ticketFixtures — count заявок, пронумерованных подряд, для проверки того,
// какие именно карточки остаются в сообщении после действия.
func ticketFixtures(start, count int, ownerID uuid.UUID, status models.TicketStatus) []*models.Ticket {
	tickets := make([]*models.Ticket, 0, count)
	for i := start; i < start+count; i++ {
		tickets = append(tickets, ticketFixture(i, status, ownerID))
	}
	return tickets
}

// Список показывается целиком, просто несколькими сообщениями: 27 заявок →
// 6 сообщений по 5/5/5/5/5/2, а не «первые 20 из 27».
func TestMyTicketsChunks_SplitsAllTickets(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})
	ownerID := uuid.New()

	tickets := make([]*models.Ticket, 0, 27)
	for i := 0; i < 27; i++ {
		tickets = append(tickets, ticketFixture(i, models.StatusOpen, ownerID))
	}

	chunks := svc.myTicketsChunks(tickets, ownerID)
	require.Len(t, chunks, 6)
	for i, chunk := range chunks {
		want := myTicketsChunkSize
		if i == len(chunks)-1 {
			want = 2
		}
		assert.Len(t, chunk.cards, want)
	}

	assert.Equal(t, "**Ваши активные заявки** (27):", chunks[0].message)
	assert.Equal(t, "Заявки 6–10 из 27:", chunks[1].message)
	assert.Equal(t, "Заявки 11–15 из 27:", chunks[2].message)
	assert.Equal(t, "Заявки 16–20 из 27:", chunks[3].message)
	assert.Equal(t, "Заявки 21–25 из 27:", chunks[4].message)
	assert.Equal(t, "Заявки 26–27 из 27:", chunks[5].message)

	// Каждая заявка показана ровно один раз и в исходном порядке.
	titles := make([]string, 0, len(tickets))
	for _, chunk := range chunks {
		for _, card := range chunk.cards {
			titles = append(titles, card.Title)
		}
	}
	require.Len(t, titles, 27, "все заявки должны попасть в сообщения")
	for i, title := range titles {
		assert.Equal(t, fmt.Sprintf("№%d — Заявка %d", i+1, i+1), title)
	}
}

// В карточке заявки показывается описание: в ленте видно, о чём заявка, не
// открывая её. Переносы строк схлопываются, длинный текст подрезается.
func TestMyTicketCard_Description(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})
	ownerID := uuid.New()

	ticket := ticketFixture(0, models.StatusOpen, ownerID)
	ticket.Description = "  Не работает\n\n  экспорт в 1С  "
	card := svc.myTicketCard(ticket, ownerID, 0)
	assert.Equal(t, "Не работает экспорт в 1С", card.Text)

	ticket.Description = strings.Repeat("а", cardSummaryLimit+50)
	card = svc.myTicketCard(ticket, ownerID, 0)
	assert.Len(t, []rune(card.Text), cardSummaryLimit+1, "длинное описание подрезается многоточием")
	assert.True(t, strings.HasSuffix(card.Text, "…"))
}

func TestCardSummary_EdgeCases(t *testing.T) {
	assert.Empty(t, cardSummary(""), "без описания карточка не меняется")
	assert.Empty(t, cardSummary("  \n\t "))
	assert.Equal(t, cardSummaryLimit, len([]rune(cardSummary(strings.Repeat("я", cardSummaryLimit)))))
}

func TestMyTicketsChunks_EmptyList(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})

	chunks := svc.myTicketsChunks(nil, uuid.New())
	require.Len(t, chunks, 1)
	assert.Equal(t, "У вас нет активных заявок.", chunks[0].message)
	assert.Empty(t, chunks[0].cards)
}

// Кнопки зависят от статуса и от того, владелец ли пользователь заявкой.
func TestMyTicketActionButtons_ByStatusAndOwner(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})
	ownerID, otherID := uuid.New(), uuid.New()

	tests := []struct {
		name   string
		status models.TicketStatus
		owner  uuid.UUID
		want   []string
	}{
		{"open у владельца можно отменить", models.StatusOpen, ownerID, []string{"Отменить заявку:ticket_cancel"}},
		{"решённую можно подтвердить и вернуть", models.StatusResolved, ownerID, []string{"Подтвердить решение:ticket_confirm", "Вернуть в работу:ticket_reopen"}},
		{"в работе действий нет", models.StatusInProgress, ownerID, nil},
		{"ожидание без действий", models.StatusPending, ownerID, nil},
		{"отложенная без действий", models.StatusOnHold, ownerID, nil},
		{"чужое открытое нельзя отменить", models.StatusOpen, otherID, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket := ticketFixture(0, tt.status, ownerID)
			if tt.owner == otherID {
				ticket.Owner = &models.UserShort{ID: otherID}
			}

			buttons := svc.myTicketActionButtons(ticket, ownerID, 0)
			var names []string
			for _, b := range buttons {
				names = append(names, b.Text+":"+b.Context["action"])
				assert.Equal(t, testBaseURL+"/api/v1/mattermost/action", b.URL)
				assert.Equal(t, ticket.ID.String(), b.Context["ticket_id"])
				assert.Equal(t, "0", b.Context["from"], "смещение нужно для перерисовки того же среза")
				assert.Equal(t, ticketActionID(ticket.ID, b.Context["action"]), b.ID)
				assert.Regexp(t, `^[A-Za-z0-9]+$`, b.ID, "id попадает в роут /posts/{id}/actions/{action_id}")
			}
			assert.Equal(t, tt.want, names)
		})
	}
}

// action_id действий должен быть уникален в пределах поста: Mattermost ищет
// действие по id в attachments поста и берёт первое совпадение, поэтому общий id
// у карточек увёл бы нажатие на чужую заявку.
func TestMyTicketsChunkAt_UniqueButtonIDs(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})
	ownerID := uuid.New()

	tickets := make([]*models.Ticket, 0, 4)
	for i := 0; i < 4; i++ {
		tickets = append(tickets, ticketFixture(i, models.StatusResolved, ownerID))
	}

	chunk, ok := svc.myTicketsChunkAt(tickets, 0, ownerID)
	require.True(t, ok)
	require.Len(t, chunk.cards, 4)

	seen := map[string]bool{}
	for i, card := range chunk.cards {
		require.Len(t, card.Buttons, 2, "у решённой заявки два действия")
		for _, b := range card.Buttons {
			require.NotEmpty(t, b.ID, "карточка %d: действие без id", i)
			assert.Falsef(t, seen[b.ID], "id %q повторяется в посте", b.ID)
			seen[b.ID] = true
		}
	}
	assert.Len(t, seen, 8)
}

// Смещение кнопки должно совпадать с позицией заявки в её сообщении, иначе
// после действия перерисуется чужой срез.
func TestMyTicketsChunkAt_FromContext(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})
	ownerID := uuid.New()

	tickets := make([]*models.Ticket, 0, 12)
	for i := 0; i < 12; i++ {
		tickets = append(tickets, ticketFixture(i, models.StatusOpen, ownerID))
	}

	chunk, ok := svc.myTicketsChunkAt(tickets, 10, ownerID)
	require.True(t, ok)
	require.Len(t, chunk.cards, 2)
	assert.Equal(t, "Заявки 11–12 из 12:", chunk.message)
	require.Len(t, chunk.cards[0].Buttons, 1)
	assert.Equal(t, "10", chunk.cards[0].Buttons[0].Context["from"])

	_, ok = svc.myTicketsChunkAt(tickets, 12, ownerID)
	assert.False(t, ok, "срез за концом списка не собирается")
	_, ok = svc.myTicketsChunkAt(tickets, -1, ownerID)
	assert.False(t, ok, "отрицательное смещение не собирается")
}

// 27 заявок уходят несколькими сообщениями, а не одним обрезанным.
func TestSendStatusMessage_SendsAllTicketsInSeveralPosts(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)

	list := make([]*models.Ticket, 0, 27)
	for i := 0; i < 27; i++ {
		list = append(list, ticketFixture(i, models.StatusInProgress, userID))
	}
	tickets.On("Get", mock.Anything, mock.Anything).Return(list, len(list), nil)

	ch := &mmChannel{
		Settings:  &models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"},
		MmUserID:  "mm1",
		ChannelID: "ch1",
	}
	require.NoError(t, svc.sendStatusMessage(context.Background(), ch))
	require.Len(t, posts, 6, "27 заявок — это шесть сообщений по 5/5/5/5/5/2")
	for i, post := range posts {
		want := myTicketsChunkSize
		if i == len(posts)-1 {
			want = 2
		}
		assert.Len(t, postCards(t, post.props), want)
		assert.Equal(t, "ch1", post.channelID)
	}
}

func TestMyTicketsChunks_WithoutBaseURL(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{})

	chunks := svc.myTicketsChunks([]*models.Ticket{{
		ID: uuid.New(), Title: "Заявка", Status: models.StatusOpen, CreatedAt: time.Now(),
	}}, uuid.New())

	require.Len(t, chunks, 1)
	require.Len(t, chunks[0].cards, 1)
	assert.Empty(t, chunks[0].cards[0].TitleLink, "без base_url ссылка на заявку не строится")
}

func TestMyTicketsChunks_TitleWithoutNumber(t *testing.T) {
	svc := NewMattermostService(&MattermostDeps{BaseURL: testBaseURL})

	chunks := svc.myTicketsChunks([]*models.Ticket{{
		ID: uuid.New(), Title: "Без номера", Status: models.StatusPending, CreatedAt: time.Now(),
	}}, uuid.New())

	require.Len(t, chunks, 1)
	require.Len(t, chunks[0].cards, 1)
	assert.Equal(t, "Без номера", chunks[0].cards[0].Title, "без номера заголовок остаётся как есть")
	assert.Equal(t, mmStatusLabels[models.StatusPending], chunks[0].cards[0].Fields[0].Value)
}

// cardChannel — канал бота с включённой интеграцией.
func cardChannel(realmID uuid.UUID) *mmChannel {
	return &mmChannel{
		Settings:  &models.RealmMattermost{RealmID: realmID, BotToken: "bot-token"},
		MmUserID:  "mm1",
		ChannelID: "ch1",
	}
}

// expectNumberLookup готовит мок поиска заявки по номеру: Tickets.Get без
// групп актора вернул бы заявку любого реалма с таким номером, поэтому дальше
// обязателен GetByID с проверкой прав.
func expectNumberLookup(tickets *MockTicketsService, number int, ticketID uuid.UUID) {
	tickets.On("Get", mock.Anything, mock.MatchedBy(func(f *models.TicketFilter) bool {
		return f.Number != nil && *f.Number == number
	})).Return([]*models.Ticket{{ID: ticketID}}, 1, nil)
}

func TestHandleTicketCard_OwnerResolved_TwoButtons(t *testing.T) {
	realmID, userID, ticketID := uuid.New(), uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	expectNumberLookup(tickets, 42, ticketID)

	due := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	tickets.On("GetByID", mock.Anything, mock.MatchedBy(func(d *models.GetTicketByIdDTO) bool {
		return d.ID == ticketID && d.Actor != nil && d.Actor.ID == userID
	})).Return(&models.Ticket{
		ID:           ticketID,
		Title:        "Не работает касса",
		Description:  "Описание заявки",
		Status:       models.StatusResolved,
		Priority:     models.PriorityHigh,
		CreatedAt:    time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
		DueDate:      &due,
		TicketNumber: ptr(42),
		Category:     &models.CategoryShort{Name: "Касса"},
		Site:         &models.SiteShort{Name: "Офис на Пушкина"},
		Owner:        &models.UserShort{ID: userID, Username: "u1"},
		Assignee:     &models.UserShort{ID: uuid.New(), Username: "executor"},
	}, nil)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№42"))
	require.Len(t, posts, 1)
	assert.Equal(t, "Заявка №42", posts[0].message)

	cards := postCards(t, posts[0].props)
	require.Len(t, cards, 1)
	card := cards[0]
	assert.Equal(t, testBaseURL+"/tasks/"+ticketID.String(), card["title_link"])
	assert.Equal(t, mmStatusColors[models.StatusResolved], card["color"])
	assert.Equal(t, "Описание заявки", card["text"])

	// Владелец решённой заявки: подтвердить решение и вернуть в работу.
	assert.Equal(t, []string{
		"Подтвердить решение:ticket_confirm",
		"Вернуть в работу:ticket_reopen",
	}, cardActionsByName(t, card))
	assert.Equal(t, []string{
		ticketActionID(ticketID, actionConfirm),
		ticketActionID(ticketID, actionReopen),
	}, cardActionIDs(t, card))

	// post_id в контекст не кладём: карточка одиночная, править список нечего,
	// а applyListEdit с пустым PostID выходит с return nil.
	for _, action := range postCardActions(t, card) {
		ctxMap := actionContext(t, action)
		assert.NotContains(t, ctxMap, "post_id")
		assert.Equal(t, ticketID.String(), ctxMap["ticket_id"])
	}
}

func TestHandleTicketCard_Fields(t *testing.T) {
	realmID, userID, ticketID := uuid.New(), uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	expectNumberLookup(tickets, 5, ticketID)

	tickets.On("GetByID", mock.Anything, mock.Anything).Return(&models.Ticket{
		ID:           ticketID,
		Title:        "Заявка",
		Status:       models.StatusOpen,
		Priority:     models.PriorityUrgent,
		CreatedAt:    time.Date(2026, 9, 5, 12, 30, 0, 0, time.UTC),
		TicketNumber: ptr(5),
		Category:     &models.CategoryShort{Name: "Касса"},
		Site:         &models.SiteShort{Name: "Офис"},
		Owner:        &models.UserShort{ID: userID, Username: "u1"},
	}, nil)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№5"))

	card := postCards(t, posts[0].props)[0]
	titles := map[string]string{}
	for _, f := range postObjects(t, card["fields"]) {
		titles[f["title"].(string)], _ = f["value"].(string)
		assert.True(t, f["short"] == true, "поля карточки короткие")
	}
	assert.Equal(t, "Новая", titles["Статус"])
	assert.Equal(t, "Срочный", titles["Приоритет"])
	assert.Equal(t, "Касса", titles["Категория"])
	assert.Equal(t, "Офис", titles["Площадка"])
	assert.Equal(t, "05.09.2026 12:30", titles["Создана"])
	assert.Equal(t, "u1", titles["Заказчик"])
	assert.NotContains(t, titles, "Срок", "без срока поле не выводится")
	assert.NotContains(t, titles, "Исполнитель", "без исполнителя поле не выводится")
}

func TestHandleTicketCard_NonOwner_NoButtons(t *testing.T) {
	realmID, userID, ticketID := uuid.New(), uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	expectNumberLookup(tickets, 9, ticketID)

	// Заявка открыта, но заказчик — другой пользователь: действий нет.
	tickets.On("GetByID", mock.Anything, mock.Anything).Return(&models.Ticket{
		ID:           ticketID,
		Title:        "Чужая заявка",
		Status:       models.StatusOpen,
		CreatedAt:    time.Now(),
		TicketNumber: ptr(9),
		Owner:        &models.UserShort{ID: uuid.New(), Username: "other"},
	}, nil)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№9"))
	card := postCards(t, posts[0].props)[0]
	assert.Empty(t, cardActionsByName(t, card), "не-владелец не получает действий")
}

// Чужая заявка не должна ни показываться, ни подтверждать своё существование:
// Get по номеру без групп актора находит её, но GetByID режет доступ.
func TestHandleTicketCard_ForeignTicket_Denied(t *testing.T) {
	realmID, userID, ticketID := uuid.New(), uuid.New(), uuid.New()

	capture := &mmCapture{posts: &[]capturedPost{}}
	svc, _, users, userRealms, tickets := dmFixtureService(t, capture, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	expectNumberLookup(tickets, 77, ticketID)
	tickets.On("GetByID", mock.Anything, mock.Anything).Return(nil, models.ErrPermissionDenied)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№77"))

	// В канал карточка не уходит; вместо неё — личное «не найдена»: существование
	// чужой заявки пользователю подтверждать нельзя.
	require.Len(t, *capture.posts, 1)
	dm := (*capture.posts)[0]
	assert.Equal(t, "dm1", dm.channelID, "ответ уходит в личные сообщения, не в канал")
	assert.Equal(t, "Заявка №77 не найдена", dm.message)
	assert.NotContains(t, dm.props, "attachments", "карточка чужой заявки не публикуется")
}

func TestHandleTicketCard_NotFound(t *testing.T) {
	realmID, userID := uuid.New(), uuid.New()

	capture := &mmCapture{posts: &[]capturedPost{}}
	svc, _, users, userRealms, tickets := dmFixtureService(t, capture, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{}, 0, nil)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№404"))
	require.Len(t, *capture.posts, 1)
	assert.Equal(t, "Заявка №404 не найдена", (*capture.posts)[0].message)
	assert.NotContains(t, (*capture.posts)[0].props, "attachments")
}

func TestHandleTicketCard_LongDescriptionTruncated(t *testing.T) {
	realmID, userID, ticketID := uuid.New(), uuid.New(), uuid.New()

	var posts []capturedPost
	svc, _, users, userRealms, tickets := dmService(t, &posts)
	expectExistingUser(users, userRealms, userID, realmID)
	expectNumberLookup(tickets, 3, ticketID)
	tickets.On("GetByID", mock.Anything, mock.Anything).Return(&models.Ticket{
		ID: ticketID, Title: "Заявка", Status: models.StatusOpen, CreatedAt: time.Now(),
		TicketNumber: ptr(3),
		Description:  strings.Repeat("а", cardDetailLimit+500),
		Owner:        &models.UserShort{ID: userID, Username: "u1"},
	}, nil)

	require.NoError(t, svc.handleTicketCard(context.Background(), cardChannel(realmID), "№3"))
	text, _ := postCards(t, posts[0].props)[0]["text"].(string)
	assert.Len(t, []rune(text), cardDetailLimit+1, "описание обрезается с многоточием")
	assert.True(t, strings.HasSuffix(text, "…"))
}

func TestDetailText(t *testing.T) {
	assert.Equal(t, "", detailText("   \n\n  "), "пустое описание не даёт пустого текста")
	assert.Equal(t, "первый абзац\nвторой абзац",
		detailText("первый абзац\n\n\r\nвторой абзац\n"), "абзацы сохраняются, пустые убираются")
}

// Диспетчер: «№N текст» остаётся комментарием, «№N» с файлом — вложением,
// голое «№N» — карточкой.
func TestDispatch_NumberSyntaxes(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"bare number is a card", "№42"},
		{"number without sign is a card", "42"},
		{"number with text is a comment", "№42 почините"},
		{"number with hash is a comment", "#42 почините"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.msg == "№42" || tc.msg == "42", attachCommands.MatchString(tc.msg))
			assert.Equal(t, tc.msg == "№42 почините" || tc.msg == "#42 почините", commentCommands.MatchString(tc.msg))
		})
	}
	// Кейс карточки в диспетчере стоит после комментария и вложения,
	// иначе перехватил бы их: commentCommands проверяется раньше.
	assert.False(t, commentCommands.MatchString("№42"), "голое «№42» не комментарий")
	assert.False(t, attachCommands.MatchString("№42 почините"), "«№N текст» не номер")
}
