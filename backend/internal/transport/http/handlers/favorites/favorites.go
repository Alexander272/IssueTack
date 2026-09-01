package favorites

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
	service services.TicketFavorites
}

func NewHandler(service services.TicketFavorites) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.TicketFavorites, middleware *middleware.Middleware) {
	h := NewHandler(service)

	api.GET("/favorites", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()), h.getByUser)

	favorite := api.Group("/tickets/:id/favorite", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		favorite.GET("", h.state)
		favorite.POST("", h.add)
		favorite.DELETE("", h.remove)
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

func (h *Handler) bindType(c *gin.Context) (models.FavoriteDTO, bool) {
	dto := models.FavoriteDTO{}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return dto, false
	}
	return dto, true
}

func (h *Handler) state(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	permanent, err := h.service.IsFavorite(c.Request.Context(), &models.IsFavoriteDTO{
		UserID:   user.ID,
		TicketID: ticketID,
	}, models.FavoriteTypePermanent)
	if err != nil {
		response.SendError(c, err)
		return
	}
	temporary, err := h.service.IsFavorite(c.Request.Context(), &models.IsFavoriteDTO{
		UserID:   user.ID,
		TicketID: ticketID,
	}, models.FavoriteTypeTemporary)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, map[string]bool{"permanent": permanent, "temporary": temporary})
}

func (h *Handler) add(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	dto, ok := h.bindType(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}
	dto.ActorID = user.ID
	dto.TicketID = ticketID

	if err := h.service.Add(c.Request.Context(), &dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	response.SendData(c, response.IdResponse{Message: "Заявка добавлена в избранное"})
}

func (h *Handler) remove(c *gin.Context) {
	ticketID, ok := h.parseTicketID(c)
	if !ok {
		return
	}
	dto, ok := h.bindType(c)
	if !ok {
		return
	}
	user := utils.GetUser(c)
	if user == nil {
		return
	}
	dto.ActorID = user.ID
	dto.TicketID = ticketID

	if err := h.service.Remove(c.Request.Context(), &dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	response.SendData(c, response.IdResponse{Message: "Заявка убрана из избранного"})
}

func (h *Handler) getByUser(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	favType := c.Query("type")
	if !models.FavoriteType(favType).IsValid() && favType != "" {
		response.SendError(c, fmt.Errorf("%w: invalid favorite type", models.ErrInvalidInput))
		return
	}
	if favType == "" {
		favType = string(models.FavoriteTypePermanent)
	}

	data, err := h.service.GetByUser(c.Request.Context(), user.ID, models.FavoriteType(favType))
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}
