package models

import (
	"time"

	"github.com/google/uuid"
)

// CategoryModifier — подкатегория (модификатор категории): справочник
// параметров, которые разово применяются к заявке при выборе подкатегории.
// NULL в DefaultDueDays/Priority означает наследование значения категории.
type CategoryModifier struct {
	ID             uuid.UUID `json:"id" db:"id"`
	RealmID        uuid.UUID `json:"realmId" db:"realm_id"`
	CategoryID     uuid.UUID `json:"categoryId" db:"category_id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	DefaultDueDays *int      `json:"defaultDueDays" db:"default_due_days"`
	Priority       *Priority `json:"priority" db:"priority"`
	IsActive       bool      `json:"isActive" db:"is_active"`
	SortOrder      int       `json:"sortOrder" db:"sort_order"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

// CategoryModifierShort — подпись подкатегории в заявке.
type CategoryModifierShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

type GetCategoryModifiersDTO struct {
	RealmID    uuid.UUID  `json:"realmId" db:"realm_id"`
	CategoryID *uuid.UUID `json:"categoryId" db:"category_id"`
}

type GetCategoryModifierByIDDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type CategoryModifierDTO struct {
	ID             *uuid.UUID `json:"id" db:"id"`
	RealmID        uuid.UUID  `json:"realmId" db:"realm_id"`
	CategoryID     uuid.UUID  `json:"categoryId" db:"category_id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	DefaultDueDays *int       `json:"defaultDueDays" db:"default_due_days"`
	Priority       *Priority  `json:"priority" db:"priority"`
	IsActive       bool       `json:"isActive" db:"is_active"`
}

type DelCategoryModifierDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}
