package mattermost

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
