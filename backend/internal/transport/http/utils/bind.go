package utils

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/goccy/go-json"
)

// jsonUnmarshalerType используется для проверки, реализует ли тип свой
// десериализатор (например uuid.UUID) — внутрь такого типа не заглядываем.
var jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

// BindJSON связывает JSON-тело запроса с dto. В отличие от штатного
// c.ShouldBindJSON, для ошибок десериализации он дополнительно определяет имя
// поля, из-за которого произошла ошибка, и возвращает *response.InputFieldError.
//
// Штатный декодер (goccy/go-json) сообщает имя поля сам, но, в отличие от
// encoding/json, это имя Go-поля («Name»), а не ключ запроса («name»), и для
// вложенных значений теряется путь. Поэтому точный путь по json-тегам
// восстанавливается рекурсивным обходом структуры (см. findField) — для ошибок
// типа и для кастомных десериализаторов. Обход выполняется только на ошибке,
// поэтому на счастливом пути накладных расходов нет.
func BindJSON(c *gin.Context, obj any) error {
	if c.Request == nil || c.Request.Body == nil {
		return errors.New("invalid request")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	return decodeJSONBody(body, obj)
}

func decodeJSONBody(data []byte, obj any) error {
	typ := reflect.TypeOf(obj)
	if err := json.Unmarshal(data, obj); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return localizeTypeError(err, typeErr, data, typ)
		}
		if isStandardJSONError(err) {
			return err
		}
		if path, fieldErr := findField("", data, typ, isCustomUnmarshalError); path != "" {
			return &response.InputFieldError{Field: path, Err: fieldErr}
		}
		return err
	}
	return validate(obj)
}

func validate(obj any) error {
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}

// localizeTypeError заменяет имя Go-поля, которое goccy указывает в ошибке типа,
// на путь по json-тегам. Сам тип ошибки сохраняется (в response.SendError он
// разбирается как BR003), меняется только поле. Если путь найти не удалось,
// ошибка возвращается как есть.
func localizeTypeError(err error, typeErr *json.UnmarshalTypeError, data []byte, typ reflect.Type) error {
	path, _ := findField("", data, typ, isTypeError)
	if path == "" || path == typeErr.Field {
		return err
	}
	fixed := *typeErr
	fixed.Field = path
	return &fixed
}

// findField рекурсивно декодирует data в typ и возвращает путь по json-тегам до
// первого поля, ошибка декодирования которого принята match, вместе с этой
// ошибкой. Вложенные структуры и срезы раскрываются, поэтому поле внутри них
// адресуется полным путём («meta.id», «tags[0]»). Если подходящего поля нет —
// возвращается пустой путь.
func findField(path string, data []byte, typ reflect.Type, match func(error) bool) (string, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	probe := reflect.New(typ)
	err := json.Unmarshal(data, probe.Interface())
	if err == nil || !match(err) {
		return "", nil
	}

	if !reflect.PointerTo(typ).Implements(jsonUnmarshalerType) {
		switch typ.Kind() {
		case reflect.Struct:
			if fieldPath, fieldErr := findFieldInStruct(path, data, typ, match); fieldPath != "" {
				return fieldPath, fieldErr
			}
		case reflect.Slice:
			if fieldPath, fieldErr := findFieldInSlice(path, data, typ, match); fieldPath != "" {
				return fieldPath, fieldErr
			}
		}
	}

	return path, err
}

func findFieldInStruct(path string, data []byte, typ reflect.Type, match func(error) bool) (string, error) {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return "", nil
	}

	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		name := jsonFieldName(f)
		if name == "" || name == "-" {
			continue
		}
		raw, ok := rawMap[name]
		if !ok {
			continue
		}

		if fieldPath, fieldErr := findField(joinFieldPath(path, name), raw, f.Type, match); fieldPath != "" {
			return fieldPath, fieldErr
		}
	}
	return "", nil
}

func findFieldInSlice(path string, data []byte, typ reflect.Type, match func(error) bool) (string, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return "", nil
	}

	elemType := typ.Elem()
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		if fieldPath, fieldErr := findField(itemPath, item, elemType, match); fieldPath != "" {
			return fieldPath, fieldErr
		}
	}
	return "", nil
}

func joinFieldPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return f.Name
	}
	return name
}

func isTypeError(err error) bool {
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &typeErr)
}

func isCustomUnmarshalError(err error) bool {
	return !isStandardJSONError(err)
}

func isStandardJSONError(err error) bool {
	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &unmarshalTypeErr)
}
