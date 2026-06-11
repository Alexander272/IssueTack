package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

func (m *Middleware) VerifyToken(c *gin.Context) {
	token := strings.Replace(c.GetHeader("Authorization"), "Bearer ", "", 1)
	if token == "" {
		token = c.Query("token")
	}

	// TODO надо попробовать забирать из keycloak ключи и проверять токен здесь
	result, err := m.keycloak.Client.RetrospectToken(c, token, m.keycloak.ClientId, m.keycloak.ClientSecret, m.keycloak.Realm)
	if err != nil {
		domain := m.auth.Domain
		if !strings.Contains(c.Request.Host, domain) {
			domain = c.Request.Host
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(constants.AuthCookie, "", -1, "/", domain, m.auth.Secure, true)
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrSessionEmpty, err))
		return
	}

	if !*result.Active {
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
