package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID              uuid.UUID           `json:"id" db:"id"`
	Name            string              `json:"name" db:"name"`
	Description     string              `json:"description" db:"description"`
	GroupID         uuid.UUID           `json:"groupId" db:"group_id"` // Группа, которая "владеет" этой категорией
	CategoryGroupID *uuid.UUID          `json:"categoryGroupId" db:"category_group_id"`
	CategoryGroup   *CategoryGroupShort `json:"categoryGroup,omitempty" db:"-"`
	Priority        Priority            `json:"priority" db:"def_priority"`
	IsActive        bool                `json:"isActive" db:"is_active"`
	// IsHidden — собственная скрытость категории (не эффективная): категория
	// в скрытом разделе тоже невидима заказчику, но её флаг остаётся false.
	IsHidden bool `json:"isHidden" db:"is_hidden"`
	// SortOrder — глобальная позиция категории в реалме (1..N). Порядок
	// разделов выводится из него: блок раздела стоит там, где стоит его
	// первая категория. Меняется только через переупорядочивание
	// (см. ReorderCategoriesDTO).
	SortOrder      int       `json:"sortOrder" db:"sort_order"`
	RealmID        uuid.UUID `json:"realmId" db:"realm_id"`
	DefaultDueDays *int      `json:"defaultDueDays" db:"default_due_days"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type CategoryShort struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	CategoryGroupName *string   `json:"categoryGroupName,omitempty" db:"category_group_name"`
}

type GetCategoriesDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
	// IncludeHidden — вернуть и скрытые категории (в т.ч. из скрытых разделов).
	// Явный флаг, а не роль: решение о роли принимает вызывающий (менеджеры и
	// исполнители видят скрытые, безгрупповой заказчик — нет).
	IncludeHidden bool `json:"-" db:"-"`
}

type GetCategoryByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type CategoryDTO struct {
	ID              *uuid.UUID `json:"id" db:"id"`
	RealmID         uuid.UUID  `json:"realmId" db:"realm_id"`
	Name            string     `json:"name" db:"name"`
	Description     string     `json:"description" db:"description"`
	GroupID         uuid.UUID  `json:"groupId" db:"group_id"`
	CategoryGroupID *uuid.UUID `json:"categoryGroupId" db:"category_group_id"`
	Priority        Priority   `json:"priority" db:"def_priority"`
	DefaultDueDays  *int       `json:"defaultDueDays" db:"default_due_days"`
	IsActive        bool       `json:"isActive" db:"is_active"`
	IsHidden        bool       `json:"isHidden" db:"is_hidden"`
}

type DelCategoryDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// ReorderCategoriesDTO — новый глобальный порядок категорий реалма.
// CategoryIDs — полный порядок: id категорий от первой к последней.
type ReorderCategoriesDTO struct {
	RealmID uuid.UUID `json:"-" db:"-"`
	// CategoryIDs — id категорий в новом порядке. Категории, которых в
	// списке нет, дописываются следом в текущем порядке (сервис).
	CategoryIDs []uuid.UUID `json:"categoryIds" db:"-"`
}
