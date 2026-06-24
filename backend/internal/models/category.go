package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	GroupID     uuid.UUID `json:"groupId" db:"group_id"` // Группа, которая "владеет" этой категорией
	Priority    Priority  `json:"priority" db:"def_priority"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	RealmID     uuid.UUID `json:"realmId" db:"realm_id"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type CategoryShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

type GetCategoriesDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type GetCategoryByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type CategoryDTO struct {
	ID          *uuid.UUID `json:"id" db:"id"`
	RealmID     uuid.UUID  `json:"realmId" db:"realm_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	GroupID     uuid.UUID  `json:"groupId" db:"group_id"`
	Priority    Priority   `json:"priority" db:"def_priority"`
	IsActive    bool       `json:"isActive" db:"is_active"`
}

type DelCategoryDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}
