package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Role      string    `json:"role" db:"role"`
	Realm     string    `json:"realm" db:"realm"`
	Object    string    `json:"object" db:"object"`
	Action    string    `json:"action" db:"action"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type GetPermsByRoleDTO struct {
	Role  string `json:"role" db:"role"`
	Realm string `json:"realm" db:"realm"`
}

type PermissionDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RoleID  uuid.UUID `json:"roleId" db:"role_id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
	Object  string    `json:"object" db:"object"`
	Action  string    `json:"action" db:"action"`
}

type DeletePermissionDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
}

type Grouping struct {
	ID      string `json:"id" db:"id"`
	Subject string `json:"subject" db:"subject"`
	Role    string `json:"role" db:"role"`
	Domain  string `json:"domain" db:"domain"`
}
