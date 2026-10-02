package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres/pq_models"
	"github.com/google/uuid"
)

// TestMapUsersData_Source фиксирует то, что mapUsersData читает колонку source
// в ОБОИХ ветках сборки UserData: пользователь без членства в realm и
// пользователь, у которого членство есть.
//
// Регресс: вторая ветка (у пользователя есть user_realms) собирает UserData
// отдельно, и Source в ней не проставлялся — поле оставалось пустым. Все
// пользователи, импортированные из Mattermost, состоят хотя бы в одном realm
// (SyncRealmUsers вызывает ensureRealmMembership), поэтому они попадали именно
// туда: защита синхронизации с Keycloak (source == 'mattermost') для них не
// срабатывала, и синк удалял их вместе с заявками (ON DELETE CASCADE).
func TestMapUsersData_Source(t *testing.T) {
	ns := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

	userID := uuid.New()
	created := time.Now()

	tests := []struct {
		name string
		rows []*pq_models.User
		want models.UserSource
	}{
		{
			name: "членство в realm не теряет источник",
			rows: []*pq_models.User{
				{
					Id:            userID.String(),
					Username:      "ivanov",
					CreatedAt:     created,
					UserSource:    ns(string(models.UserSourceMattermost)),
					UserRealmId:   ns(uuid.New().String()),
					RealmId:       ns(uuid.New().String()),
					RoleId:        ns(uuid.New().String()),
					RoleName:      ns("Пользователь"),
					RealmName:     ns("Основной"),
					UserIsActive:  ns2Bool(true),
					IsActive:      ns2Bool(true),
					RealmIsActive: ns2Bool(true),
				},
			},
			want: models.UserSourceMattermost,
		},
		{
			name: "несколько членств в realm не теряют источник",
			rows: []*pq_models.User{
				{
					Id:          userID.String(),
					Username:    "ivanov",
					CreatedAt:   created,
					UserSource:  ns(string(models.UserSourceMattermost)),
					UserRealmId: ns(uuid.New().String()),
				},
				{
					Id:          userID.String(),
					Username:    "ivanov",
					CreatedAt:   created,
					UserSource:  ns(string(models.UserSourceMattermost)),
					UserRealmId: ns(uuid.New().String()),
				},
			},
			want: models.UserSourceMattermost,
		},
		{
			name: "без членства в realm источник сохраняется",
			rows: []*pq_models.User{
				{
					Id:         userID.String(),
					Username:   "ivanov",
					CreatedAt:  created,
					UserSource: ns(string(models.UserSourceMattermost)),
				},
			},
			want: models.UserSourceMattermost,
		},
		{
			name: "пустой источник нормализуется в keycloak",
			rows: []*pq_models.User{
				{
					Id:         userID.String(),
					Username:   "ivanov",
					CreatedAt:  created,
					UserSource: sql.NullString{},
				},
			},
			want: models.UserSourceKeycloak,
		},
		{
			name: "неизвестный источник нормализуется в keycloak",
			rows: []*pq_models.User{
				{
					Id:         userID.String(),
					Username:   "ivanov",
					CreatedAt:  created,
					UserSource: ns("ldap"),
				},
			},
			want: models.UserSourceKeycloak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := mapUsersData(tt.rows)
			if err != nil {
				t.Fatalf("mapUsersData() error = %v", err)
			}
			if len(data) != 1 {
				t.Fatalf("mapUsersData() вернул %d пользователей, ожидался 1", len(data))
			}
			if data[0].Source != tt.want {
				t.Errorf("Source = %q, ожидалось %q", data[0].Source, tt.want)
			}
		})
	}
}

// ns2Bool — короткий конструктор sql.NullBool для тестовых строк.
func ns2Bool(v bool) sql.NullBool { return sql.NullBool{Bool: v, Valid: true} }
