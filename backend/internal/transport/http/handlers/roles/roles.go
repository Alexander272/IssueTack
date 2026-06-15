package roles

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.Roles
}

func NewHandler(service services.Roles) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Roles, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	roles := api.Group("/roles", middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Read()))
	{
		roles.GET("", handlers.getAll)
		roles.GET("/all/stats", handlers.getWithStats)
		roles.GET("/:id", handlers.get)
		roles.GET("/:id/permissions", handlers.getWithPermissions)

		write := roles.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Write()))
		{
			write.POST("", handlers.create)
			write.PUT("/:id", handlers.update)
			write.PUT("/:id/permissions", handlers.setPermissions)
		}

		delete := roles.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Delete()))
		{
			delete.DELETE("/:id", handlers.delete)
		}
	}
}

func (h *Handler) getAll(c *gin.Context) {
	roles, err := h.service.GetAll(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, roles, len(roles))
}

func (h *Handler) get(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	role, err := h.service.GetOne(c, &models.GetRoleDTO{ID: id})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, role)
}

func (h *Handler) getWithStats(c *gin.Context) {
	roles, err := h.service.GetWithStats(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, roles, len(roles))
}

func (h *Handler) getWithPermissions(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	role, err := h.service.GetOneWithPermissions(c, &models.GetRoleDTO{ID: id})
	if err != nil {
		response.SendError(c, err, id)
		return
	}
	response.SendData(c, role)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.RoleDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	dto.Actor = models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.JSON(http.StatusCreated, response.IdResponse{Message: "Роль создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.RoleDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if id != dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = id

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	dto.Actor = models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.JSON(http.StatusOK, response.IdResponse{Message: "Роль обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	dto := &models.DeleteRoleDTO{ID: id}

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	dto.Actor = models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}

	if err := h.service.Delete(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) setPermissions(c *gin.Context) {
	var req struct {
		PermissionIDs []string `json:"permissionIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		response.SendError(c, err)
		return
	}

	roleID := c.Param("id")

	if err := h.service.SetPermissions(c, roleID, req.PermissionIDs); err != nil {
		response.SendError(c, err, req)
		return
	}

	c.JSON(http.StatusOK, response.IdResponse{Message: "Права роли обновлены"})
}
