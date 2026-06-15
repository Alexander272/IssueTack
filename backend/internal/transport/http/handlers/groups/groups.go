package groups

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.Groups
}

func NewHandler(service services.Groups) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Groups, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	groups := api.Group("/groups", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Read()))
	{
		groups.GET("", handlers.getAll)
		groups.GET("/:id", handlers.getByID)

		write := groups.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Write()))
		{
			write.POST("", handlers.create)
			write.PUT("/:id", handlers.update)
			write.POST("/members/add", handlers.addMember)
			write.POST("/members/remove", handlers.removeMember)
		}

		delete := groups.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Delete()))
		{
			delete.DELETE("/:id", handlers.delete)
		}
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.Get(c, &models.GetGroupsDTO{})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByID(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	data, err := h.service.GetByID(c, &models.GetGroupDTO{ID: id})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.GroupDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Группа создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.GroupDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if id != dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = id

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Группа обновлена"})
}

func (h *Handler) addMember(c *gin.Context) {
	dto := &models.GroupMemberDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.AddMember(c, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Участник добавлен"})
}

func (h *Handler) removeMember(c *gin.Context) {
	dto := &models.GroupMemberDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.RemoveMember(c, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.service.Delete(c, &models.DelGroupDTO{ID: id}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
