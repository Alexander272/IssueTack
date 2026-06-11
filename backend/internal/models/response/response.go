package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/pkg/error_bot"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// validatorMessages сопоставляет теги валидатора с русскоязычными сообщениями
var validatorMessages = map[string]string{
	"required":    "обязательное поле",
	"email":       "некорректный email адрес",
	"min":         "значение меньше допустимого минимума",
	"max":         "значение превышает допустимый максимум",
	"len":         "неверная длина",
	"oneof":       "значение должно быть одним из допустимых",
	"url":         "некорректный URL",
	"numeric":     "значение должно быть числом",
	"alphanum":    "допустимы только буквы и цифры",
	"datetime":    "некорректный формат даты/времени",
	"uuid":        "некорректный UUID",
	"gte":         "значение должно быть не меньше",
	"lte":         "значение должно быть не больше",
	"gt":          "значение должно быть больше",
	"lt":          "значение должно быть меньше",
}

// translateTag переводит тег валидатора на русский язык
func translateTag(tag string) string {
	if msg, ok := validatorMessages[tag]; ok {
		return msg
	}
	return fmt.Sprintf("не прошло проверку '%s'", tag)
}

// HTTPError is an interface for errors that carry HTTP response metadata.
// Domain errors should implement this interface to allow the response package
// to handle them without a hard dependency on the models package.
type HTTPError interface {
	error
	Status() int
	Code() string
	Message() string
}

// DataResponse uses Generics for type-safe data responses.
type DataResponse[T any] struct {
	Data  T   `json:"data"`
	Total int `json:"total,omitempty"`
}

// IdResponse uses `any` instead of the deprecated `interface{}`.
type IdResponse struct {
	Id      any    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Message string      `json:"message"`
	Code    string      `json:"code,omitempty"`
	Fields  []FieldError `json:"fields,omitempty"`
}

// FieldError describes a specific field validation error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

// SendError is the centralized method for sending error responses.
// It automatically detects:
// 1. Gin/Validator validation errors (returns 400 with field details)
// 2. Custom HTTPError implementations (uses their metadata)
// 3. Unknown errors (returns 500)
// For server errors (5xx), it logs as Error and notifies the error_bot.
// For client errors (4xx), it logs as Info.
func SendError(c *gin.Context, err error, request ...any) {
	var status int
	var code string
	var message string
	var fields []FieldError

	// 1. First, check for Gin's validator errors (from c.BindJSON)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		status = http.StatusBadRequest
		code = "BR001"
		message = "Отправлены некорректные данные"
		for _, fe := range validationErrors {
			tag := fe.Tag()
			fields = append(fields, FieldError{
				Field:   fe.Field(),
				Message: translateTag(tag),
				Tag:     tag,
			})
		}
	} else {
		// 2. Check for custom HTTPError
		var httpErr HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.Status()
			code = httpErr.Code()
			message = httpErr.Message()
		} else {
			// 3. Unknown error
			status = http.StatusInternalServerError
			code = "U001"
			message = "Внутренняя ошибка сервера"
		}
	}

	loggerValues := []any{
		logger.StringAttr("url", c.Request.URL.Path),
		logger.StringAttr("method", c.Request.Method),
		logger.StringAttr("ip", c.ClientIP()),
		logger.StringAttr("error", err.Error()),
		logger.StringAttr("code", code),
	}

	// Determine log level and send to bot based on status code
	if status >= 500 {
		logger.Error("request_failed", loggerValues...)
		// Notify error bot for server errors
		error_bot.Send(c, err.Error(), extractRequest(request))
	} else {
		logger.Info("request_failed", loggerValues...)
	}

	c.AbortWithStatusJSON(status, ErrorResponse{
		Message: message,
		Code:    code,
		Fields:  fields,
	})
}

// SendData is a helper method for successful data responses.
func SendData[T any](c *gin.Context, data T, total ...int) {
	t := 0
	if len(total) > 0 {
		t = total[0]
	}
	c.JSON(http.StatusOK, DataResponse[T]{
		Data:  data,
		Total: t,
	})
}

// extractRequest safely extracts the request body from the variadic parameter.
func extractRequest(req []any) any {
	if len(req) > 0 {
		return req[0]
	}
	return nil
}
