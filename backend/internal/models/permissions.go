package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Role        string    `json:"role" db:"role"`
	Realm       string    `json:"realm" db:"realm"`
	Object      string    `json:"object" db:"object"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type GroupedPermission struct {
	Group string        `json:"group"`
	Title string        `json:"title"`
	Items []*Permission `json:"items"`
}

type GetPermsByRoleDTO struct {
	Role  string `json:"role" db:"role"`
	Realm string `json:"realm" db:"realm"`
}

type PermissionDTO struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Object      string    `json:"object" db:"object"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
}

type DeletePermissionDTO struct {
	ActorID uuid.UUID `json:"actorId" db:"actor_id"`
	ID      uuid.UUID `json:"id" db:"id"`
}

type Grouping struct {
	ID      string `json:"id" db:"id"`
	Subject string `json:"subject" db:"subject"`
	Role    string `json:"role" db:"role"`
	Domain  string `json:"domain" db:"domain"`
}

type PermsWithCount struct {
	Own       Perm `json:"own"`
	Inherited Perm `json:"inherited"`
	Total     Perm `json:"total"`
}
type Perm struct {
	Items []string `json:"items"`
	Count int      `json:"count"`
}

type GetPermsCountDTO struct {
	Role      string
	Inherited []string
}

type RolePermissionItem struct {
	PermissionID uuid.UUID `json:"permissionId" db:"permission_id"`
	Object       string    `json:"object" db:"object"`
	Action       string    `json:"action" db:"action"`
	IsAssigned   bool      `json:"isAssigned"`
	IsInherited  bool      `json:"isInherited"`
}

type RolePermissionsGrouped struct {
	Group     string                `json:"group"`
	Title     string                `json:"title"`
	Resources []*RolePermissionItem `json:"resources"`
}
