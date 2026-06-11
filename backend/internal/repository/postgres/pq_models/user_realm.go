package pq_models

import "time"

type UserRealm struct {
	Id               string    `json:"id" db:"id"`
	UserId           string    `json:"userId" db:"user_id"`
	RealmId          string    `json:"realmId" db:"realm_id"`
	RoleId           string    `json:"roleId" db:"role_id"`
	IsActive         bool      `json:"isActive" db:"is_active"`
	RoleSlug         string    `json:"roleSlug" db:"role_slug"`
	RoleName         string    `json:"roleName" db:"role_name"`
	RoleLevel        int       `json:"roleLevel" db:"role_level"`
	RealmName        string    `json:"realmName" db:"realm_name"`
	RealmDescription string    `json:"realmDescription" db:"realm_description"`
	RealmCreatedAt   time.Time `json:"realmCreatedAt" db:"realm_created_at"`
}
