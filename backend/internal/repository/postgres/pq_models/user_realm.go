package pq_models

import (
	"time"

	"github.com/google/uuid"
)

type UserRealm struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"userId" db:"user_id"`
	RealmID          uuid.UUID `json:"realmId" db:"realm_id"`
	RoleID           uuid.UUID `json:"roleId" db:"role_id"`
	IsActive         bool      `json:"isActive" db:"is_active"`
	RoleSlug         string    `json:"roleSlug" db:"role_slug"`
	RoleName         string    `json:"roleName" db:"role_name"`
	RoleLevel        int       `json:"roleLevel" db:"role_level"`
	RealmName        string    `json:"realmName" db:"realm_name"`
	RealmDescription string    `json:"realmDescription" db:"realm_description"`
	RealmCreatedAt   time.Time `json:"realmCreatedAt" db:"realm_created_at"`
}
