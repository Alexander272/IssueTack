package subtasks

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.Subtasks
}

func NewHandler(service services.Subtasks) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Subtasks) {
	handlers := NewHandler(service)

	subtasks := api.Group("/tickets/:ticketId/subtasks")
	{
		subtasks.GET("", handlers.getByTicket)
		subtasks.POST("", handlers.create)
		subtasks.PUT("/:id", handlers.update)
		subtasks.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getByTicket(c *gin.Context) {
	ticketId := c.Param("ticketId")
	id, err := uuid.Parse(ticketId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	data, err := h.service.GetByTicketID(c, id, user.ID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.SubtaskDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	ticketId := c.Param("ticketId")
	id, err := uuid.Parse(ticketId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	dto.TicketID = id

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Create(c, nil, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Подзадача создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.SubtaskDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
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

	if err := h.service.Update(c, nil, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: dto.ID, Message: "Подзадача обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto := &models.DelSubtaskDTO{ID: id, Actor: actor}

	if err := h.service.Delete(c, nil, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
