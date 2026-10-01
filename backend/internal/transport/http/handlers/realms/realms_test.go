package realms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSyncMattermost реализует services.Mattermost через встраивание интерфейса:
// переопределён только SyncRealmUsers, остальные методы не вызываются.
type stubSyncMattermost struct {
	services.Mattermost
	result *services.SyncUsersResult
	err    error

	called     bool
	gotRealmID uuid.UUID
	gotActor   *models.Actor
	gotTeams   []string
}

func (s *stubSyncMattermost) SyncRealmUsers(_ context.Context, realmID uuid.UUID, actor *models.Actor, teamNames []string) (*services.SyncUsersResult, error) {
	s.called = true
	s.gotRealmID = realmID
	s.gotActor = actor
	s.gotTeams = teamNames
	return s.result, s.err
}

// syncHandler собирает gin-роутер с одним хендлером и подставленным в контекст
// пользователем — так проверяется проброс id realm'а и актора в сервис.
func syncHandler(t *testing.T, stub *stubSyncMattermost, user models.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	h := NewHandler(nil, stub)
	engine := gin.New()
	engine.POST("/realms/:id/mattermost/sync", func(c *gin.Context) {
		c.Set(constants.CtxUser, user)
		h.syncMattermostUsers(c)
	})
	return engine
}

func syncRequest(t *testing.T, engine *gin.Engine, realmID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/realms/"+realmID+"/mattermost/sync", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestSyncMattermostUsers_ReturnsCounters(t *testing.T) {
	realmID := uuid.New()
	actorID := uuid.New()
	stub := &stubSyncMattermost{result: &services.SyncUsersResult{Created: 12, Linked: 3, Failed: 1}}

	engine := syncHandler(t, stub, models.User{ID: actorID, Name: "admin"})
	rec := syncRequest(t, engine, realmID.String())

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp struct {
		Data services.SyncUsersResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 12, resp.Data.Created)
	assert.Equal(t, 3, resp.Data.Linked)
	assert.Equal(t, 1, resp.Data.Failed, "счётчик ошибок доезжает до веба")

	assert.Equal(t, realmID, stub.gotRealmID)
	require.NotNil(t, stub.gotActor)
	assert.Equal(t, actorID, stub.gotActor.ID, "актор из контекста идёт в сервис для аудита")
	assert.Nil(t, stub.gotTeams, "кнопка синка берёт всех пользователей сервера")
}

func TestSyncMattermostUsers_RejectsBadRealmID(t *testing.T) {
	stub := &stubSyncMattermost{result: &services.SyncUsersResult{}}
	engine := syncHandler(t, stub, models.User{ID: uuid.New()})

	rec := syncRequest(t, engine, "not-a-uuid")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "BR001", "некорректный id realm'а — доменная ошибка ввода")
	assert.False(t, stub.called, "до сервиса запрос не доходит")
}

func TestSyncMattermostUsers_PropagatesPermissionError(t *testing.T) {
	stub := &stubSyncMattermost{err: models.ErrPermissionDenied}
	engine := syncHandler(t, stub, models.User{ID: uuid.New()})

	rec := syncRequest(t, engine, uuid.New().String())
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "AU002")
}

func TestSyncMattermostUsers_PropagatesServiceError(t *testing.T) {
	stub := &stubSyncMattermost{err: models.ErrInvalidInput}
	engine := syncHandler(t, stub, models.User{ID: uuid.New()})

	rec := syncRequest(t, engine, uuid.New().String())
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "BR001")
}

func TestSyncMattermostUsers_RequiresActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubSyncMattermost{result: &services.SyncUsersResult{}}
	h := NewHandler(nil, stub)

	engine := gin.New()
	engine.POST("/realms/:id/mattermost/sync", h.syncMattermostUsers)

	rec := syncRequest(t, engine, uuid.New().String())
	assert.NotEqual(t, http.StatusOK, rec.Code, "без пользователя в контексте запрос не выполняется")
	assert.False(t, stub.called, "без актора синк не запускается")
}
