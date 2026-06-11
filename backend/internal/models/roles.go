package models

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`   // Уникальный строковой идентификатор (напр. "manager")
	Name        string    `json:"name" db:"name"`   // Отображаемое имя (напр. "Начальник смены")
	Realm       string    `json:"realm" db:"realm"` // site / tenant
	Description string    `json:"description" db:"description"`
	Level       int       `json:"level" db:"level"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	IsSystem    bool      `json:"isSystem" db:"is_system"`
	IsEditable  bool      `json:"isEditable" db:"is_editable"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type RoleWithStats struct {
	Role
	Children   []string       `json:"children"`
	PermsCount PermsWithCount `json:"perms"`
	UserCount  int            `json:"userCount"`
}

type RoleShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Slug string    `json:"slug" db:"slug"`
	Name string    `json:"name" db:"name"`
}

type GetRoleDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Slug  string    `json:"slug" db:"slug"`
	Name  string    `json:"name" db:"name"`
	Realm string    `json:"realm" db:"realm"`
}

type RoleDTO struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Actor       Actor
	RealmID     uuid.UUID `json:"realmId" db:"realm_id"`
	Slug        string    `json:"slug" db:"slug"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Level       int       `json:"level" db:"level"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	IsSystem    bool      `json:"isSystem" db:"is_system"`
	IsEditable  bool      `json:"isEditable" db:"is_editable"`
	Permissions []string  `json:"permissions" db:"permissions"`
	Inherits    []string  `json:"inherits" db:"inherits"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type DeleteRoleDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Actor Actor
}

type RoleInheritance struct {
	ParentRole string
	ChildRole  string
	Realm      string
}

type RolePermission struct {
	RoleID       uuid.UUID `json:"roleId" db:"role_id"`
	PermissionID uuid.UUID `json:"permissionId" db:"permission_id"`
}

type RolePermissionDTO struct {
	ActorID      uuid.UUID `json:"actorId" db:"actor_id"`
	RoleID       uuid.UUID `json:"roleId" db:"role_id"`
	PermissionID uuid.UUID `json:"permissionId" db:"permission_id"`
}

type RoleWithPerms struct {
	Role
	Inherits []string                  `json:"inherits"`
	Perms    []*RolePermissionsGrouped `json:"perms"`
}
