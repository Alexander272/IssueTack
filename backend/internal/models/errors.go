package models

import (
	"errors"
	"net/http"
)

// DomainError is a custom error type that carries HTTP response information.
// It implements the HTTPError interface defined in the response package.
type DomainError struct {
	err     error
	status  int
	code    string
	message string
}

func (e *DomainError) Error() string {
	return e.err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.err
}

func (e *DomainError) Status() int {
	return e.status
}

func (e *DomainError) Code() string {
	return e.code
}

func (e *DomainError) Message() string {
	return e.message
}

// NewDomainError creates a new DomainError.
func NewDomainError(err error, status int, code string, message string) *DomainError {
	return &DomainError{err: err, status: status, code: code, message: message}
}

var (
	// 404 Not Found
	ErrNotFound = NewDomainError(errors.New("resource not found"), http.StatusNotFound, "NF001", "Ничего не найдено")
	ErrNoRows   = NewDomainError(errors.New("row not found"), http.StatusNotFound, "NF002", "Запись не найдена")
	ErrNoData   = NewDomainError(errors.New("no data"), http.StatusNotFound, "NF003", "Данные отсутствуют")

	// 409 Conflict
	ErrAlreadyExists = NewDomainError(errors.New("resource already exists"), http.StatusConflict, "AE001", "Запись уже существует")

	// 400 Bad Request
	ErrInvalidInput          = NewDomainError(errors.New("invalid input data"), http.StatusBadRequest, "BR001", "Переданы некорректные данные")
	ErrRelatedRecordNotFound = NewDomainError(errors.New("related record not found"), http.StatusBadRequest, "BR002", "Указанный связанный ресурс не существует")
	ErrNotValid              = NewDomainError(errors.New("data is not valid"), http.StatusBadRequest, "BR003", "Данные не валидны")

	// 401 & 403
	ErrUnauthenticated  = NewDomainError(errors.New("unauthenticated"), http.StatusUnauthorized, "AU001", "Требуется авторизация")
	ErrPermissionDenied = NewDomainError(errors.New("permission denied"), http.StatusForbidden, "AU002", "Недостаточно прав для выполнения операции")
	ErrSessionEmpty     = NewDomainError(errors.New("user session not found"), http.StatusUnauthorized, "AU003", "Сессия пользователя не найдена")
	ErrSessionExpired   = NewDomainError(errors.New("session expired"), http.StatusUnauthorized, "AU004", "Время сессии истекло, повторите вход")
	ErrInvalidToken     = NewDomainError(errors.New("invalid token"), http.StatusUnauthorized, "AU005", "Токен невалиден")

	// 500 & 504
	ErrInternal         = NewDomainError(errors.New("internal server error"), http.StatusInternalServerError, "SRV01", "Внутренняя ошибка сервера")
	ErrDeadlineExceeded = NewDomainError(errors.New("deadline exceeded"), http.StatusGatewayTimeout, "SRV02", "Время ожидания операции истекло")
	ErrPolicyCheck      = NewDomainError(errors.New("policy check error"), http.StatusInternalServerError, "SRV03", "Ошибка во время проверки прав")

	// Role errors
	ErrReservedRole          = NewDomainError(errors.New("cannot create or update reserved role"), http.StatusBadRequest, "RL001", "Нельзя создать или обновить зарезервированную роль")
	ErrCannotInheritFromSelf = NewDomainError(errors.New("role cannot inherit from itself"), http.StatusBadRequest, "RL002", "Роль не может наследоваться от самой себя")
	ErrParentRoleNotFound    = NewDomainError(errors.New("parent role not found or inactive"), http.StatusNotFound, "RL003", "Указанная родительская роль не найдена")
	ErrCircularInheritance   = NewDomainError(errors.New("circular inheritance detected"), http.StatusConflict, "RL004", "Обнаружено циклическое наследование ролей")
	ErrRoleNotEditable       = NewDomainError(errors.New("role is not editable"), http.StatusBadRequest, "RL005", "Роль не редактируема")

	// Дополнительные ошибки (преобразованы в DomainError)
	ErrChangeRealm         = NewDomainError(errors.New("cannot change realm"), http.StatusForbidden, "CH001", "Невозможно изменить область")
	ErrConstraintViolation = NewDomainError(errors.New("constraint violation"), http.StatusBadRequest, "CV001", "Нарушение ограничения целостности")
	ErrInvalidPermission   = NewDomainError(errors.New("invalid permission"), http.StatusForbidden, "PE001", "Недопустимое разрешение")
	ErrFieldNotAllowed     = NewDomainError(errors.New("field is not allowed"), http.StatusForbidden, "PE002", "Поле недопустимо")
)
