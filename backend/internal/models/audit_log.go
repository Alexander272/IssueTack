package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	ChangedBy     uuid.UUID        `json:"changedBy" db:"changed_by"`
	ChangedByName string           `json:"changedByName" db:"changed_by_name"`
	Action        string           `json:"action" db:"action"`
	EntityType    string           `json:"entityType" db:"entity_type"`
	EntityID      *uuid.UUID       `json:"entityId" db:"entity_id"`
	RealmID       *uuid.UUID       `json:"realmId" db:"realm_id"`
	RealmName     string           `json:"realmName" db:"realm_name"`
	OldValues     *json.RawMessage `json:"oldValues" db:"old_values"`
	NewValues     *json.RawMessage `json:"newValues" db:"new_values"`
	CreatedAt     time.Time        `json:"createdAt" db:"created_at"`
}

type GetAuditLogsDTO struct{}

type GetAuditLogsByRealmDTO struct {
	RealmID uuid.UUID
}

type AuditLogDTO struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	ChangedBy     uuid.UUID       `json:"changedBy" db:"changed_by"`
	ChangedByName string          `json:"changedByName" db:"changed_by_name"`
	Action        string          `json:"action" db:"action"`
	EntityType    string          `json:"entityType" db:"entity_type"`
	Entity        *string         `json:"entity" db:"entity"`
	EntityID      *uuid.UUID      `json:"entityId" db:"entity_id"`
	RealmID       *uuid.UUID      `json:"realmId"`
	RealmName     string          `json:"realmName"`
	OldValues     json.RawMessage `json:"oldValues" db:"old_values"`
	NewValues     json.RawMessage `json:"newValues" db:"new_values"`
}
