package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ActivityLog struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	Action        string           `json:"action" db:"action"`   // 'INSERT', 'UPDATE', 'DELETE'
	ChangedBy     uuid.UUID        `json:"changedBy" db:"changed_by"`
	ChangedByName string           `json:"changedByName" db:"changed_by_name"`
	EntityType    string           `json:"entityType" db:"entity_type"`
	EntityID      uuid.UUID        `json:"entityId" db:"entity_id"`
	Entity        string           `json:"entity" db:"entity"` // отображаемое имя
	ParentID      *uuid.UUID       `json:"parentId" db:"parent_id"`
	RealmID       *uuid.UUID       `json:"realmId" db:"realm_id"`
	RealmName     string           `json:"realmName" db:"realm_name"`
	OldValues     *json.RawMessage `json:"oldValues" db:"old_value"`
	NewValues     *json.RawMessage `json:"newValues" db:"new_value"`
	CreatedAt     time.Time        `json:"createdAt" db:"created_at"`
}

type GetLogsDTO struct {
	EntityID   *uuid.UUID `json:"entityId"`
	EntityType *string    `json:"entityType"`
	ParentID   *uuid.UUID `json:"parentId"`
	RealmID    *uuid.UUID `json:"realmId"`
}

type ActivityLogDTO struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	Action        string           `json:"action" db:"action"`
	ChangedBy     uuid.UUID        `json:"changedBy" db:"changed_by"`
	ChangedByName string           `json:"changedByName" db:"changed_by_name"`
	EntityType    string           `json:"entityType" db:"entity_type"`
	EntityID      uuid.UUID        `json:"entityId" db:"entity_id"`
	Entity        string           `json:"entity" db:"entity"`
	ParentID      *uuid.UUID       `json:"parentId" db:"parent_id"`
	RealmID       *uuid.UUID       `json:"realmId" db:"realm_id"`
	RealmName     string           `json:"realmName" db:"realm_name"`
	OldValues     *json.RawMessage `json:"oldValues" db:"old_value"`
	NewValues     *json.RawMessage `json:"newValues" db:"new_value"`
}

func (l *ActivityLogDTO) SetOldValues(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal old values: %w", err)
	}
	l.OldValues = (*json.RawMessage)(&data)
	return nil
}

func (l *ActivityLogDTO) SetNewValues(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal new values: %w", err)
	}
	l.NewValues = (*json.RawMessage)(&data)
	return nil
}
