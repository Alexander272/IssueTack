package models

import "github.com/google/uuid"

type RoleHierarchy struct {
	Role       Role `json:"childRole"`
	ParentRole Role `json:"parentRole"`
	// ParentRoleID uuid.UUID `json:"parentRoleId" db:"parent_role_id"`
	// ChildRoleID  uuid.UUID `json:"childRoleId" db:"child_role_id"`
}

type RoleWithHierarchy struct {
	Role
	InheritsFrom []*RoleHierarchy `json:"inherits_from"` // от кого наследуем
	InheritedBy  []*RoleHierarchy `json:"inherited_by"`  // кто наследует от нас
}

type RoleHierarchyDTO struct {
	ParentRoleID uuid.UUID `json:"parentRoleId" db:"parent_role_id"`
	RoleID       uuid.UUID `json:"childRoleId" db:"child_role_id"`
	RealmID      uuid.UUID `json:"realmId" db:"realm_id"`
	Actor        *Actor
}

type GetRoleInheritance struct {
	Role  string
	Realm string
}

type GetRolesInheritance struct {
	Roles []string
	Realm string
}

type SyncRoleInheritance struct {
	Role       string
	ParentRole string
	Realm      string
}
