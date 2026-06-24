package users

import (
	"fmt"
	"net/http"

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
	service services.Users
}

func NewHandler(service services.Users) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, services services.Users, middleware *middleware.Middleware) {
	handler := NewHandler(services)

	users := api.Group("/users", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Read()))
	{
		users.GET("/by-realm", handler.getByRealm)
		users.GET("/:id", handler.getByID)

		users.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Write()))
		users.GET("", handler.getAll)
		users.POST("/sync", handler.sync)
		users.PUT("/:id", handler.updateAccount)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.GetAll(c, nil)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByRealm(c *gin.Context) {
	var realmID *uuid.UUID
	if id, ok := utils.GetRealmUUID(c); ok {
		realmID = &id
	}

	data, err := h.service.GetAll(c, realmID)
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

	data, err := h.service.GetByID(c, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) sync(c *gin.Context) {
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	if err := h.service.Sync(c, actor); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователи синхронизированы"})
}

func (h *Handler) updateAccount(c *gin.Context) {
	dto := &models.UpdateAccountDTO{}
	if err := c.ShouldBindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	if id != dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = id

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.UpdateAccount(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователь обновлен"})
}
