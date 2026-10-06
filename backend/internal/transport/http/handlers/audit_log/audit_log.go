package audit_log

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
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

	logs := api.Group("/audit", middleware.CheckPermissions(access.Reg.R(access.ResourceAudit).Read()))
	{
		logs.GET("", handler.getAll)
		logs.GET("/by-realm/:realmId", handler.getByRealm)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.Get(c, &models.GetAuditLogsDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

// getByRealm: Casbin проверяет реалм из заголовка, а не из пути, поэтому path-параметр
// обязан совпадать с авторизованным — иначе читался бы журнал чужой области.
func (h *Handler) getByRealm(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	requested, err := uuid.Parse(c.Param("realmId"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	if requested != realmID {
		response.SendError(c, models.ErrNotFound)
		return
	}

	data, err := h.service.GetByRealm(c, &models.GetAuditLogsByRealmDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}
