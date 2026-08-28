package utils

import (
	"encoding/json"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bindProbeDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type bindNestedDTO struct {
	Meta bindProbeDTO `json:"meta"`
	Tags []uuid.UUID  `json:"tags"`
}

func TestDecodeJSONBody_Valid(t *testing.T) {
	var dto bindProbeDTO
	body := []byte(`{"id":"12345678-1234-1234-1234-123456789abc","name":"x"}`)
	err := decodeJSONBody(body, &dto)
	require.NoError(t, err)
	assert.Equal(t, "12345678-1234-1234-1234-123456789abc", dto.ID.String())
	assert.Equal(t, "x", dto.Name)
}

func TestDecodeJSONBody_InvalidUUIDField(t *testing.T) {
	var dto bindProbeDTO
	body := []byte(`{"id":"not-a-uuid","name":"x"}`)
	err := decodeJSONBody(body, &dto)
	require.Error(t, err)

	var inputErr *response.InputFieldError
	require.ErrorAs(t, err, &inputErr)
	assert.Equal(t, "id", inputErr.Field)
}

func TestDecodeJSONBody_InvalidUUIDNested(t *testing.T) {
	var dto bindNestedDTO
	body := []byte(`{"meta":{"id":"bad","name":"x"},"tags":[]}`)
	err := decodeJSONBody(body, &dto)
	require.Error(t, err)

	var inputErr *response.InputFieldError
	require.ErrorAs(t, err, &inputErr)
	assert.Equal(t, "meta.id", inputErr.Field)
}

func TestDecodeJSONBody_InvalidUUIDInSlice(t *testing.T) {
	var dto bindNestedDTO
	body := []byte(`{"meta":{"id":"12345678-1234-1234-1234-123456789abc","name":"x"},"tags":["abc"]}`)
	err := decodeJSONBody(body, &dto)
	require.Error(t, err)

	var inputErr *response.InputFieldError
	require.ErrorAs(t, err, &inputErr)
	assert.Equal(t, "tags[0]", inputErr.Field)
}

func TestDecodeJSONBody_TypeMismatch(t *testing.T) {
	var dto bindProbeDTO
	body := []byte(`{"id":"12345678-1234-1234-1234-123456789abc","name":123}`)
	err := decodeJSONBody(body, &dto)
	require.Error(t, err)

	// Ошибка типа распознаётся штатным декодером — поле указывается по json-тегу,
	// reflect-обход для неё не запускается.
	var unmarshalErr *json.UnmarshalTypeError
	require.ErrorAs(t, err, &unmarshalErr)
	assert.Equal(t, "name", unmarshalErr.Field)
}

func TestDecodeJSONBody_SyntaxError(t *testing.T) {
	var dto bindProbeDTO
	body := []byte(`{"id":`)
	err := decodeJSONBody(body, &dto)
	require.Error(t, err)

	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}
