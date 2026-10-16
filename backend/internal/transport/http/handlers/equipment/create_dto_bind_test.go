package equipment

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateEquipmentDTO_BindEmbedded защищает биндинг создания комплекта:
// поля единицы приходят в JSON плоско (CreateEquipmentDTO встраивает
// EquipmentDTO), а рядом лежит необязательный состав components.
//
// Без встроенной структуры components пришлось бы слать отдельным запросом, а
// если бы валидатор не заходил во встроенную структуру, отсутствие typeId/name
// доходило бы до сервиса и падало 500 вместо 400.
func TestCreateEquipmentDTO_BindEmbedded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Кастомный enum-валидатор регистрирует приложение (handler.go) — в тесте
	// его надо зарегистрировать самим, иначе биндинг падает в панику на теге
	// `binding:"required,enum"` у Status.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		require.NoError(t, v.RegisterValidation("enum", models.UniversalEnumValidator))
	}

	router := gin.New()
	router.POST("/bind", func(c *gin.Context) {
		dto := &models.CreateEquipmentDTO{}
		if err := c.ShouldBindJSON(dto); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, dto)
	})

	const typeUUID = "0f3d4a5b-1111-4222-8333-444455556666"
	const componentUUID = "aa11bb22-3333-4444-8555-666677778888"

	t.Run("комплект с составом", func(t *testing.T) {
		body := []byte(`{"name":"Комплект","status":"in_stock",` +
			`"isKit":true,"components":[{"componentId":"` + componentUUID + `"}]}`)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/bind", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"isKit":true`)
		assert.Contains(t, w.Body.String(), componentUUID)
	})

	t.Run("нет typeId — допустимо для комплекта/но не валидирует отдельно", func(t *testing.T) {
		body := []byte(`{"name":"Без типа","status":"in_stock"}`)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/bind", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Биндинг без валидации обязательности typeId для единицы проходит здесь.
		// Правило «единица требует тип» живёт на уровне сервиса.
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("нет name", func(t *testing.T) {
		body := []byte(`{"typeId":"` + typeUUID + `","status":"in_stock"}`)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/bind", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("единица без components", func(t *testing.T) {
		body := []byte(`{"typeId":"` + typeUUID + `","name":"Ноутбук","status":"in_stock"}`)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/bind", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), typeUUID)
	})
}
