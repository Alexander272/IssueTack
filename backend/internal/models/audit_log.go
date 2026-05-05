package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	ChangedBy uuid.UUID        `json:"changedBy" db:"changed_by"`
	Action    string           `json:"action" db:"action"`
	RoleID    *uuid.UUID       `json:"roleId" db:"role_id"`
	RuleID    *uuid.UUID       `json:"ruleId" db:"rule_id"`
	RealmID   *uuid.UUID       `json:"realmId" db:"realm_id"`
	UserID    *uuid.UUID       `json:"userId" db:"user_id"`
	OldValues *json.RawMessage `json:"oldValues" db:"old_values"`
	NewValues *json.RawMessage `json:"newValues" db:"new_values"`
	CreatedAt time.Time        `json:"createdAt" db:"created_at"`
}

type GetAuditLogsDTO struct{}

type GetAuditLogsByRealmDTO struct {
	RealmID uuid.UUID
}

type AuditLogDTO struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	ChangedBy uuid.UUID        `json:"changedBy" db:"changed_by"`
	Action    string           `json:"action" db:"action"`
	RoleID    *uuid.UUID       `json:"roleId" db:"role_id"`
	RuleID    *uuid.UUID       `json:"ruleId" db:"rule_id"`
	RealmID   *uuid.UUID       `json:"realmId" db:"realm_id"`
	UserID    *uuid.UUID       `json:"userId" db:"user_id"`
	OldValues *json.RawMessage `json:"oldValues" db:"old_values"`
	NewValues *json.RawMessage `json:"newValues" db:"new_values"`
}
