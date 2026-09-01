package tickets

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
	service services.Tickets
}

func NewHandler(service services.Tickets) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Tickets, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	tickets := api.Group("/tickets", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		tickets.GET("", handlers.getAll)
		tickets.GET("/:id", handlers.getByID)
		tickets.POST("/:id/take", handlers.take)
		tickets.POST("/:id/transfer", handlers.transfer)

		tickets.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Write()))
		tickets.POST("", handlers.create)
		tickets.PUT("/:id", handlers.update)

		tickets.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Delete()))
		tickets.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	filter := &models.TicketFilter{}
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if ok {
		filter.RealmID = &realmID
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	filter.Actor = actor

	if favType := c.Query("favoritesType"); favType != "" {
		ft := models.FavoriteType(favType)
		if !ft.IsValid() {
			response.SendError(c, fmt.Errorf("%w: invalid favoritesType", models.ErrInvalidInput))
			return
		}
		filter.FavoritesByUser = &actor.ID
		filter.FavoriteType = &ft
	}

	data, total, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err, filter)
		return
	}
	response.SendData(c, data, total)
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

	realmIdStr := c.GetHeader("realm")

	data, err := h.service.GetByID(c, &models.GetTicketByIdDTO{ID: id, Actor: actor, RealmID: realmIdStr})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) take(c *gin.Context) {
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

	realmIdStr := c.GetHeader("realm")

	if err := h.service.Take(c, &models.TakeTicketDTO{ID: id, Actor: actor, RealmID: realmIdStr}); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Заявка взята в работу"})
}

func (h *Handler) transfer(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.TransferTicketDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = &id
	if *dto.AssigneeID == uuid.Nil {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	realmIdStr := c.GetHeader("realm")
	dto.RealmID = realmIdStr

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Transfer(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Заявка передана"})
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.TicketDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}
	dto.RealmID = &realmID

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

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
	if id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if ok {
		dto.RealmID = &realmID
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

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

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	realmIdStr := c.GetHeader("realm")

	if err := h.service.Delete(c, &models.DeleteTicketDTO{ID: id, Actor: actor, RealmID: realmIdStr}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
