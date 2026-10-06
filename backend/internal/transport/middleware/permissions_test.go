package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// fakeUserRealms — заглушка services.UserRealms: отвечает только на GetByUserAndRealm,
// который вызывает checkRealmMembership.
type fakeUserRealms struct {
	services.UserRealms
	membership *models.UserRealm
	err        error

	calls []uuid.UUID // домены, которые запрашивали
}

func (f *fakeUserRealms) GetByUserAndRealm(_ context.Context, userID, realmID uuid.UUID) (*models.UserRealm, error) {
	f.calls = append(f.calls, realmID)
	if f.err != nil {
		return nil, f.err
	}
	if f.membership == nil {
		return nil, models.ErrNotFound
	}
	return f.membership, nil
}

// fakeAccessPolicies разрешает всё в переданном домене и запоминает проверки.
type fakeAccessPolicies struct {
	services.AccessPolicies
	enforced []string
}

func (f *fakeAccessPolicies) Enforce(sub, dom, obj, act string) (bool, error) {
	f.enforced = append(f.enforced, sub+"|"+dom+"|"+obj+"|"+act)
	return true, nil
}

func newPermissionsMiddleware(realms *fakeUserRealms, policies *fakeAccessPolicies) *Middleware {
	return &Middleware{services: &services.Services{
		UserRealms:     realms,
		AccessPolicies: policies,
	}}
}

func runPermissions(t *testing.T, m *Middleware, realmHeader string) *httptest.ResponseRecorder {
	t.Helper()
	w, _, _ := runPermissionsCtx(t, m, realmHeader)
	return w
}

// runPermissionsCtx возвращает значения, которые мидлвар положил в контекст для
// следующих обработчиков: realm из контекста — единственный авторизованный.
func runPermissionsCtx(t *testing.T, m *Middleware, realmHeader string) (*httptest.ResponseRecorder, any, any) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	if realmHeader != "" {
		c.Request.Header.Set("realm", realmHeader)
	}
	c.Set(constants.CtxUser, models.User{ID: uuid.New()})

	var gotRealm, gotUser any
	handler := m.CheckPermissions(access.Reg.R(access.ResourceTicket).Read())
	probe := func(pc *gin.Context) {
		gotRealm, _ = pc.Get(constants.CtxRealm)
		gotUser, _ = pc.Get(constants.CtxUser)
	}
	for _, h := range []gin.HandlerFunc{handler, probe} {
		h(c)
	}
	return w, gotRealm, gotUser
}

// TestCheckPermissions_ForeignRealmDenied — регрессия на IDOR: заголовок realm задаёт
// Casbin-домен, поэтому без проверки членства пользователь подставлял чужой realm и
// получал coarse-права ticket:read в чужой области (тикеты грузятся по id без
// realm-фильтра). Чужой realm должен отсекаться до Enforce.
func TestCheckPermissions_ForeignRealmDenied(t *testing.T) {
	realmID := uuid.New()
	realms := &fakeUserRealms{err: models.ErrNotFound}
	policies := &fakeAccessPolicies{}
	w := runPermissions(t, newPermissionsMiddleware(realms, policies), realmID.String())

	assert.Equal(t, http.StatusForbidden, w.Code, "чужой realm должен давать 403")
	assert.Equal(t, []uuid.UUID{realmID}, realms.calls, "членство проверяется для домена из заголовка")
	assert.Empty(t, policies.enforced, "Enforce не должен вызываться для чужого realm")
}

// TestCheckPermissions_InactiveMembershipDenied — деактивированное членство не даёт прав,
// даже если запись о нём есть.
func TestCheckPermissions_InactiveMembershipDenied(t *testing.T) {
	realmID := uuid.New()
	realms := &fakeUserRealms{membership: &models.UserRealm{RealmID: realmID, IsActive: false}}
	policies := &fakeAccessPolicies{}
	w := runPermissions(t, newPermissionsMiddleware(realms, policies), realmID.String())

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, policies.enforced, "Enforce не должен вызываться при неактивном членстве")
}

