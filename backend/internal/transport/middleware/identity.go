package middleware

import (
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

// VerifyToken проверяет access-токен локально по JWKS Keycloak.
// Интроспекция не используется: с коротким TTL access-токена (~5 минут)
// цена валидного отозванного токена до истечения приемлема, взамен —
// меньше латентности, отказоустойчивее и меньше нагрузки на Keycloak.
func (m *Middleware) VerifyToken(c *gin.Context) {
	token := strings.TrimSpace(strings.Replace(c.GetHeader("Authorization"), "Bearer ", "", 1))
	if token == "" {
		response.SendError(c, models.ErrSessionEmpty)
		return
	}

	user, err := m.services.Session.DecodeAccessToken(c, token)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionExpired, err))
		return
	}

	c.Set(constants.CtxUser, *user)
	c.Next()
}
