package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runPluginGuarded invokes middleware.PluginTokenGuard as a middleware on a minimal route so
// that the pass-through case actually runs c.Next().
func runPluginGuarded(t *testing.T, token, auth string) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	engine := gin.New()
	engine.Use(middleware.PluginTokenGuard(token))
	hit := false
	engine.GET("/", func(c *gin.Context) { hit = true })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec, hit
}

func TestPluginTokenGuard(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		auth      string
		wantCode  int
		wantAbort bool
	}{
		{name: "empty config token disables plugin", token: "", auth: "Bearer secret", wantCode: http.StatusUnauthorized, wantAbort: true},
		{name: "missing Authorization header", token: "secret", auth: "", wantCode: http.StatusUnauthorized, wantAbort: true},
		{name: "non-bearer token", token: "secret", auth: "secret", wantCode: http.StatusUnauthorized, wantAbort: true},
		{name: "wrong token", token: "secret", auth: "Bearer wrong", wantCode: http.StatusUnauthorized, wantAbort: true},
		{name: "correct token passes", token: "secret", auth: "Bearer secret", wantCode: http.StatusOK, wantAbort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, hit := runPluginGuarded(t, tt.token, tt.auth)
			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, !tt.wantAbort, hit)
		})
	}
}

// TestPluginLinkContextRouteWiring проверяет, что ручка deep-link зарегистрирована:
// запрошенный путь отвечает 400 (невалидный ввод), а не 404 (маршрут не зарегистрирован).
// Guards проходятся корректными IP/токеном, но до сервиса дело не доходит —
// userId обязателен, поэтому nil-сервис не трогается.
func TestPluginLinkContextRouteWiring(t *testing.T) {
	engine := gin.New()
	h := &Handler{service: nil}
	h.registerPluginRoutes(engine.Group("/api/v1"), config.MattermostConfig{
		PluginToken:      "secret",
		AllowedServerIPs: []string{"192.0.2.0/24"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/tickets/abc/link-context", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "404")
}

// TestPluginRoutesWiring проверяет, что fail-closed срабатывает раньше хендлера:
// плагин с пустым токеном получит 401, с запрещённым источником — 403.
func TestPluginRoutesWiring(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.MattermostConfig
		wantCode int
	}{
		{
			name:     "untrusted source is rejected (403)",
			cfg:      config.MattermostConfig{PluginToken: "secret"},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "empty plugin token disables plugin (401)",
			cfg:      config.MattermostConfig{AllowedServerIPs: []string{"192.0.2.0/24"}},
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			h := &Handler{service: nil} // guard abortит раньше обращения к сервису
			h.registerPluginRoutes(engine.Group("/api/v1"), tt.cfg)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/plugin/context",
				strings.NewReader(`{"channelId":"c1","userId":"u1"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

// stubCreateMattermost реализует services.Mattermost через встраивание интерфейса:
// переопределён только PluginCreateTicket, остальные методы не вызываются.
type stubCreateMattermost struct {
	services.Mattermost
	result *models.PluginCreateTicketResult
}

func (s stubCreateMattermost) PluginCreateTicket(context.Context, *models.PluginCreateTicketInput) (*models.PluginCreateTicketResult, error) {
	return s.result, nil
}

// TestPluginCreateTicketReturnsDeepLink защищает от регрессии, при которой хендлер
// собирает ответ вручную через gin.H и теряет deepLink: без него webapp-плагин
// не получает site-relative ссылку на заявку внутри Mattermost.
func TestPluginCreateTicketReturnsDeepLink(t *testing.T) {
	ticketID := uuid.MustParse("b80349e7-9344-4217-8548-2860e381f2fd")
	deepLink := models.PluginRoutePrefix + "/ticket/" + ticketID.String()

	engine := gin.New()
	h := &Handler{service: stubCreateMattermost{result: &models.PluginCreateTicketResult{
		ID:       ticketID,
		Number:   31,
		Title:    "t",
		Link:     "http://192.168.4.159:9000/tasks/" + ticketID.String(),
		DeepLink: deepLink,
	}}}
	h.registerPluginRoutes(engine.Group("/api/v1"), config.MattermostConfig{
		PluginToken:      "secret",
		AllowedServerIPs: []string{"192.0.2.0/24"},
	})

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	require.NoError(t, mw.WriteField("channelId", "c1"))
	require.NoError(t, mw.WriteField("userId", "u1"))
	require.NoError(t, mw.WriteField("title", "t"))
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugin/tickets", body)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			Link     string `json:"link"`
			DeepLink string `json:"deepLink"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.Link)
	assert.Equal(t, deepLink, resp.Data.DeepLink)
}

// stubScopeMattermost захватывает scope, дошедший до сервиса, — так проверяется
// проброс botUserId из личного диалога.
type stubScopeMattermost struct {
	services.Mattermost
	got models.PluginScope
}

func (s *stubScopeMattermost) PluginContext(_ context.Context, scope models.PluginScope) (*models.PluginContextResult, error) {
	s.got = scope
	return &models.PluginContextResult{Bound: false}, nil
}

// TestPluginContextPassesBotUserID проверяет, что хендлер /plugin/context
// доносит botUserId собеседника: по нему сервис определяет реалм для личного
// диалога, который не привязан к каналу.
func TestPluginContextPassesBotUserID(t *testing.T) {
	engine := gin.New()
	stub := &stubScopeMattermost{}
	h := &Handler{service: stub}
	h.registerPluginRoutes(engine.Group("/api/v1"), config.MattermostConfig{
		PluginToken:      "secret",
		AllowedServerIPs: []string{"192.0.2.0/24"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugin/context",
		strings.NewReader(`{"channelId":"dm1","userId":"u1","botUserId":"bot1"}`))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, models.PluginScope{ChannelID: "dm1", MmUserID: "u1", BotUserID: "bot1"}, stub.got)
}
