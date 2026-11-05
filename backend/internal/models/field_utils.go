package models

import (
	"fmt"
	"reflect"
	"time"
)

// formatFieldValue приводит значение поля к строке для журнала/уведомлений.
// nil и nil-указатели — "none" (во фронте это «—»), время — RFC3339, чтобы
// потребитель мог его разобрать и отформатировать; иначе %v.
func formatFieldValue(v any) string {
	if v == nil {
		return "none"
	}

	switch t := v.(type) {
	case time.Time:
		return t.Format(time.RFC3339)
	case *time.Time:
		if t == nil {
			return "none"
		}
		return t.Format(time.RFC3339)
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		if rv.IsNil() {
			return "none"
		}
	}

	return fmt.Sprintf("%v", v)
}
