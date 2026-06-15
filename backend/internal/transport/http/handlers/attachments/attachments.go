package attachments

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
	service services.Attachments
}

func NewHandler(service services.Attachments) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Attachments) {
	handlers := NewHandler(service)

	attachments := api.Group("/attachments")
	{
		attachments.GET("/:entityType/:entityId", handlers.getByEntity)
		attachments.POST("/:entityType/:entityId", handlers.upload)
		attachments.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getByEntity(c *gin.Context) {
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")

	id, err := uuid.Parse(entityID)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	data, err := h.service.GetByEntity(c, entityType, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) upload(c *gin.Context) {
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")

	id, err := uuid.Parse(entityID)
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

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	defer file.Close()

	att, err := h.service.Upload(c, nil, entityType, id, header.Filename, file, user.ID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: att.ID, Message: "Файл загружен"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.service.Delete(c, nil, id); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
