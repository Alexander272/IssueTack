package transport

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/limiter"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	keycloak *auth.KeycloakClient
	services *services.Services
	hub      *ws_hub.Hub
}

func NewHandler(keycloak *auth.KeycloakClient, services *services.Services, hub *ws_hub.Hub) *Handler {
	return &Handler{
		keycloak: keycloak,
		services: services,
		hub:      hub,
	}
}

func (h *Handler) Init(conf *config.Config) *gin.Engine {
	router := gin.Default()

	router.Use(
		limiter.Limit(conf.Limiter.RPS, conf.Limiter.Burst, conf.Limiter.TTL),
		gin.CustomRecovery(h.ErrorHandler),
	)

	router.GET("/api/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	h.initAPI(router, conf)

	return router
}

func (h *Handler) ErrorHandler(c *gin.Context, origErr any) {
	err := fmt.Errorf("unexpected error: %v", origErr)

	rawStack := string(debug.Stack())                        // 1. Получаем стек в виде байтов
	cleanStack := strings.ReplaceAll(rawStack, "\t", "    ") // 2. Заменяем все табуляции на 4 пробела для красоты
	stackLines := strings.Split(cleanStack, "\n")            // 3. Превращаем в срез строк, разделяя по символу \n

	error_bot.Send(c, err.Error(), gin.H{"PANIC": true, "Stack trace": stackLines})
	debug.PrintStack()
	response.SendError(c, err)
}

func (h *Handler) initAPI(router *gin.Engine, conf *config.Config) {
	// middleware := middleware.NewMiddleware(h.services, conf.Auth, h.keycloak)
	// handler := handlers.NewHandler(&handlers.Deps{Services: h.services, Conf: conf, Middleware: middleware})
	// handler := handlers.NewHandler(&handlers.Deps{Services: h.services, Conf: conf, Hub: h.hub})
	// wsHandler := ws.NewWsHandler(h.hub, conf.Http, h.services)

	// api := router.Group("/api")
	// handler.Init(api)

	// api.GET("/ws", func(c *gin.Context) {
	// 	wsHandler.HandleWS(c)
	// })
}
