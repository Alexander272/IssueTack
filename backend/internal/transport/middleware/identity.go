package middleware

import (
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

func (m *Middleware) VerifyToken(c *gin.Context) {
	token := strings.Replace(c.GetHeader("Authorization"), "Bearer ", "", 1)

	// TODO надо попробовать забирать из keycloak ключи и проверять токен здесь
	result, err := m.keycloak.Client.RetrospectToken(c, token, m.keycloak.ClientId, m.keycloak.ClientSecret, m.keycloak.Realm)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionEmpty, err))
		return
	}

	if result == nil || result.Active == nil || !*result.Active {
		response.SendError(c, models.ErrSessionExpired)
		return
	}

	user, err := m.services.Session.DecodeAccessToken(c, token)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidToken, err))
		return
	}

	c.Set(constants.CtxUser, *user)
	c.Next()
}
