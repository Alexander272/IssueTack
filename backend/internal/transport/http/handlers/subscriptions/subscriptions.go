package subscriptions

import (
	"fmt"

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
	service services.Subscriptions
}

func NewHandler(service services.Subscriptions) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Subscriptions, middleware *middleware.Middleware) {
	h := NewHandler(service)

	subscriptions := api.Group("/tickets/:id/subscription", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		subscriptions.GET("", h.isSubscribed)
		subscriptions.POST("", h.subscribe)
		subscriptions.DELETE("", h.unsubscribe)
	}
}

func (h *Handler) parseTicketID(c *gin.Context) (uuid.UUID, bool) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return uuid.Nil, false
	}
	return ticketID, true
}

func (h *Handler) isSubscribed(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	subscribed, err := h.service.IsSubscribed(c.Request.Context(), &models.IsSubscribedDTO{
		ActorID:  user.ID,
		TicketID: ticketID,
	})
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, map[string]bool{"subscribed": subscribed})
}

func (h *Handler) subscribe(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	if err := h.service.Subscribe(c.Request.Context(), &models.SubscribeDTO{
		ActorID:  user.ID,
		TicketID: ticketID,
	}); err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, response.IdResponse{Message: "Подписка оформлена"})
}

func (h *Handler) unsubscribe(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	if err := h.service.Unsubscribe(c.Request.Context(), &models.SubscribeDTO{
		ActorID:  user.ID,
		TicketID: ticketID,
	}); err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, response.IdResponse{Message: "Подписка отменена"})
}
