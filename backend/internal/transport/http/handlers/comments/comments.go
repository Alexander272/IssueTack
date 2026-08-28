package comments

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
	service services.Comments
}

func NewHandler(service services.Comments) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Comments, middleware *middleware.Middleware) {
	h := NewHandler(service)

	comments := api.Group("/tickets/:id/comments", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		comments.GET("", h.getByTicket)

		comments.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Write()))
		comments.POST("", h.create)

		comments.DELETE("/:commentId", h.delete)
	}
}

func (h *Handler) getByTicket(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	realm := c.GetHeader("realm")

	data, err := h.service.GetByTicket(c, ticketID, user.ID, realm)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) create(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	realm := c.GetHeader("realm")

	var dto models.CreateCommentDTO
	if err := utils.BindJSON(c, &dto); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto.TicketID = ticketID
	dto.UserID = user.ID
	dto.Realm = realm

	comment, err := h.service.Create(c, nil, &dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: comment.ID, Message: "Комментарий добавлен"})
}

func (h *Handler) delete(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	if err := h.service.Delete(c, nil, &models.DeleteCommentDTO{
		ID:      commentID,
		ActorID: user.ID,
	}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
