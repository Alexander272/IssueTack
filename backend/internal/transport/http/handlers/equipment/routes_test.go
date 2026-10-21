package equipment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegister_Routes проверяет, что дерево маршрутов оборудования собирается:
// статические префиксы (/types, /templates, /acts) соседствуют с параметром /:id
// в одном и том же методе. Если gin не разведёт их — Register паникует на старте
// приложения, а не на компиляции. Тест ловит это без поднятия сервера и БД:
// хендлеры не вызываются, только регистрируются.
func TestRegister_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		Register(api, nil, 1024, &middleware.Middleware{})
	})
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
