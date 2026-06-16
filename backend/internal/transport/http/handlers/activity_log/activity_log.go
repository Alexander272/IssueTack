package activity_log

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.ActivityLog
}

func NewHandler(service services.ActivityLog) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.ActivityLog, middleware *middleware.Middleware) {
	handler := NewHandler(service)

	logs := api.Group("/activity-log", middleware.CheckPermissions(access.Reg.R(access.ResourceActivity).Read()))
	{
		logs.GET("", handler.getAll)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	dto := &models.GetLogsDTO{}

	if entityID := c.Query("entityId"); entityID != "" {
		id, err := uuid.Parse(entityID)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		dto.EntityID = &id
	}
	if entityType := c.Query("entityType"); entityType != "" {
		dto.EntityType = &entityType
	}
	if parentID := c.Query("parentId"); parentID != "" {
		id, err := uuid.Parse(parentID)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		dto.ParentID = &id
	}
	if realmID := c.Query("realmId"); realmID != "" {
		id, err := uuid.Parse(realmID)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		dto.RealmID = &id
	}

	data, err := h.service.Get(c, dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}
