package mattermost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/mattermost/mattermost/server/public/model"
)

type Handler struct {
	service services.Mattermost
}

func Register(r *gin.RouterGroup, svc services.Mattermost) {
	h := &Handler{service: svc}

	mm := r.Group("/mattermost")
	{
		mm.POST("/webhook", h.handleWebhook)
		mm.POST("/dialog/open", h.handleDialogOpen)
		mm.POST("/dialog/:realmId", h.handleDialogSubmission)
		mm.POST("/action", h.handleInteractiveAction)
		mm.POST("/event", h.handleWSEvent)
	}
}

func (h *Handler) handleWebhook(c *gin.Context) {
	token := c.GetHeader("Token")
	if token == "" {
		response.SendError(c, fmt.Errorf("missing token"))
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		response.SendError(c, fmt.Errorf("invalid form: %w", err))
		return
	}

	bodyToken := c.Request.FormValue("token")
	if bodyToken == "" {
		bodyToken = token
	}

	triggerID := c.Request.FormValue("trigger_id")
	userID := c.Request.FormValue("user_id")
	channelID := c.Request.FormValue("channel_id")
	message := c.Request.FormValue("text")

	fileIDs := c.Request.Form["file_ids"]

	err := h.service.HandleDM(c, &services.HandleDMInput{
		MmUserID:  userID,
		ChannelID: channelID,
		Message:   message,
		FileIDs:   fileIDs,
		TriggerID: triggerID,
	})
	if err != nil {
		response.SendError(c, fmt.Errorf("failed to process command: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) handleDialogOpen(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.SendError(c, fmt.Errorf("failed to read body: %w", err))
		return
	}

	var payload struct {
		TriggerId string            `json:"trigger_id"`
		UserID    string            `json:"user_id"`
		ChannelId string            `json:"channel_id"`
		PostId    string            `json:"post_id"`
		Context   map[string]string `json:"context"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		response.SendError(c, fmt.Errorf("invalid dialog open payload: %w", err))
		return
	}

	if payload.TriggerId == "" {
		response.SendError(c, fmt.Errorf("missing trigger_id"))
		return
	}

	if err := h.service.HandleDialogOpen(c, payload.TriggerId, payload.UserID, payload.ChannelId, payload.PostId, payload.Context); err != nil {
		response.SendError(c, fmt.Errorf("failed to open dialog: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) handleDialogSubmission(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.SendError(c, fmt.Errorf("failed to read body: %w", err))
		return
	}

	var submission model.SubmitDialogRequest
	if err := json.Unmarshal(body, &submission); err != nil {
		response.SendError(c, fmt.Errorf("invalid dialog submission: %w", err))
		return
	}

	if err := h.service.HandleDialogSubmission(c, &submission); err != nil {
		response.SendError(c, fmt.Errorf("failed to create ticket: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) handleInteractiveAction(c *gin.Context) {
	var payload struct {
		UserID    string            `json:"user_id"`
		ChannelID string            `json:"channel_id"`
		Context   map[string]string `json:"context"`
	}
	if err := utils.BindJSON(c, &payload); err != nil {
		response.SendError(c, fmt.Errorf("invalid interactive action payload: %w", err))
		return
	}

	post, err := h.service.HandleInteractiveAction(c, payload.UserID, payload.ChannelID, payload.Context)
	if err != nil {
		response.SendError(c, fmt.Errorf("failed to handle interactive action: %w", err))
		return
	}

	if post != nil {
		c.JSON(http.StatusOK, gin.H{"update": post})
	} else {
		c.Status(http.StatusOK)
	}
}

func (h *Handler) handleWSEvent(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.SendError(c, fmt.Errorf("failed to read body: %w", err))
		return
	}

	var raw struct {
		Event  string `json:"event"`
		UserID string `json:"user_id"`
		Data   struct {
			Post       string `json:"post"`
			ChannelID  string `json:"channel_id"`
			SenderName string `json:"sender_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		response.SendError(c, fmt.Errorf("invalid WS event payload: %w", err))
		return
	}

	if raw.Event != "posted" || raw.Data.Post == "" {
		c.Status(http.StatusOK)
		return
	}

	var post model.Post
	if err := json.Unmarshal([]byte(raw.Data.Post), &post); err != nil {
		response.SendError(c, fmt.Errorf("invalid post in WS event: %w", err))
		return
	}

	if post.UserId == "" || (post.Message == "" && len(post.FileIds) == 0) {
		c.Status(http.StatusOK)
		return
	}

	err = h.service.HandleDM(c, &services.HandleDMInput{
		MmUserID:  post.UserId,
		ChannelID: post.ChannelId,
		Message:   post.Message,
		FileIDs:   post.FileIds,
	})
	if err != nil {
		logger.Error("failed to handle WS event",
			logger.StringAttr("user_id", post.UserId),
			logger.ErrAttr(err),
		)
	}

	c.Status(http.StatusOK)
}
