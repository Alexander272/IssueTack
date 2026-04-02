package response

import (
	"errors"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Используем Generics для типизации данных
type DataResponse[T any] struct {
	Data  T   `json:"data"`
	Total int `json:"total,omitempty"`
}

type IdResponse struct {
	Id      interface{} `json:"id,omitempty"` // interface{} если ID может быть string или int
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type AppError struct {
	Status  int
	Code    string
	Message string
}

var errorRegistry = map[error]*AppError{
	// 404 Not Found
	models.ErrNotFound: {http.StatusNotFound, "NF001", "Ничего не найдено"},

	// 409 Conflict
	models.ErrAlreadyExists: {http.StatusConflict, "AE001", "Запись уже существует"},

	// 400 Bad Request
	models.ErrInvalidInput:          {http.StatusBadRequest, "BR001", "Переданы некорректные данные"},
	models.ErrRelatedRecordNotFound: {http.StatusBadRequest, "BR002", "Указанный связанный ресурс не существует"},

	// 401 & 403
	models.ErrUnauthenticated:  {http.StatusUnauthorized, "AU001", "Требуется авторизация"},
	models.ErrPermissionDenied: {http.StatusForbidden, "AU002", "Недостаточно прав для выполнения операции"},

	// 500 & 504
	models.ErrInternal:         {http.StatusInternalServerError, "SRV01", "Внутренняя ошибка сервера"},
	models.ErrDeadlineExceeded: {http.StatusGatewayTimeout, "SRV02", "Время ожидания операции истекло"},

	models.ErrReservedRole:          {http.StatusBadRequest, "RL001", "Нельзя создать или обновить зарезервированную роль"},
	models.ErrCannotInheritFromSelf: {http.StatusBadRequest, "RL002", "Роль не может наследоваться от самой себя"},
	models.ErrParentRoleNotFound:    {http.StatusNotFound, "RL003", "Указанная родительская роль не найдена"},
	models.ErrCircularInheritance:   {Status: http.StatusConflict, Code: "RL004", Message: "Обнаружено циклическое наследование ролей"},
}

// Централизованный метод для отправки ошибок
func SendError(c *gin.Context, err error, customMessage ...string) {
	meta := &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "U001",
		Message: "Внутренняя ошибка сервера",
	}

	// Ищем ошибку в реестре (раскрываем цепочку ошибок через errors.Is)
	for domainErr, registeredMeta := range errorRegistry {
		if errors.Is(err, domainErr) {
			meta = registeredMeta
			break
		}
	}

	// Если передано кастомное сообщение — используем его вместо стандартного
	finalMessage := meta.Message
	if len(customMessage) > 0 && customMessage[0] != "" {
		finalMessage = customMessage[0]
	}

	// Логируем системные детали
	logger.Error("request_failed",
		logger.StringAttr("url", c.Request.URL.Path),
		logger.StringAttr("method", c.Request.Method),
		logger.StringAttr("ip", c.ClientIP()),
		logger.StringAttr("error", err.Error()), // Тут будет реальная ошибка БД
		logger.StringAttr("code", meta.Code),
	)

	c.AbortWithStatusJSON(meta.Status, ErrorResponse{
		Message: finalMessage,
		Code:    meta.Code,
	})
}

// Вспомогательный метод для успеха
func SendData[T any](c *gin.Context, data T, total int) {
	c.JSON(http.StatusOK, DataResponse[T]{
		Data:  data,
		Total: total,
	})
}