// TestCheckPermissions_OwnRealmAllowed — контроль кейса выше: активное членство
// пропускает запрос, а домен в Enforce остаётся тем, что прислал клиент.
func TestCheckPermissions_OwnRealmAllowed(t *testing.T) {
	realmID := uuid.New()
	realms := &fakeUserRealms{membership: &models.UserRealm{RealmID: realmID, IsActive: true}}
	policies := &fakeAccessPolicies{}
	w := runPermissions(t, newPermissionsMiddleware(realms, policies), realmID.String())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []uuid.UUID{realmID}, realms.calls)
	assert.Len(t, policies.enforced, 1)
	assert.Contains(t, policies.enforced[0], realmID.String())
}

// TestCheckPermissions_MalformedRealmDenied — не-UUID в заголовке realm отсекается:
// такой домен не может быть валидным Casbin-доменом, но молчаливый отказ по правам
// плохо диагностируется.
func TestCheckPermissions_MalformedRealmDenied(t *testing.T) {
	realms := &fakeUserRealms{}
	policies := &fakeAccessPolicies{}
	w := runPermissions(t, newPermissionsMiddleware(realms, policies), "not-a-uuid")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, realms.calls, "невалидный UUID не должен идти в БД")
	assert.Empty(t, policies.enforced)
}

// TestCheckPermissions_EmptyRealmSkipsMembership — пустой realm не проверяется:
// поведение нужно для маршрутов вне реалмов, и Enforce на пустом домене сам
// ничего не разрешит.
func TestCheckPermissions_EmptyRealmSkipsMembership(t *testing.T) {
	realms := &fakeUserRealms{}
	policies := &fakeAccessPolicies{}
	w := runPermissions(t, newPermissionsMiddleware(realms, policies), "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, realms.calls, "пустой realm не требует проверки членства")
	assert.Len(t, policies.enforced, 1)
	assert.Contains(t, policies.enforced[0], "||")
}

// TestCheckPermissions_NoSession — без пользователя в контексте членство не проверяется.
func TestCheckPermissions_NoSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	realms := &fakeUserRealms{}
	policies := &fakeAccessPolicies{}
	m := newPermissionsMiddleware(realms, policies)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	c.Request.Header.Set("realm", uuid.New().String())

	m.CheckPermissions(access.Reg.R(access.ResourceTicket).Read())(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, realms.calls)
	assert.Empty(t, policies.enforced)
}

// TestCheckPermissions_RealmInContext — мидлвар обязан класть в контекст realm,
// под которым запрос разрешён: хендлеры берут его оттуда (utils.RequireRealmUUID),
// иначе можно было бы подменить realm в самом handler-слое.
func TestCheckPermissions_RealmInContext(t *testing.T) {
	realmID := uuid.New()
	realms := &fakeUserRealms{membership: &models.UserRealm{RealmID: realmID, IsActive: true}}
	policies := &fakeAccessPolicies{}

	w, gotRealm, gotUser := runPermissionsCtx(t, newPermissionsMiddleware(realms, policies), realmID.String())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, realmID.String(), gotRealm, "в контексте должен лежать авторизованный realm")
	assert.NotNil(t, gotUser)
}

// TestCheckPermissions_NoRealmInContext — на маршрутах вне реалмов домен пустой,
// и в контексте тоже пусто: хендлеры обязаны это учитывать (RequireRealmUUID).
func TestCheckPermissions_NoRealmInContext(t *testing.T) {
	realms := &fakeUserRealms{}
	policies := &fakeAccessPolicies{}

	_, gotRealm, _ := runPermissionsCtx(t, newPermissionsMiddleware(realms, policies), "")

	assert.Equal(t, "", gotRealm, "без заголовка домен пустой, авторизованного realm нет")
}
