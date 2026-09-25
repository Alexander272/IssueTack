package mattermost

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxPluginBodyBytes — общий лимит тела multipart-запроса создания заявки из
// плагина MM (все файлы сразу). Пер-файл лимит (по умолчанию 10 МБ) отдельно
// enforce-ится в AttachmentService при загрузке.
const maxPluginBodyBytes = 64 << 20

// maxPluginFiles — ограничение числа файлов в одном запросе плагина.
const maxPluginFiles = 10

// registerPluginRoutes монтирует /plugin/* под guard-ом источника и Bearer-токеном.
func (h *Handler) registerPluginRoutes(r *gin.RouterGroup, cfg config.MattermostConfig) {
	plugin := r.Group("/plugin")
	plugin.Use(middleware.SourceGuard(cfg))
	plugin.Use(middleware.PluginTokenGuard(cfg.PluginToken))
	{
		plugin.POST("/context", h.handlePluginContext)
		plugin.GET("/tickets", h.handlePluginListMine)
		plugin.POST("/tickets", h.handlePluginCreateTicket)
		plugin.GET("/tickets/:id", h.handlePluginGetTicket)
		plugin.GET("/tickets/:id/comments", h.handlePluginGetComments)
		plugin.POST("/tickets/:id/comments", h.handlePluginCreateComment)
		plugin.GET("/attachments/:id", h.handlePluginAttachmentContent)
	}
}

// pluginContextRequest — запрос /plugin/context.
type pluginContextRequest struct {
	ChannelID string `json:"channelId" binding:"required"`
	UserID    string `json:"userId" binding:"required"`
}

// handlePluginContext возвращает контекст канала (реалм, справочники,
// пользователь). Используется webapp-плагином для решения, показывать ли иконку,
// и для наполнения формы создания заявки.
func (h *Handler) handlePluginContext(c *gin.Context) {
	var req pluginContextRequest
	if err := utils.BindJSON(c, &req); err != nil {
		response.SendError(c, err)
		return
	}

	ctx, err := h.service.PluginContext(c, req.ChannelID, req.UserID)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, ctx)
}

func (h *Handler) handlePluginListMine(c *gin.Context) {
	channelID := c.Query("channelId")
	userID := c.Query("userId")
	if channelID == "" || userID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	list, err := h.service.PluginListMine(c, channelID, userID)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, list)
}

func (h *Handler) handlePluginGetTicket(c *gin.Context) {
	channelID := c.Query("channelId")
	userID := c.Query("userId")
	if channelID == "" || userID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	ticketID := c.Param("id")
	if ticketID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	detail, err := h.service.PluginGetTicket(c, channelID, userID, ticketID)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, detail)
}

// handlePluginCreateTicket создаёт заявку из multipart-формы плагина MM
// (поля + files[]). Возвращает id, номер, заголовок и ссылку.
func (h *Handler) handlePluginCreateTicket(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPluginBodyBytes)

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.SendError(c, models.ErrFileTooLarge)
			return
		}
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	input := &services.PluginCreateTicketInput{
		ChannelID:   c.Request.FormValue("channelId"),
		MmUserID:    c.Request.FormValue("userId"),
		Title:       strings.TrimSpace(c.Request.FormValue("title")),
		Description: strings.TrimSpace(c.Request.FormValue("description")),
	}

	if input.ChannelID == "" || input.MmUserID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	if input.Title == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	if raw := c.Request.FormValue("categoryId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		input.CategoryID = id
	}
	if raw := c.Request.FormValue("siteId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		input.SiteID = id
	}

	files := c.Request.MultipartForm.File["files"]
	if len(files) > maxPluginFiles {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		input.Files = append(input.Files, services.PluginFile{
			FileName: fh.Filename,
			FileSize: int64(len(data)),
			MimeType: fh.Header.Get("Content-Type"),
			Content:  bytes.NewReader(data),
		})
	}

	dto, err := h.service.PluginCreateTicket(c, input)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, gin.H{
		"id":     dto.ID,
		"number": dto.Number,
		"title":  dto.Title,
		"link":   dto.Link,
	})
}

func (h *Handler) handlePluginGetComments(c *gin.Context) {
	channelID := c.Query("channelId")
	userID := c.Query("userId")
	if channelID == "" || userID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	ticketID := c.Param("id")

	comments, err := h.service.PluginGetComments(c, channelID, userID, ticketID)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, comments)
}

func (h *Handler) handlePluginCreateComment(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPluginBodyBytes)

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.SendError(c, models.ErrFileTooLarge)
			return
		}
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	input := &services.PluginCreateCommentInput{
		ChannelID: c.Request.FormValue("channelId"),
		MmUserID:  c.Request.FormValue("userId"),
		TicketID:  c.Param("id"),
		Text:      strings.TrimSpace(c.Request.FormValue("text")),
	}

	if input.ChannelID == "" || input.MmUserID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	files := c.Request.MultipartForm.File["files"]
	if len(files) > maxPluginFiles {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			response.SendError(c, models.ErrInvalidInput)
			return
		}
		input.Files = append(input.Files, services.PluginFile{
			FileName: fh.Filename,
			FileSize: int64(len(data)),
			MimeType: fh.Header.Get("Content-Type"),
			Content:  bytes.NewReader(data),
		})
	}

	comment, err := h.service.PluginCreateComment(c, input)
	if err != nil {
		response.SendError(c, err)
		return
	}

	response.SendData(c, comment)
}

func (h *Handler) handlePluginAttachmentContent(c *gin.Context) {
	channelID := c.Query("channelId")
	userID := c.Query("userId")
	if channelID == "" || userID == "" {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	att, reader, err := h.service.PluginGetAttachmentContent(c, channelID, userID, c.Param("id"))
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
