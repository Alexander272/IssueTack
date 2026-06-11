package response

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestSendError_WithHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "Domain Error Not Found",
			err:        models.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "NF001",
			wantMsg:    "Ничего не найдено",
		},
		{
			name:       "Domain Error Internal",
			err:        models.ErrInternal,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SRV01",
			wantMsg:    "Внутренняя ошибка сервера",
		},
		{
			name:       "Wrapped Domain Error",
			err:        fmt.Errorf("some wrapper: %w", models.ErrAlreadyExists),
			wantStatus: http.StatusConflict,
			wantCode:   "AE001",
			wantMsg:    "Запись уже существует",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			SendError(c, tt.err)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantCode)
			assert.Contains(t, w.Body.String(), tt.wantMsg)
		})
	}
}

func TestSendError_WithUnknownError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	err := errors.New("some random error")
	SendError(c, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "U001")
	assert.Contains(t, w.Body.String(), "Внутренняя ошибка сервера")
}

func TestSendData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	data := []string{"a", "b"}
	SendData(c, data, 2)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
	assert.Contains(t, w.Body.String(), "total")
}

func TestHTTPErrorInterface(t *testing.T) {
	var _ HTTPError = models.ErrNotFound
	var _ HTTPError = models.ErrInternal
	var _ HTTPError = models.ErrAlreadyExists
}

func TestSendError_WithValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Используем реальную валидацию для генерации ошибок
	type TestDTO struct {
		Email    string `validate:"required,email"`
		Password string `validate:"min=8"`
	}

	v := validator.New()
	dto := TestDTO{} // Пустая структура вызовет ошибки required и min

	err := v.Struct(dto)
	assert.Error(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	SendError(c, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "BR001")
	assert.Contains(t, w.Body.String(), "Отправлены некорректные данные")
	assert.Contains(t, w.Body.String(), `"field":"Email"`)
	assert.Contains(t, w.Body.String(), `"field":"Password"`)
	// Проверяем русскоязычные сообщения
	assert.Contains(t, w.Body.String(), "обязательное поле")
	assert.Contains(t, w.Body.String(), "значение меньше допустимого минимума")
}
