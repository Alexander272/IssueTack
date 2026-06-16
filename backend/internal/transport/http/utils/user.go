package utils

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
)

func GetActor(c *gin.Context) *models.Actor {
	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return nil
	}
	user, ok := u.(models.User)
	if !ok {
		response.SendError(c, fmt.Errorf("invalid user type in context"))
		return nil
	}

	actor := &models.Actor{
		ID:   user.ID,
		Name: user.Name,
	}
	return actor
}

func GetUser(c *gin.Context) *models.User {
	u, exists := c.Get(constants.CtxUser)
	if !exists {
		response.SendError(c, models.ErrSessionEmpty)
		return nil
	}
	user, ok := u.(models.User)
	if !ok {
		response.SendError(c, fmt.Errorf("invalid user type in context"))
		return nil
	}

	return &user
}
