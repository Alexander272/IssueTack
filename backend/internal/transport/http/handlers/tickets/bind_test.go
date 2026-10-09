package tickets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTicketFilter_BindUUIDParams защищает биндинг uuid-полей фильтра из query-параметров.
// Gin не умеет писать uuid.UUID ([16]byte) в форму без тега parser — без него на любой
// непустой assigneeId/ownerId приходит 400 «is not valid value for uuid.UUID».
func TestTicketFilter_BindUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("enum", models.UniversalEnumValidator))
	}

	router := gin.New()
	router.GET("/bind", func(c *gin.Context) {
		filter := &models.TicketFilter{}
		if err := c.ShouldBindQuery(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filter)
	})

	const ownerUUID = "0f3d4a5b-1111-4222-8333-444455556666"
	const assigneeUUID = "1f4e5a6b-2222-4333-8444-555566667777"
	const realmUUID = "2f5e6a7b-3333-4333-8444-555566667777"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bind?ownerId="+ownerUUID+"&assigneeId="+assigneeUUID+"&realmId="+realmUUID+"&mode=created", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.TicketFilter
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, uuid.MustParse(ownerUUID), *got.OwnerID)
	assert.Equal(t, uuid.MustParse(assigneeUUID), *got.AssigneeID)
	assert.Equal(t, uuid.MustParse(realmUUID), *got.RealmID)
	assert.NotNil(t, got.Mode)
	assert.Equal(t, "created", *got.Mode)
}
