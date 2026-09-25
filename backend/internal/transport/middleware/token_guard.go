package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

// PluginTokenGuard проверяет Authorization: Bearer <token> (constant-time).
// Пустой токен в конфиге — группа полностью отключена.
func PluginTokenGuard(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			response.SendError(c, models.ErrInvalidToken)
			return
		}
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") || subtle.ConstantTimeCompare([]byte(auth), []byte("Bearer "+token)) != 1 {
			response.SendError(c, models.ErrInvalidToken)
			return
		}
		c.Next()
	}
}