package models

import "github.com/go-playground/validator/v10"

// Интерфейс для всех enum, которые умеют себя проверять
type ValidateEnum interface {
	IsValid() bool
}

// UniversalEnumValidator проверяет любой тип, реализующий ValidateEnum
func UniversalEnumValidator(fl validator.FieldLevel) bool {
	// Получаем значение поля как интерфейс
	field := fl.Field().Interface()

	// Проверяем, реализует ли тип наш интерфейс
	if enum, ok := field.(ValidateEnum); ok {
		return enum.IsValid()
	}

	return false
}
