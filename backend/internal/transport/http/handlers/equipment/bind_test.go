package equipment

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentActsFilter_BindUUIDParams защищает биндинг uuid-полей фильтра
// актов из query-параметров: без тега parser Gin отвечает 400 на любой непустой
// equipmentId/kitId («is not valid value for uuid.UUID»), и вкладка «Акты»
// падала бы на каждом открытии единицы.
func TestEquipmentActsFilter_BindUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/bind", func(c *gin.Context) {
		filter := &models.EquipmentActsFilter{}
		if err := c.ShouldBindQuery(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filter)
	})

	const equipmentUUID = "0f3d4a5b-1111-4222-8333-444455556666"
	const kitUUID = "1f4e5a6b-2222-4333-8444-555566667777"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bind?equipmentId="+equipmentUUID+"&kitId="+kitUUID+"&limit=50", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.EquipmentActsFilter
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.NotNil(t, got.EquipmentID)
	require.NotNil(t, got.KitID)
	assert.Equal(t, uuid.MustParse(equipmentUUID), *got.EquipmentID)
	assert.Equal(t, uuid.MustParse(kitUUID), *got.KitID)
	assert.Equal(t, 50, got.Limit)
}

// TestEquipmentFilter_BindUUIDParams — тот же регресс для фильтра списка: селектор
// «Место» шлёт siteId и locationId вместе. Без parser=encoding.TextUnmarshaler
// Gin отвечал бы 400 на непустой uuid, и фильтр был бы недостижим.
func TestEquipmentFilter_BindUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// enum-валидатор регистрирует приложение (handler.go); здесь его нет.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("enum", models.UniversalEnumValidator))
	}

	router := gin.New()
	router.GET("/bind", func(c *gin.Context) {
		filter := &models.EquipmentFilter{}
		if err := c.ShouldBindQuery(filter); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filter)
	})

	const siteUUID = "2f5d6a7c-3333-4444-8555-666677778888"
	const locationUUID = "3e6c7b8d-4444-5555-9666-777788889999"

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/bind?siteId="+siteUUID+"&locationId="+locationUUID+"&limit=50",
		nil,
	)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var got models.EquipmentFilter
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.NotNil(t, got.SiteID)
	require.NotNil(t, got.LocationID)
	assert.Equal(t, uuid.MustParse(siteUUID), *got.SiteID)
	assert.Equal(t, uuid.MustParse(locationUUID), *got.LocationID)
	assert.Equal(t, 50, got.Limit)
}
