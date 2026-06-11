package models

import (
	"time"

	"github.com/google/uuid"
)

type ActivityLog struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Action        string     `json:"action" db:"action"`   // 'INSERT', 'UPDATE', 'DELETE'
	ChangedBy     uuid.UUID  `json:"changedBy" db:"changed_by"`
	ChangedByName string     `json:"changedByName" db:"changed_by_name"`
	EntityType    string     `json:"entityType" db:"entity_type"`
	EntityID      uuid.UUID  `json:"entityId" db:"entity_id"`
	Entity        string     `json:"entity" db:"entity"` // отображаемое имя
	ParentID      *uuid.UUID `json:"parentId" db:"parent_id"`
	RealmID       *uuid.UUID `json:"realmId" db:"realm_id"`
	RealmName     string     `json:"realmName" db:"realm_name"`
	OldValue      *string    `json:"oldValue" db:"old_value"`
	NewValue      *string    `json:"newValue" db:"new_value"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
}

type GetLogsDTO struct {
	EntityID   *uuid.UUID `json:"entityId"`
	EntityType *string    `json:"entityType"`
	ParentID   *uuid.UUID `json:"parentId"`
	RealmID    *uuid.UUID `json:"realmId"`
}

type ActivityLogDTO struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Action        string     `json:"action" db:"action"`
	ChangedBy     uuid.UUID  `json:"changedBy" db:"changed_by"`
	ChangedByName string     `json:"changedByName" db:"changed_by_name"`
	EntityType    string     `json:"entityType" db:"entity_type"`
	EntityID      uuid.UUID  `json:"entityId" db:"entity_id"`
	Entity        string     `json:"entity" db:"entity"`
	ParentID      *uuid.UUID `json:"parentId" db:"parent_id"`
	RealmID       *uuid.UUID `json:"realmId" db:"realm_id"`
	RealmName     string     `json:"realmName" db:"realm_name"`
	OldValue      *string    `json:"oldValue" db:"old_value"`
	NewValue      *string    `json:"newValue" db:"new_value"`
}
