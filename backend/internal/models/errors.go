package models

import "errors"

var (
	// Базовые ошибки ресурсов (404, 409, 400)
	ErrNotFound      = errors.New("resource not found")      // Запись не найдена
	ErrAlreadyExists = errors.New("resource already exists") // Нарушение уникальности (email, login)
	ErrInvalidInput  = errors.New("invalid input data")      // Ошибка валидации или плохой формат

	// Ошибки доступа (401, 403)
	ErrUnauthenticated  = errors.New("unauthenticated")   // Не авторизован (нет токена)
	ErrPermissionDenied = errors.New("permission denied") // Нет прав (RBAC/ACL)

	// Ошибки связей (400 или 409)
	ErrRelatedRecordNotFound = errors.New("related record not found") // Ссылка на несуществующий ID (Foreign Key)
	ErrConstraintViolation   = errors.New("constraint violation")     // Общее нарушение логики БД

	// Системные ошибки (500, 503)
	ErrInternal         = errors.New("internal server error") // Непредвиденная ошибка
	ErrDeadlineExceeded = errors.New("deadline exceeded")     // Таймаут операции

	ErrReservedRole          = errors.New("cannot create or update reserved role")
	ErrChangeRealm           = errors.New("cannot change realm")
	ErrCircularInheritance   = errors.New("circular inheritance detected")
	ErrCannotInheritFromSelf = errors.New("role cannot inherit from itself")
	ErrParentRoleNotFound    = errors.New("parent role not found or inactive")

	ErrInvalidPermission = errors.New("invalid permission")
)
