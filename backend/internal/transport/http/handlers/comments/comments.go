package comments

import (
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	services *services.Services
}

func NewHandler(services *services.Services) *Handler {
	return &Handler{
		services: services,
	}
}

func Register(api *gin.RouterGroup, services *services.Services) {
	h := NewHandler(services)
	//TODO реализовать

	comments := api.Group("/tickets/:ticketId/comments")
	{
		comments.GET("", h.getByTicket)
		comments.POST("", h.create)
		comments.DELETE("/:id", h.delete)
	}
}

func (h *Handler) getByTicket(c *gin.Context) {
	response.SendError(c, models.ErrNotImplemented)
}

func (h *Handler) create(c *gin.Context) {
	response.SendError(c, models.ErrNotImplemented)
}

func (h *Handler) delete(c *gin.Context) {
	response.SendError(c, models.ErrNotImplemented)
}
