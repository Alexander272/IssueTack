package ws

import (
	"log"
	"net/http"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WsHandler struct {
	hub            *ws_hub.Hub
	services       *services.Services
	allowedOrigins map[string]struct{}
}

func NewWsHandler(hub *ws_hub.Hub, services *services.Services, allowedOrigins []string) *WsHandler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = struct{}{}
	}
	return &WsHandler{
		hub:            hub,
		services:       services,
		allowedOrigins: origins,
	}
}

func (h *WsHandler) upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}
			if len(h.allowedOrigins) == 0 {
				return false
			}
			_, ok := h.allowedOrigins[origin]
			return ok
		},
	}
}

func (h *WsHandler) HandleWS(c *gin.Context) {
	u, exists := c.Get(constants.CtxUser)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user, ok := u.(models.User)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	conn, err := h.upgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("failed to upgrade ws connection: %v", err)
		return
	}

	client := ws_hub.NewClient(conn, h.hub)
	client.UserID = user.ID
	client.Subscribe("user:" + user.ID.String())

	if err := h.services.Notifications.SendUnread(c.Request.Context(), client); err != nil {
		log.Printf("failed to send unread notifications: %v", err)
	}

	go client.WritePump(30*time.Second, 10*time.Second)
	client.ReadPump(60 * time.Second)
}
