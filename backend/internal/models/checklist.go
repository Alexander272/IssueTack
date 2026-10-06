package models

import (
	"time"

	"github.com/google/uuid"
)

type ChecklistTemplate struct {
	ID          uuid.UUID                `json:"id" db:"id"`
	RealmID     uuid.UUID                `json:"realmId" db:"realm_id"`
	Title       string                   `json:"title" db:"title"`
	Description string                   `json:"description" db:"description"`
	CreatedBy   *uuid.UUID               `json:"createdBy,omitempty" db:"created_by"`
	Items       []*ChecklistTemplateItem `json:"items,omitempty"`
	CreatedAt   time.Time                `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time                `json:"updatedAt" db:"updated_at"`
}

type ChecklistTemplateItem struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TemplateID  uuid.UUID `json:"templateId" db:"template_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	SortOrder   int       `json:"sortOrder" db:"sort_order"`
}

type GetChecklistTemplateDTO struct {
	ID uuid.UUID `json:"id"`
	// RealmID — реалм, под которым запрос авторизован (заполняет хендлер):
	// шаблон чужого реалма по «своему realm + чужой uuid» иначе читался и правился.
	RealmID uuid.UUID `json:"-"`
}

type GetChecklistTemplatesDTO struct {
	RealmID uuid.UUID `json:"realmId"`
	// Actor — кто запрашивает (заполняется хендлером). Если у пользователя нет
	// права checklist:read, сервис вернёт только его собственные шаблоны.
	Actor *Actor `json:"-"`
	// OwnerID — фильтр «только мои» (устанавливает сервис).
	OwnerID *uuid.UUID `json:"-"`
}

type ChecklistTemplateDTO struct {
	// RealmID подставляется сервером из контекста (см. GetChecklistTemplateDTO.RealmID).
	RealmID     uuid.UUID `json:"-"`
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	// CreatedBy проставляется сервером (модель владения шаблоном).
	CreatedBy *uuid.UUID `json:"-"`
	// Actor заполняется хендлером из контекста.
	Actor *Actor `json:"-"`
}

type ChecklistTemplateItemDTO struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description" db:"description"`
	SortOrder   int       `json:"sortOrder"`
}

type DelChecklistTemplateDTO struct {
	ID      uuid.UUID `json:"id"`
	RealmID uuid.UUID `json:"-"`
}

type ApplyTemplateDTO struct {
	TicketID   uuid.UUID
	TemplateID uuid.UUID
	Actor      *Actor
	// RealmID — авторизованный реалм (из контекста): шаблон должен быть из него же,
	// иначе к тикету применялся бы чек-лист чужой области.
	RealmID uuid.UUID
}
