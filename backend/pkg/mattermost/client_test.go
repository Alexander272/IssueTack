package mattermost

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetAllUsers запрашивает у Mattermost только активных пользователей:
// деактивированные учётные записи не должны импортироваться в систему.
func TestClient_GetAllUsers_ActiveOnly(t *testing.T) {
	var activeParam string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeParam = r.URL.Query().Get("active")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"u1","username":"active","delete_at":0},
			{"id":"u2","username":"deactivated","delete_at":123456},
			{"id":"u3","username":"bot","is_bot":true}
		]`))
	}))
	defer server.Close()

	users, err := NewClient(server.URL).GetAllUsers("bot-token")
	require.NoError(t, err)
	assert.Equal(t, "true", activeParam, "Mattermost must return only active users")
	require.Len(t, users, 1, "bots and deactivated users must be filtered out")
	assert.Equal(t, "u1", users[0].Id)
}

// GetInactiveUsers запрашивает у Mattermost деактивированных пользователей —
// по ним синк снимает локальный is_active у связанных учётных записей.
func TestClient_GetInactiveUsers(t *testing.T) {
	var inactiveParam, activeParam string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inactiveParam = r.URL.Query().Get("inactive")
		activeParam = r.URL.Query().Get("active")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"u2","username":"deactivated","delete_at":123456}]`))
	}))
	defer server.Close()

	users, err := NewClient(server.URL).GetInactiveUsers("bot-token")
	require.NoError(t, err)
	assert.Equal(t, "true", inactiveParam)
	assert.Empty(t, activeParam, "фильтры active и inactive взаимно исключают друг друга")
	require.Len(t, users, 1)
	assert.Equal(t, "u2", users[0].Id)
}
