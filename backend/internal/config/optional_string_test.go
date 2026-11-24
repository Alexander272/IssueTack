package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestOptionalString_TriState фиксирует разницу между «ключа нет» и «задано пусто»:
// первое даёт дефолт из кода, второе выключает задачу. env-default тут был бы неверен —
// cleanenv подставил бы дефолт вместо явного ”.
func TestOptionalString_TriState(t *testing.T) {
	var unset struct {
		Schedule OptionalString `yaml:"schedule"`
	}
	requireNoError(t, yaml.Unmarshal([]byte("other: 1"), &unset))
	assert.False(t, unset.Schedule.Set)
	assert.Equal(t, "@hourly", unset.Schedule.OrDefault("@hourly"))

	var empty struct {
		Schedule OptionalString `yaml:"schedule"`
	}
	requireNoError(t, yaml.Unmarshal([]byte("schedule: ''"), &empty))
	assert.True(t, empty.Schedule.Set)
	assert.Empty(t, empty.Schedule.OrDefault("@hourly"), "явное '' отключает задачу")

	var explicit struct {
		Schedule OptionalString `yaml:"schedule"`
	}
	requireNoError(t, yaml.Unmarshal([]byte("schedule: '@every 30m'"), &explicit))
	assert.Equal(t, "@every 30m", explicit.Schedule.OrDefault("@hourly"))
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
