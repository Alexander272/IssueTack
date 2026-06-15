package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service    services.Session
	middleware *middleware.Middleware
	auth       config.AuthConfig
}

type Deps struct {
	Service    services.Session
	Middleware *middleware.Middleware
	Auth       config.AuthConfig
}

func NewAuthHandlers(deps Deps) *AuthHandler {
	return &AuthHandler{
		service:    deps.Service,
		middleware: deps.Middleware,
		auth:       deps.Auth,
	}
}

func Register(api *gin.RouterGroup, deps Deps) {
	handlers := NewAuthHandlers(deps)

	auth := api.Group("/auth")
	{
		auth.POST("/sign-in", handlers.SignIn)
		auth.POST("/sign-out", handlers.SignOut)
		auth.POST("refresh", handlers.Refresh)
	}
}

func (h *AuthHandler) SignIn(c *gin.Context) {
	dto := &models.SignIn{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	user, err := h.service.SignIn(c, *dto)
	if err != nil {
		logger.Info("Неудачная попытка авторизации",
			logger.StringAttr("section", "auth"),
			logger.StringAttr("ip", c.ClientIP()),
			logger.StringAttr("username", dto.Username),
			logger.ErrAttr(err),
		)

		if strings.Contains(err.Error(), "invalid_grant") {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err), dto)
			return
		}
		response.SendError(c, err, dto)
		return
	}

	domain := h.auth.Domain
	if !strings.Contains(c.Request.Host, domain) {
		domain = c.Request.Host
	}

	logger.Info("Пользователь успешно авторизовался",
		logger.StringAttr("section", "auth"),
		logger.StringAttr("ip", c.ClientIP()),
		logger.StringAttr("user", user.Name),
		logger.StringAttr("user_id", user.ID.String()),
	)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(constants.AuthCookie, user.RefreshToken, int(h.auth.RefreshTokenTTL.Seconds()), "/", domain, h.auth.Secure, true)
	response.SendData(c, user)
}

func (h *AuthHandler) SignOut(c *gin.Context) {
	refreshToken, err := c.Cookie(constants.AuthCookie)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionEmpty, err))
		return
	}

	if err := h.service.SignOut(c, refreshToken); err != nil {
		response.SendError(c, err)
		return
	}

	domain := h.auth.Domain
	if !strings.Contains(c.Request.Host, domain) {
		domain = c.Request.Host
	}

	logger.Info("Пользователь вышел из системы",
		logger.StringAttr("section", "auth"),
		logger.StringAttr("ip", c.ClientIP()),
	)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(constants.AuthCookie, "", -1, "/", domain, h.auth.Secure, true)
	c.JSON(http.StatusNoContent, response.IdResponse{})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(constants.AuthCookie)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionEmpty, err))
		return
	}

	if c.Request.ContentLength > 0 {
		dto := &models.SignIn{}
		if err := c.ShouldBindJSON(dto); err != nil {
			response.SendError(c, err)
			return
		}
	}

	user, err := h.service.Refresh(c, refreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionEmpty, err))
			return
		}
		response.SendError(c, err)
		return
	}

	domain := h.auth.Domain
	if !strings.Contains(c.Request.Host, domain) {
		domain = c.Request.Host
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(constants.AuthCookie, user.RefreshToken, int(h.auth.RefreshTokenTTL.Seconds()), "/", domain, h.auth.Secure, true)
	response.SendData(c, user)
}
