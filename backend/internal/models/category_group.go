package models

import (
	"time"

	"github.com/google/uuid"
)

// CategoryGroup — раздел категорий (таксономия). Не путать с группой-владельцем
// категории (categories.group_id): раздел только группирует категории в списках
// выбора, на маршрутизацию и права не влияет.
type CategoryGroup struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RealmID     uuid.UUID `json:"realmId" db:"realm_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	SortOrder   int       `json:"sortOrder" db:"sort_order"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type CategoryGroupShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}

type CategoryGroupDTO struct {
	ID          *uuid.UUID `json:"id" db:"id"`
	RealmID     uuid.UUID  `json:"realmId" db:"realm_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	SortOrder   int        `json:"sortOrder" db:"sort_order"`
}

type GetCategoryGroupsDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type GetCategoryGroupByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type DelCategoryGroupDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}
