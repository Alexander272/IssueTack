package tickets

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
	service services.Tickets
}

func NewHandler(service services.Tickets) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Tickets) {
	handlers := NewHandler(service)

	tickets := api.Group("/tickets")
	{
		tickets.GET("", handlers.getAll)
		tickets.GET("/:id", handlers.getByID)
		tickets.POST("", handlers.create)
		tickets.PUT("/:id", handlers.update)
		tickets.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	filter := &models.TicketFilter{}

	if siteID := c.Query("siteId"); siteID != "" {
		id, err := uuid.Parse(siteID)
		if err == nil {
			filter.SiteID = &id
		}
	}
	if status := c.Query("status"); status != "" {
		s := models.TicketStatus(status)
		filter.Status = &s
	}
	if ownerID := c.Query("ownerId"); ownerID != "" {
		id, err := uuid.Parse(ownerID)
		if err == nil {
			filter.OwnerID = &id
		}
	}
	if assigneeID := c.Query("assigneeId"); assigneeID != "" {
		id, err := uuid.Parse(assigneeID)
		if err == nil {
			filter.AssigneeID = &id
		}
	}
	if groupID := c.Query("groupId"); groupID != "" {
		id, err := uuid.Parse(groupID)
		if err == nil {
			filter.GroupID = &id
		}
	}

	data, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err, filter)
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

	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}
	user := u.(models.User)
	data, err := h.service.GetByID(c, &models.GetTicketByIdDTO{ID: id, Actor: models.Actor{ID: user.ID, Name: user.Name}})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.TicketDTO{}
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
	dto.Actor = models.Actor{ID: user.ID, Name: user.Name}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Заявка создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.TicketDTO{}
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
	dto.Actor = models.Actor{ID: user.ID, Name: user.Name}

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: dto.ID, Message: "Заявка обновлена"})
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

	if err := h.service.Delete(c, &models.DeleteTicketDTO{ID: id, Actor: models.Actor{ID: user.ID, Name: user.Name}}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
