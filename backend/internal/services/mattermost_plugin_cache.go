package services

import (
	"slices"
	"sync"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// pluginContextCacheTTL — срок жизни положительных ответов, связанных с
// контекстом плагина. Кэш сглаживает всплеск запросов /plugin/context при
// включении плагина (шапка канала опрашивает контекст у каждого клиента).
const pluginContextCacheTTL = time.Minute

// pluginSettingsEntry — закэшированные настройки интеграции для scope.
type pluginSettingsEntry struct {
	settings *models.RealmMattermost
	expires  time.Time
}

// pluginRealmEntry — справочники реалма, общие для всех его пользователей.
type pluginRealmEntry struct {
	realmName  string
	categories []*models.Category
	sites      []*models.Site
	groups     []models.GroupShort
	expires    time.Time
}

// pluginContextCache хранит результаты дорогих резолвов плагина: настройки
// интеграции по scope (включая проверку состава DM) и справочники реалма.
// Негативные результаты намеренно не кэшируются.
type pluginContextCache struct {
	mu       sync.Mutex
	settings map[string]pluginSettingsEntry
	realms   map[uuid.UUID]pluginRealmEntry
}

func newPluginContextCache() *pluginContextCache {
	return &pluginContextCache{
		settings: make(map[string]pluginSettingsEntry),
		realms:   make(map[uuid.UUID]pluginRealmEntry),
	}
}

// pluginScopeCacheKey собирает ключ по всем входам resolvePluginSettings:
// каналу, боту и Mattermost-пользователю (последний влияет на проверку DM).
func pluginScopeCacheKey(scope models.PluginScope) string {
	return scope.ChannelID + "|" + scope.BotUserID + "|" + scope.MmUserID
}

func (c *pluginContextCache) getSettings(key string) (*models.RealmMattermost, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.settings[key]
	if !ok || time.Now().After(ent.expires) {
		return nil, false
	}
	cp := *ent.settings
	return &cp, true
}

func (c *pluginContextCache) setSettings(key string, settings *models.RealmMattermost) {
	if settings == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := *settings
	c.settings[key] = pluginSettingsEntry{settings: &cp, expires: time.Now().Add(pluginContextCacheTTL)}
}

func (c *pluginContextCache) getRealm(id uuid.UUID) (pluginRealmEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.realms[id]
	if !ok || time.Now().After(ent.expires) {
		return pluginRealmEntry{}, false
	}
	ent.categories = slices.Clone(ent.categories)
	ent.sites = slices.Clone(ent.sites)
	ent.groups = slices.Clone(ent.groups)
	return ent, true
}

func (c *pluginContextCache) setRealm(id uuid.UUID, realmName string, categories []*models.Category, sites []*models.Site, groups []models.GroupShort) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.realms[id] = pluginRealmEntry{
		realmName:  realmName,
		categories: slices.Clone(categories),
		sites:      slices.Clone(sites),
		groups:     slices.Clone(groups),
		expires:    time.Now().Add(pluginContextCacheTTL),
	}
}
