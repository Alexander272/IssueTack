package permissions

import (
	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service services.Permissions
}

func NewHandler(service services.Permissions) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Permissions, middleware *middleware.Middleware) {
	handler := NewHandler(service)

	permissions := api.Group("/permissions", middleware.CheckPermissions(access.Reg.R(access.ResourcePerm).Read()))
	{
		permissions.GET("", handler.getAll)
		permissions.GET("/resources", handler.getResources)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.GetGrouped(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getResources(c *gin.Context) {
	data := access.Reg.List()
	response.SendData(c, data, len(data))
}
