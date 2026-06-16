package notifications

import (
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service services.Notifications
}

func NewHandler(service services.Notifications) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Notifications) {
	h := NewHandler(service)
	//TODO реализовать

	notifications := api.Group("/notifications")
	{
		notifications.GET("", h.getSettings)
		notifications.PUT("/settings", h.updateSettings)
		notifications.PUT("/:id/read", h.markRead)
	}
}

func (h *Handler) getSettings(c *gin.Context) {
	response.SendError(c, nil)
}

func (h *Handler) getUnread(c *gin.Context) {
	response.SendError(c, nil)
}

func (h *Handler) updateSettings(c *gin.Context) {
	response.SendError(c, nil)
}

func (h *Handler) markRead(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
