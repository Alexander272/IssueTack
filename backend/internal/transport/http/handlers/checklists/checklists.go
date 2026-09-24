package checklists

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.Checklists
}

func NewHandler(service services.Checklists) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Checklists, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	// Доступ к шаблонам чек-листов проверяется на уровне сервиса: без права
	// checklist:read пользователь видит и меняет только свои шаблоны.
	checklists := api.Group("/checklists")
	{
		checklists.GET("", handlers.getAll)
		checklists.GET("/:id", handlers.getByID)
		checklists.GET("/:id/items", handlers.getItems)
		checklists.POST("", handlers.create)
		checklists.PUT("/:id", handlers.update)
		checklists.PUT("/:id/items", handlers.setItems)
		checklists.POST("/:id/apply/:ticketId", handlers.apply)
		checklists.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	dto := &models.GetChecklistTemplatesDTO{}

	if realmID := c.Query("realmId"); realmID != "" {
		id, err := uuid.Parse(realmID)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		dto.RealmID = id
	}
	if ctxRealm, ok := utils.GetRealmUUID(c); ok {
		dto.RealmID = ctxRealm
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	data, err := h.service.Get(c, dto)
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

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	data, err := h.service.GetByID(c, &models.GetChecklistTemplateDTO{ID: id}, actor.ID, c.GetHeader("realm"))
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getItems(c *gin.Context) {
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

	data, err := h.service.GetItems(c, id, actor.ID, c.GetHeader("realm"))
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.ChecklistTemplateDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if ctxRealm, ok := utils.GetRealmUUID(c); ok {
		dto.RealmID = ctxRealm
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Шаблон создан"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.ChecklistTemplateDTO{}
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

	if err := h.service.Update(c, dto, actor.ID, c.GetHeader("realm")); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: dto.ID, Message: "Шаблон обновлен"})
}

func (h *Handler) setItems(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	var items []*models.ChecklistTemplateItemDTO
	if err := c.BindJSON(&items); err != nil {
		response.SendError(c, err)
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	if err := h.service.SetItems(c, nil, id, items, actor.ID, c.GetHeader("realm")); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Элементы шаблона обновлены"})
}

func (h *Handler) apply(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	ticketID, err := uuid.Parse(c.Param("ticketId"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	if err := h.service.ApplyTemplate(c, nil, &models.ApplyTemplateDTO{TicketID: ticketID, TemplateID: templateID, Actor: actor}); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Шаблон применен"})
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

	if err := h.service.Delete(c, &models.DelChecklistTemplateDTO{ID: id}, actor.ID, c.GetHeader("realm")); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
