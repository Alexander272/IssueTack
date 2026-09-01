package comments

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	text := strings.TrimSpace(c.Request.FormValue("text"))
	if text == "" {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, errors.New("поле text обязательно")))
		return
	}

	isInternal, _ := strconv.ParseBool(c.Request.FormValue("isInternal"))
	commentType := c.Request.FormValue("type")

	dto := &models.CreateCommentDTO{
		Text:       text,
		TicketID:   ticketID,
		IsInternal: isInternal,
		Type:       commentType,
		UserID:     user.ID,
		Realm:      realm,
		Files:      nil,
	}

	if files := c.Request.MultipartForm.File["files"]; len(files) > 0 {
		dto.Files = make([]*models.UploadAttachmentDTO, 0, len(files))
		for _, fh := range files {
			f, openErr := fh.Open()
			if openErr != nil {
				response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, openErr))
				return
			}
			defer f.Close()
			dto.Files = append(dto.Files, &models.UploadAttachmentDTO{
				EntityType: "ticket",
				EntityID:   ticketID,
				FileName:   fh.Filename,
				FileSize:   fh.Size,
				MimeType:   fh.Header.Get("Content-Type"),
				File:       f,
				UploadedBy: user.ID,
				Realm:      realm,
			})
		}
	}

	comment, err := h.service.Create(c, nil, dto)
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
