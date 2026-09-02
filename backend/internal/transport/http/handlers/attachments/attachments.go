package attachments

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler — обработчики вложений. Чтение и удаление идут через Attachments-сервис,
// а загрузка — через Tickets (сервис-владелец агрегата тикета), чтобы уведомление о
// новом вложении рассылалось на стороне домена, а не в транспортном слое.
type Handler struct {
	service    services.Attachments
	attachment services.Tickets
}

func NewHandler(service services.Attachments, attachment services.Tickets) *Handler {
	return &Handler{
		service:    service,
		attachment: attachment,
	}
}

func Register(api *gin.RouterGroup, service services.Attachments, attachment services.Tickets, middleware *middleware.Middleware) {
	handlers := NewHandler(service, attachment)

	attachments := api.Group("/attachments", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		attachments.GET("/content/:id", handlers.getContent)
		attachments.GET("/:entityType/:entityId", handlers.getByEntity)

		attachments.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Write()))
		attachments.POST("/:entityType/:entityId", handlers.upload)

		attachments.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Delete()))
		attachments.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getContent(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	realmIdStr := c.GetHeader("realm")

	att, reader, err := h.service.GetContent(c, id, user.ID, realmIdStr)
	if err != nil {
		response.SendError(c, err)
		return
	}
	defer reader.Close()

	mimeType := att.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(att.FileName))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	size := att.FileSize
	if size == 0 {
		if info, err := os.Stat(att.FilePath); err == nil {
			size = info.Size()
		}
	}

	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, att.FileName))
	c.DataFromReader(http.StatusOK, size, mimeType, reader, nil)
}

func (h *Handler) getByEntity(c *gin.Context) {
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")

	id, err := uuid.Parse(entityID)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	realmIdStr := c.GetHeader("realm")

	dto := &models.EntityAccessDTO{
		EntityType: entityType,
		EntityID:   id,
		ActorID:    user.ID,
		Realm:      realmIdStr,
	}

	data, err := h.service.GetByEntity(c, dto)
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

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	defer file.Close()

	realmIdStr := c.GetHeader("realm")

	dto := &models.UploadAttachmentDTO{
		EntityType: entityType,
		EntityID:   id,
		FileName:   header.Filename,
		FileSize:   header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		File:       file,
		UploadedBy: user.ID,
		Realm:      realmIdStr,
	}

	att, err := h.attachment.UploadAttachment(c, nil, dto)
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

	user := utils.GetUser(c)
	if user == nil {
		return
	}

	realmIdStr := c.GetHeader("realm")

	dto := &models.DeleteAttachmentDTO{
		ID:      id,
		ActorID: user.ID,
		Realm:   realmIdStr,
	}

	if err := h.service.Delete(c, nil, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
