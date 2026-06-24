package utils

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetRealmUUID(c *gin.Context) (uuid.UUID, bool) {
	s := c.GetHeader("realm")
	if s == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: invalid realm header: %v", models.ErrInvalidInput, err))
		return uuid.Nil, false
	}
	return id, true
}
