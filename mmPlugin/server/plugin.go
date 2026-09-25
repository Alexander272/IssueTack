package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	// apiPrefix — точка входа плагина: webapp обращается к /plugins/issuetrack/api/...
	apiPrefix = "/api"
	// backendAPIPrefix — префикс маршрутов плагина на бэкенде.
	backendAPIPrefix = "/api/v1/plugin"
	// contextNegCacheTTL — срок жизни негативного ответа /context (канал не
	// привязан к реалму). Кэш общий на весь плагин, поэтому N пользователей
	// одного непривязанного канала = 1 запрос на бэкенд за TTL.
	contextNegCacheTTL = 60 * time.Second
)

// configuration — настройки плагина (задаются в System Console).
type configuration struct {
	BackendURL string `json:"backend_url"`
	APIToken   string `json:"api_token"`
}

// contextNegEntry — закэшированный ответ /context для непривязанного канала.
type contextNegEntry struct {
	ts     time.Time
	status int
	header http.Header
	body   []byte
}

// Plugin — серверная часть MM-плагина: прозрачный прокси в бэкенд IssueTrack
// с сервисным токеном и эфемерным подтверждением создания заявки.
type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	client *http.Client

	contextLock  sync.RWMutex
	contextCache map[string]*contextNegEntry
}

// OnActivate инициализирует HTTP-клиент прокси.
func (p *Plugin) OnActivate() error {
	p.client = &http.Client{Timeout: 120 * time.Second}
	return nil
}

// OnConfigurationChange читает настройки плагина из System Console.
func (p *Plugin) OnConfigurationChange() error {
	cfg := new(configuration)
	if err := p.API.LoadPluginConfiguration(cfg); err != nil {
		return err
	}
	cfg.BackendURL = strings.TrimSpace(cfg.BackendURL)
	cfg.APIToken = strings.TrimSpace(cfg.APIToken)

	p.configurationLock.Lock()
	p.configuration = cfg
	p.configurationLock.Unlock()
	return nil
}

func (p *Plugin) config() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()
	if p.configuration == nil {
		return &configuration{}
	}
	return p.configuration
}

// ServeHTTP проксирует /api/* в бэкенд: /api/tickets → {backend}/api/v1/plugin/tickets.
// Тело передаётся как есть (multipart поддерживается), добавляется заголовок
// Authorization: Bearer <api_token>. После успешного создания заявки отправляет
// пользователю эфемерный пост-подтверждение.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	cfg := p.config()

	if !strings.HasPrefix(r.URL.Path, apiPrefix) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if cfg.BackendURL == "" {
		http.Error(w, "backend_url is not configured", http.StatusInternalServerError)
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, apiPrefix)
	target := strings.TrimSuffix(cfg.BackendURL, "/") + backendAPIPrefix + rel

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "failed to build request", http.StatusBadGateway)
		return
	}
	outReq.URL.RawQuery = r.URL.RawQuery
	outReq.Header = r.Header.Clone()
	stripHopByHopHeaders(outReq.Header)
	if token := cfg.APIToken; token != "" {
		outReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.client.Do(outReq)
	if err != nil {
		p.API.LogError("plugin proxy upstream error", "url", target, "error", err.Error())
		http.Error(w, "failed to reach backend", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Непривязанный канал: отдаём закэшированный ответ, не трогая бэкенд.
	var cacheHit *contextNegEntry
	if r.Method == http.MethodPost && rel == "/context" {
		channelID := r.URL.Query().Get("channelId")
		if channelID != "" {
			cacheHit, _ = p.cachedContext(channelID)
		}
	}

	if cacheHit == nil {
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
	}

	if cacheHit != nil {
		w.WriteHeader(cacheHit.status)
		_, _ = w.Write(cacheHit.body)
		return
	}

	w.WriteHeader(resp.StatusCode)

	if r.Method == http.MethodPost && rel == "/context" {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			if p.isUnboundContext(body) {
				p.storeContext(r.URL.Query().Get("channelId"), resp.StatusCode, resp.Header, body)
			}
			_, _ = w.Write(body)
			return
		}
	}

	if r.Method == http.MethodPost && rel == "/tickets" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			p.ephemeralCreateConfirmation(r, body)
		}
		_, _ = w.Write(body)
		return
	}

	_, _ = io.Copy(w, resp.Body)
}

// cachedContext возвращает живой негативный ответ /context для канала или nil.
func (p *Plugin) cachedContext(channelID string) (*contextNegEntry, bool) {
	p.contextLock.RLock()
	defer p.contextLock.RUnlock()

	ent, ok := p.contextCache[channelID]
	if !ok || time.Since(ent.ts) > contextNegCacheTTL {
		return nil, false
	}
	return ent, true
}

// storeContext сохраняет негативный ответ /context до истечения TTL.
func (p *Plugin) storeContext(channelID string, status int, header http.Header, body []byte) {
	if channelID == "" {
		return
	}

	p.contextLock.Lock()
	defer p.contextLock.Unlock()

	if p.contextCache == nil {
		p.contextCache = make(map[string]*contextNegEntry)
	}
	now := time.Now()
	for k, ent := range p.contextCache {
		if now.Sub(ent.ts) > contextNegCacheTTL {
			delete(p.contextCache, k)
		}
	}
	p.contextCache[channelID] = &contextNegEntry{
		ts:     now,
		status: status,
		header: header.Clone(),
		body:   body,
	}
}

// isUnboundContext определяет по телу ответа, что канал не привязан к реалму
// (backend отвечает 200 с {"data":{"bound":false}}).
func (p *Plugin) isUnboundContext(body []byte) bool {
	var res struct {
		Data struct {
			Bound *bool `json:"bound"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return false
	}
	return res.Data.Bound != nil && !*res.Data.Bound
}

// ephemeralCreateConfirmation читает ответ создания заявки и шлёт эфемерный пост
// автору в текущий канал. userId/channelId передаются webapp-плагином в query,
// т.к. серверная часть не может получить их из тела запроса.
func (p *Plugin) ephemeralCreateConfirmation(r *http.Request, body []byte) {
	userID := r.URL.Query().Get("userId")
	channelID := r.URL.Query().Get("channelId")
	if userID == "" || channelID == "" {
		return
	}

	var res struct {
		Data struct {
			ID     string `json:"id"`
			Number int    `json:"number"`
			Title  string `json:"title"`
			Link   string `json:"link"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil || res.Data.ID == "" {
		return
	}

	msg := fmt.Sprintf("Заявка №%d создана.\n**Заголовок:** %s", res.Data.Number, res.Data.Title)
	if res.Data.Link != "" {
		msg += "\nОткрыть: " + res.Data.Link
	}

	post := &model.Post{ChannelId: channelID, Message: msg}
	if p.API.SendEphemeralPost(userID, post) == nil {
		p.API.LogError("failed to send ephemeral create confirmation", "user_id", userID)
	}
}

// stripHopByHopHeaders удаляет hop-by-hop заголовки и Content-Length, чтобы
// транспорт Go сам корректно пересчитал длину тела.
func stripHopByHopHeaders(h http.Header) {
	for _, key := range []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Transfer-Encoding",
		"Upgrade", "TE", "Trailer", "Content-Length",
	} {
		h.Del(key)
	}
	h.Del("Accept-Encoding")
}
