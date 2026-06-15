package users

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
	service services.Users
}

func NewHandler(service services.Users) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, services *services.Services, middleware *middleware.Middleware) {
	handler := NewHandler(services.Users)

	users := api.Group("/users", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Read()))
	{
		users.GET("", handler.getAll)
		users.GET("/:id", handler.getByID)

		write := users.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceGroup).Write()))
		{
			write.POST("/sync", handler.sync)
			write.PUT("/:id", handler.updateAccount)
		}
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.GetAll(c)
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
	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	actor := &models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}

	if err := h.service.Sync(c, actor); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователи синхронизированы"})
}

func (h *Handler) updateAccount(c *gin.Context) {
	dto := &models.UpdateAccountDTO{}
	if err := c.BindJSON(dto); err != nil {
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

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	dto.Actor = &models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}

	if err := h.service.UpdateAccount(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователь обновлен"})
}


