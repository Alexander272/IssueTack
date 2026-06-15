package audit_log

import (
	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.AuditLogs
}

func NewHandler(service services.AuditLogs) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.AuditLogs, middleware *middleware.Middleware) {
	handler := NewHandler(service)

	logs := api.Group("/audit-log", middleware.CheckPermissions(access.Reg.R(access.ResourceAudit).Read()))
	{
		logs.GET("", handler.getAll)
		logs.GET("/by-realm/:realmId", handler.getByRealm)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.Get(c, &models.GetAuditLogsDTO{})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByRealm(c *gin.Context) {
	realmID, err := uuid.Parse(c.Param("realmId"))
	if err != nil {
		response.SendError(c, err)
		return
	}

	data, err := h.service.GetByRealm(c, &models.GetAuditLogsByRealmDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}
