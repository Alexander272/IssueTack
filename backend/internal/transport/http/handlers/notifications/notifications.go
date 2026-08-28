package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
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

func Register(api *gin.RouterGroup, service services.Notifications, middleware *middleware.Middleware) {
	h := NewHandler(service)

	notifications := api.Group("/notifications", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		notifications.GET("", h.getSettings)
		notifications.PUT("/settings", h.updateSettings)
		notifications.PUT("/:id/read", h.markRead)
	}
}

// getSettings возвращает персональные настройки уведомлений текущего пользователя.
func (h *Handler) getSettings(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	settings, err := h.service.GetSettings(c.Request.Context(), user.ID)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, settings)
}

// updateSettings сохраняет персональные настройки уведомлений текущего пользователя.
func (h *Handler) updateSettings(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	var body map[string]bool
	if err := utils.BindJSON(c, &body); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	settings, err := json.Marshal(body)
	if err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.SaveSettings(c.Request.Context(), user.ID, settings); err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, response.IdResponse{Message: "Настройки сохранены"})
}

func (h *Handler) markRead(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
