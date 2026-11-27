package equipment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// routePaths собирает дерево маршрутов оборудования без поднятия сервера и БД.
func routePaths(engine *gin.Engine) map[string]bool {
	routes := make(map[string]bool)
	for _, r := range engine.Routes() {
		routes[r.Method+" "+strings.TrimPrefix(r.Path, "/api")] = true
	}
	return routes
}

func wantRoutes(groups ...[]string) map[string]bool {
	want := make(map[string]bool)
	for _, group := range groups {
		for _, r := range group {
			want[r] = true
		}
	}
	return want
}

func baseRoutes() map[string]bool {
	return wantRoutes(
		[]string{
			"GET /equipment",
			"GET /equipment/:id",
			"GET /equipment/:id/children",
			"GET /equipment/:id/assignments",
			"GET /equipment/:id/events",
			"GET /equipment/:id/composition",
			"GET /equipment/:id/composition/history",
			"GET /equipment/tree",
			"GET /equipment/stats",
			"GET /equipment/types",
			"GET /equipment/types/:id",
			"GET /equipment/shops",
			"GET /equipment/templates",
			"GET /equipment/templates/:id",
			"GET /equipment/acts",
			"GET /equipment/acts/:id",
			"GET /equipment/acts/:id/document",
			"GET /equipment/acts/:id/signed",
			"GET /equipment/import/template",
		},
		[]string{
			"POST /equipment",
			"PUT /equipment/:id",
			"POST /equipment/:id/assign",
			"POST /equipment/:id/unassign",
			"POST /equipment/:id/status",
			"POST /equipment/:id/defect",
			"POST /equipment/:id/write-off",
			"POST /equipment/:id/composition",
			"DELETE /equipment/:id/composition",
			"POST /equipment/:id/composition/attach",
			"POST /equipment/:id/composition/detach",
			"POST /equipment/types",
			"PUT /equipment/types/:id",
			"POST /equipment/templates",
			"PUT /equipment/templates/:id",
			"POST /equipment/acts",
			"POST /equipment/acts/:id/signed",
			"POST /equipment/import",
		},
		[]string{
			"DELETE /equipment/:id",
			"DELETE /equipment/types/:id",
			"DELETE /equipment/templates/:id",
		},
	)
}

// TestRegister_Routes проверяет полное дерево маршрутов оборудования после
// разнесения регистрации по тематическим файлам: при переносе маршрута легко
// потерять его или случайно отдать под другой уровень прав. Уровни прав заданы
// явными группами read/write/delete в Register, а не порядком вызовов register*.
// Статические префиксы (/types, /templates, /acts) соседствуют с параметром
// /:id: если gin их не разведёт, Register паникует на старте приложения, а не
// на компиляции.
func TestRegister_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		Register(api, Deps{Middleware: &middleware.Middleware{}})
	})

	want := baseRoutes()
	require.Len(t, engine.Routes(), len(want))
	assert.Equal(t, want, routePaths(engine))
}

// TestRegister_OptionalTaskRoutes — периодические задачи и правила автозадач
// register-функции подключают только при непустых сервисах.
func TestRegister_OptionalTaskRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		Register(api, Deps{
			Middleware:     &middleware.Middleware{},
			PeriodicTasks:  &services.EquipmentPeriodicTaskService{},
			OperationTasks: &services.EquipmentOperationTaskService{},
		})
	})

	want := baseRoutes()
	for _, r := range []string{
		"GET /equipment/periodic-tasks",
		"GET /equipment/periodic-tasks/templates/:id",
		"GET /equipment/periodic-tasks/instances",
		"GET /equipment/operation-tasks",
		"GET /equipment/operation-tasks/:id",
		"POST /equipment/periodic-tasks/templates",
		"PUT /equipment/periodic-tasks/templates/:id",
		"DELETE /equipment/periodic-tasks/templates/:id",
		"POST /equipment/periodic-tasks/run",
		"POST /equipment/operation-tasks",
		"PUT /equipment/operation-tasks/:id",
		"DELETE /equipment/operation-tasks/:id",
	} {
		want[r] = true
	}
	require.Len(t, engine.Routes(), len(want))
	assert.Equal(t, want, routePaths(engine))
}

// TestEquipmentRoutes_StaticBeatsParam закрывает регресс, из-за которого
// GET /equipment/templates отдавал 400 «invalid UUID length: 9»: запрос попадал
// в getByID с id="templates", потому что в дереве gin статический узел /templates
// не должен проигрывать параметру /:id. Регистрация маршрутов (TestRegister_Routes)
// этого не ловит: дерево собирается без паники, но выбор ветки зависит от порядка
// и набора статических детей в одном методе. Здесь набор путей и порядок повторяют
// Register, а запрос реально проходит через роутер — падение снова станет красным.
func TestEquipmentRoutes_StaticBeatsParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	equipment := engine.Group("/api/v1/equipment")

	// Только GET-ветка: в gin дерево строится по методам, поэтому POST/PUT/DELETE
	// на выбор ветки не влияют. Порядок — как в Register: параметр объявлен раньше.
	equipment.GET("", func(c *gin.Context) { c.String(http.StatusOK, "getAll") })
	equipment.GET("/:id", func(c *gin.Context) { c.String(http.StatusOK, "getByID") })
	equipment.GET("/tree", func(c *gin.Context) { c.String(http.StatusOK, "getTree") })
	equipment.GET("/stats", func(c *gin.Context) { c.String(http.StatusOK, "getStats") })
	equipment.GET("/types", func(c *gin.Context) { c.String(http.StatusOK, "getTypes") })
	equipment.GET("/shops", func(c *gin.Context) { c.String(http.StatusOK, "getShops") })
	equipment.GET("/templates", func(c *gin.Context) { c.String(http.StatusOK, "getTemplates") })
	equipment.GET("/templates/:id", func(c *gin.Context) { c.String(http.StatusOK, "getTemplateByID") })
	equipment.GET("/acts", func(c *gin.Context) { c.String(http.StatusOK, "getActs") })

	for path, want := range map[string]string{
		"/api/v1/equipment/tree":      "getTree",
		"/api/v1/equipment/stats":     "getStats",
		"/api/v1/equipment/types":     "getTypes",
		"/api/v1/equipment/shops":     "getShops",
		"/api/v1/equipment/templates": "getTemplates",
		"/api/v1/equipment/acts":      "getActs",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, path)
		assert.Equal(t, want, w.Body.String(), path)
	}
}
