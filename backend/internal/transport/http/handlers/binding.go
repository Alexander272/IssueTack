package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin/binding"
)

func init() {
	binding.JSON = fieldAwareBinding{}
}

type fieldAwareBinding struct{}

func (fieldAwareBinding) Name() string { return "json" }

func (b fieldAwareBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return decodeJSONBody(body, obj)
}

func (fieldAwareBinding) BindBody(body []byte, obj any) error {
	return decodeJSONBody(body, obj)
}

func decodeJSONBody(data []byte, obj any) error {
	if err := json.Unmarshal(data, obj); err != nil {
		if isStandardJSONError(err) {
			return err
		}
		if fieldErr := inspectValue("", data, reflect.TypeOf(obj)); fieldErr != nil {
			return fieldErr
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

// inspectValue recursively decodes data into typ and finds the exact field
// that caused a custom unmarshaler error (e.g. uuid.UUID).
func inspectValue(path string, data []byte, typ reflect.Type) error {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	testVal := reflect.New(typ)
	err := json.Unmarshal(data, testVal.Interface())
	if err == nil {
		return nil
	}
	if isStandardJSONError(err) {
		return nil
	}

	hasCustomUnmarshaler := reflect.PointerTo(typ).Implements(reflect.TypeOf((*json.Unmarshaler)(nil)).Elem())

	if !hasCustomUnmarshaler {
		switch typ.Kind() {
		case reflect.Struct:
			return inspectStruct(path, data, typ)
		case reflect.Slice:
			return inspectSlice(path, data, typ)
		}
	}

	if path == "" {
		return err
	}
	return &response.InputFieldError{Field: path, Err: err}
}

func inspectStruct(path string, data []byte, typ reflect.Type) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil
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

		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}

		if err := inspectValue(fieldPath, raw, f.Type); err != nil {
			return err
		}
	}
	return nil
}

func inspectSlice(path string, data []byte, typ reflect.Type) error {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}

	elemType := typ.Elem()
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, idx)
		if err := inspectValue(itemPath, item, elemType); err != nil {
			return err
		}
	}
	return nil
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

func isStandardJSONError(err error) bool {
	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &unmarshalTypeErr)
}
