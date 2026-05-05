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
	IsSystem    bool      `json:"isSystem" db:"is_system"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type GetRoleDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Name  string    `json:"name" db:"name"`
	Realm string    `json:"realm" db:"realm"`
}

type RoleDTO struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ActorID     uuid.UUID `json:"actorId" db:"actor_id"`
	Slug        string    `json:"slug" db:"slug"`
	Name        string    `json:"name" db:"name"`
	RealmID     string    `json:"realmId" db:"realm_id"`
	Level       int       `json:"level" db:"level"`
	IsSystem    bool      `json:"isSystem" db:"is_system"`
	Permissions []string  `json:"permissions" db:"permissions"`
	Children    []string  `json:"children" db:"children"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

type DeleteRoleDTO struct {
	ID uuid.UUID `json:"id" db:"id"`
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
