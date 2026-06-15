package subtasks

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
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

	data, err := h.service.GetByTicketID(c, id)
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

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	actor := models.Actor{ID: user.ID, Name: user.Name}

	if err := h.service.Create(c, nil, dto, actor); err != nil {
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

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	actor := models.Actor{ID: user.ID, Name: user.Name}

	if err := h.service.Update(c, nil, dto, actor); err != nil {
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

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)

	dto := &models.DelSubtaskDTO{
		ID: id,
		Actor: models.Actor{
			ID:   user.ID,
			Name: user.Name,
		},
	}

	if err := h.service.Delete(c, nil, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
