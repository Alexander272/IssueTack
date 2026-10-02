package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// TestProtectedFromKeycloakSync фиксирует главное правило синхронизации:
// пользователя из Mattermost нельзя удалять, даже если его нет в Keycloak.
// tickets.creator_id объявлен как ON DELETE CASCADE, поэтому ошибка здесь
// означала бы безвозвратную потерю заявок заявителей.
func TestProtectedFromKeycloakSync(t *testing.T) {
	tests := []struct {
		name      string
		user      *models.UserData
		protected bool
	}{
		{
			name:      "пользователь из Mattermost защищён",
			user:      &models.UserData{ID: uuid.New(), Source: models.UserSourceMattermost},
			protected: true,
		},
		{
			name:      "системный пользователь защищён",
			user:      &models.UserData{ID: uuid.New(), IsSystem: true, Source: models.UserSourceKeycloak},
			protected: true,
		},
		{
			name:      "сотрудник из Keycloak удаляется при уходе",
			user:      &models.UserData{ID: uuid.New(), Source: models.UserSourceKeycloak},
			protected: false,
		},
		{
			name: "пустой источник считается keycloak и не защищает",
			// Нормализация пустого значения живёт в репозитории; на случай
			// непрочитанного источника считаем пользователя обычным.
			user:      &models.UserData{ID: uuid.New()},
			protected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := protectedFromKeycloakSync(tt.user); got != tt.protected {
				t.Errorf("protectedFromKeycloakSync() = %v, want %v", got, tt.protected)
			}
		})
	}
}

// TestHasUnknownSource фиксирует страховку от повторения массового удаления:
// пользователь, у которого источник не прочитан, удалять нельзя. Пустой Source
// возможен только при потере поля в маппинге (например, в ветке mapUsersData для
// пользователя с членством в realm) — именно это молча делало всех
// импортированных из Mattermost пользователей удаляемыми.
func TestHasUnknownSource(t *testing.T) {
	tests := []struct {
		name    string
		user    *models.UserData
		unknown bool
	}{
		{
			name:    "пустой источник — неизвестен",
			user:    &models.UserData{ID: uuid.New()},
			unknown: true,
		},
		{
			name:    "источник mattermost известен",
			user:    &models.UserData{ID: uuid.New(), Source: models.UserSourceMattermost},
			unknown: false,
		},
		{
			name:    "источник keycloak известен",
			user:    &models.UserData{ID: uuid.New(), Source: models.UserSourceKeycloak},
			unknown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasUnknownSource(tt.user); got != tt.unknown {
				t.Errorf("hasUnknownSource() = %v, want %v", got, tt.unknown)
			}
		})
	}
}

func TestUserSourceIsFromMattermost(t *testing.T) {
	if models.UserSourceKeycloak.IsFromMattermost() {
		t.Error("keycloak не должен считаться источником mattermost")
	}
	if !models.UserSourceMattermost.IsFromMattermost() {
		t.Error("mattermost должен определяться как источник mattermost")
	}
}

// TestErrKeycloakSyncLinkedUsers проверяет, что ошибка остановки синхронизации
// доходит до клиента с кодом SY001, называет пользователей и предлагает осознанный
// обход: клиент должен видеть, что делать, а не просто «ошибка».
func TestErrKeycloakSyncLinkedUsers(t *testing.T) {
	err := models.ErrKeycloakSyncLinkedUsers([]string{"ivanov", "petrov"}, 2)

	if got := err.Code(); got != "SY001" {
		t.Errorf("Code() = %q, want SY001", got)
	}
	for _, want := range []string{"ivanov", "petrov", "всего: 2", "force=1"} {
		if !strings.Contains(err.Message(), want) {
			t.Errorf("Message() = %q, должен содержать %q", err.Message(), want)
		}
	}
}

// TestErrKeycloakSyncLinkedUsers_TruncatesList: длинный список имён в сообщение
// целиком не влезает — показываем первые 10 и общее количество, иначе ответ
// превращается в простыню.
func TestErrKeycloakSyncLinkedUsers_TruncatesList(t *testing.T) {
	usernames := make([]string, 0, 12)
	for i := range 12 {
		usernames = append(usernames, fmt.Sprintf("user%d", i))
	}

	err := models.ErrKeycloakSyncLinkedUsers(usernames, len(usernames))

	if !strings.Contains(err.Message(), "user9") {
		t.Errorf("Message() = %q, должен содержать десятое имя", err.Message())
	}
	if strings.Contains(err.Message(), "user10") {
		t.Errorf("Message() = %q, не должен содержать оставшиеся имена", err.Message())
	}
	if !strings.Contains(err.Message(), "всего: 12") {
		t.Errorf("Message() = %q, должен содержать общее количество", err.Message())
	}
}
