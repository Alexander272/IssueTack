package middleware

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

func (m *Middleware) CheckPermissions(perms ...access.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exists := c.Get(constants.CtxUser)
		if !exists {
			response.SendError(c, models.ErrSessionEmpty)
			return
		}
		user, ok := u.(models.User)
		if !ok {
			response.SendError(c, fmt.Errorf("invalid user type in context"))
			return
		}

		var accessAllowed bool
		var lastErr error

		realmId := c.GetHeader("realm")
		if realmId == "" {
			realmId = c.Query("realm")
		}

		for _, r := range perms {
			ok, err := m.services.AccessPolices.Enforce(user.ID.String(), realmId, string(r.Resource), string(r.Action))
			if err != nil {
				lastErr = err
				continue
			}
			if ok {
				accessAllowed = true
				break
			}
		}

		if lastErr != nil && !accessAllowed {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrPolicyCheck, lastErr))
			return
		}

		if !accessAllowed {
			response.SendError(c, models.ErrPermissionDenied)
			return
		}

		c.Next()
	}
}
