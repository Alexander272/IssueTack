package events

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type PolicyUpdateListener chan PolicyEvent

type PolicyEventManager struct {
	mu        sync.Mutex
	listeners []PolicyUpdateListener
}

type PolicyEvent struct {
	ChangedBy     uuid.UUID       `json:"changedBy" db:"changed_by"`
	ChangedByName string          `json:"changedByName" db:"changed_by_name"`
	Action        string          `json:"action" db:"action"`
	EntityType    string          `json:"entityType" db:"entity_type"`
	Entity        *string         `json:"entity"` // Название объекта (напр. "john@email.com", "Администратор")
	EntityID      *uuid.UUID      `json:"entityId" db:"entity_id"`
	RealmID       *uuid.UUID      `json:"realmId"`
	RealmName     string          `json:"realmName"`
	OldValues     json.RawMessage `json:"oldValues" db:"old_values"`
	NewValues     json.RawMessage `json:"newValues" db:"new_values"`
}

func (m *PolicyEventManager) Subscribe() PolicyUpdateListener {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(PolicyUpdateListener, 1)
	m.listeners = append(m.listeners, ch)
	return ch
}

func (m *PolicyEventManager) Notify(event PolicyEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ch := range m.listeners {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *PolicyEventManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = nil
}
