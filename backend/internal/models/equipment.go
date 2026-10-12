package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EquipmentStatus — статус единицы оборудования.
type EquipmentStatus string

const (
	EquipmentStatusInStock    EquipmentStatus = "in_stock"
	EquipmentStatusInUse      EquipmentStatus = "in_use"
	EquipmentStatusRepair     EquipmentStatus = "repair"
	EquipmentStatusReserved   EquipmentStatus = "reserved"
	EquipmentStatusWrittenOff EquipmentStatus = "written_off"
)

// IsValid проверяет, что статус принадлежит перечислению (для enum-валидатора).
func (s EquipmentStatus) IsValid() bool {
	switch s {
	case EquipmentStatusInStock, EquipmentStatusInUse, EquipmentStatusRepair,
		EquipmentStatusReserved, EquipmentStatusWrittenOff:
		return true
	}
	return false
}

// EquipmentEventType — тип события журнала оборудования.
type EquipmentEventType string

const (
	EquipmentEventCreated           EquipmentEventType = "created"
	EquipmentEventUpdated           EquipmentEventType = "updated"
	EquipmentEventAssigned          EquipmentEventType = "assigned"
	EquipmentEventUnassigned        EquipmentEventType = "unassigned"
	EquipmentEventMoved             EquipmentEventType = "moved"
	EquipmentEventStatusChanged     EquipmentEventType = "status_changed"
	EquipmentEventComponentAdded    EquipmentEventType = "component_added"
	EquipmentEventComponentRemoved  EquipmentEventType = "component_removed"
	EquipmentEventComponentReplaced EquipmentEventType = "component_replaced"
	EquipmentEventRepaired          EquipmentEventType = "repaired"
	EquipmentEventWrittenOff        EquipmentEventType = "written_off"
)

// IsValid проверяет, что тип события принадлежит перечислению (для enum-валидатора).
func (e EquipmentEventType) IsValid() bool {
	switch e {
	case EquipmentEventCreated, EquipmentEventUpdated, EquipmentEventAssigned, EquipmentEventUnassigned,
		EquipmentEventMoved, EquipmentEventStatusChanged, EquipmentEventComponentAdded,
		EquipmentEventComponentRemoved, EquipmentEventComponentReplaced, EquipmentEventRepaired,
		EquipmentEventWrittenOff:
		return true
	}
	return false
}

// EquipmentType — тип оборудования (справочник реалма): ноутбук, RAM, монитор и т.д.
type EquipmentType struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RealmID     uuid.UUID `json:"realmId" db:"realm_id"`
	Name        string    `json:"name" db:"name"`
	IsComponent bool      `json:"isComponent" db:"is_component"`
	SortOrder   int       `json:"sortOrder" db:"sort_order"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type GetEquipmentTypesDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type GetEquipmentTypeByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type EquipmentTypeDTO struct {
	ID          *uuid.UUID `json:"id" db:"id"`
	RealmID     uuid.UUID  `json:"realmId" db:"realm_id"`
	Name        string     `json:"name" validate:"required" db:"name"`
	IsComponent bool       `json:"isComponent" db:"is_component"`
	SortOrder   int        `json:"sortOrder" db:"sort_order"`
}

type DelEquipmentTypeDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// Equipment — единица оборудования (или компонент комплекта).
type Equipment struct {
	ID              uuid.UUID        `json:"id" db:"id"`
	RealmID         uuid.UUID        `json:"realmId" db:"realm_id"`
	TypeID          uuid.UUID        `json:"typeId" db:"type_id"`
	TypeName        string           `json:"typeName" db:"type_name"`
	IsComponent     bool             `json:"isComponent" db:"is_component"`
	Name            string           `json:"name" db:"name"`
	Vendor          string           `json:"vendor" db:"vendor"`
	Model           string           `json:"model" db:"model"`
	SerialNumber    *string          `json:"serialNumber" db:"serial_number"`
	InventoryNumber *string          `json:"inventoryNumber" db:"inventory_number"`
	Specs           json.RawMessage  `json:"specs" db:"specs"`
	Status          EquipmentStatus  `json:"status" db:"status"`
	ParentID        *uuid.UUID       `json:"parentId" db:"parent_id"`
	SiteID          *uuid.UUID       `json:"siteId" db:"site_id"`
	SiteName        string           `json:"siteName" db:"site_name"`
	HolderUserID    *uuid.UUID       `json:"holderUserId" db:"holder_user_id"`
	HolderName      *string          `json:"holderName" db:"holder_name"`
	HolderFullName  string           `json:"holderFullName" db:"holder_user_full_name"`
	Location        *string          `json:"location" db:"location"`
	PurchaseDate    *string          `json:"purchaseDate" db:"purchase_date"`
	WarrantyUntil   *string          `json:"warrantyUntil" db:"warranty_until"`
	Cost            *float64         `json:"cost" db:"cost"`
	Notes           string           `json:"notes" db:"notes"`
	CreatedBy       *uuid.UUID       `json:"createdBy" db:"created_by"`
	IsKit           bool             `json:"isKit" db:"is_kit"`
	KitID           *uuid.UUID       `json:"kitId" db:"kit_id"`
	CreatedAt       time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time        `json:"updatedAt" db:"updated_at"`
}

// EquipmentFilter — фильтры и пагинация списка единиц.
type EquipmentFilter struct {
	RealmID   uuid.UUID        `json:"realmId" db:"realm_id"`
	TypeID    *uuid.UUID       `form:"typeId,parser=encoding.TextUnmarshaler" json:"typeId" db:"type_id"`
	Status    *EquipmentStatus `form:"status" json:"status" db:"status" binding:"omitempty,enum"`
	SiteID    *uuid.UUID       `form:"siteId,parser=encoding.TextUnmarshaler" json:"siteId" db:"site_id"`
	HolderID  *uuid.UUID       `form:"holderId,parser=encoding.TextUnmarshaler" json:"holderId" db:"holder_user_id"`
	Serial    *string          `form:"serial" json:"serial"`
	Inventory *string          `form:"inventory" json:"inventory"`
	Search    *string          `form:"search" json:"search"`
	Limit     int              `form:"limit" json:"limit"`
	Offset    int              `form:"offset" json:"offset"`
}

type GetEquipmentByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// EquipmentDTO — создание/обновление единицы. Даты — строки «2006-01-02»,
// в БД кладутся через ::date и читаются ::text.
type EquipmentDTO struct {
	ID              *uuid.UUID      `json:"id" db:"id"`
	RealmID         uuid.UUID       `json:"realmId" db:"realm_id"`
	TypeID          uuid.UUID       `json:"typeId" db:"type_id" binding:"required,uuid"`
	Name            string          `json:"name" db:"name" binding:"required"`
	Vendor          string          `json:"vendor" db:"vendor"`
	Model           string          `json:"model" db:"model"`
	SerialNumber    *string         `json:"serialNumber" db:"serial_number"`
	InventoryNumber *string         `json:"inventoryNumber" db:"inventory_number"`
	Specs           json.RawMessage `json:"specs" db:"specs"`
	Status          EquipmentStatus `json:"status" db:"status" binding:"required,enum"`
	ParentID        *uuid.UUID      `json:"parentId" db:"parent_id"`
	SiteID          *uuid.UUID      `json:"siteId" db:"site_id"`
	HolderUserID    *uuid.UUID      `json:"holderUserId" db:"holder_user_id"`
	HolderName      *string         `json:"holderName" db:"holder_name"`
	Location        *string         `json:"location" db:"location"`
	PurchaseDate    *string         `json:"purchaseDate" db:"purchase_date"`
	WarrantyUntil   *string         `json:"warrantyUntil" db:"warranty_until"`
	Cost            *float64        `json:"cost" db:"cost"`
	Notes           string          `json:"notes" db:"notes"`
	IsKit           bool            `json:"isKit" db:"is_kit"`
	KitID           *uuid.UUID      `json:"kitId" db:"kit_id"`
	CreatedBy       *uuid.UUID      `json:"-" db:"created_by"`
}

type DelEquipmentDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// KitComposition — членство компонента в логическом комплекте (интервальная история).
type KitComposition struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	KitID       uuid.UUID  `json:"kitId" db:"kit_id"`
	ComponentID uuid.UUID  `json:"componentId" db:"component_id"`
	StartedAt   time.Time  `json:"startedAt" db:"started_at"`
	EndedAt     *time.Time `json:"endedAt" db:"ended_at"`
	StartedBy   *uuid.UUID `json:"startedBy" db:"started_by"`
	EndedBy     *uuid.UUID `json:"endedBy" db:"ended_by"`
	Reason      string     `json:"reason" db:"reason"`
	ActID       *uuid.UUID `json:"actId" db:"act_id"`
}

// EquipmentAssignment — текущая/историческая выдача единицы.
type EquipmentAssignment struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	EquipmentID  uuid.UUID  `json:"equipmentId" db:"equipment_id"`
	UserID       *uuid.UUID `json:"userId" db:"user_id"`
	HolderName   *string    `json:"holderName" db:"holder_name"`
	SiteID       *uuid.UUID `json:"siteId" db:"site_id"`
	Location     *string    `json:"location" db:"location"`
	AssignedAt   time.Time  `json:"assignedAt" db:"assigned_at"`
	UnassignedAt *time.Time `json:"unassignedAt" db:"unassigned_at"`
	AssignedBy   *uuid.UUID `json:"assignedBy" db:"assigned_by"`
	Comment      string     `json:"comment" db:"comment"`
}

// EquipmentEvent — событие жизненного цикла оборудования.
type EquipmentEvent struct {
	ID           uuid.UUID          `json:"id" db:"id"`
	EquipmentID  uuid.UUID          `json:"equipmentId" db:"equipment_id"`
	EventType    EquipmentEventType `json:"eventType" db:"event_type"`
	OperationID  uuid.UUID          `json:"operationId" db:"operation_id"`
	ComponentID  *uuid.UUID         `json:"componentId" db:"component_id"`
	OldValue     json.RawMessage    `json:"oldValue" db:"old_value"`
	NewValue     json.RawMessage    `json:"newValue" db:"new_value"`
	UserID       *uuid.UUID         `json:"userId" db:"user_id"`
	Comment      string             `json:"comment" db:"comment"`
	CreatedAt    time.Time          `json:"createdAt" db:"created_at"`
}

// EquipmentAct — акт выдачи/приёма.
type EquipmentAct struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	RealmID          uuid.UUID       `json:"realmId" db:"realm_id"`
	Number           string          `json:"number" db:"number"`
	KitEquipmentID   *uuid.UUID      `json:"kitEquipmentId" db:"kit_equipment_id"`
	RecipientUserID  *uuid.UUID      `json:"recipientUserId" db:"recipient_user_id"`
	RecipientName    *string         `json:"recipientName" db:"recipient_name"`
	SiteID           *uuid.UUID      `json:"siteId" db:"site_id"`
	Location         *string         `json:"location" db:"location"`
	IssuedBy         *uuid.UUID      `json:"issuedBy" db:"issued_by"`
	IssuedAt         time.Time       `json:"issuedAt" db:"issued_at"`
	Content          json.RawMessage `json:"content" db:"content"`
	Comment          string          `json:"comment" db:"comment"`
	CreatedAt        time.Time       `json:"createdAt" db:"created_at"`
}

// AssignEquipmentDTO — выдача оборудования.
type AssignEquipmentDTO struct {
	ID               uuid.UUID  `json:"id"`
	RealmID          uuid.UUID  `json:"realmId"`
	UserID           *uuid.UUID `json:"userId"`
	HolderName       *string    `json:"holderName"`
	SiteID           *uuid.UUID `json:"siteId"`
	Location         *string    `json:"location"`
	Comment          string     `json:"comment"`
	Force            bool       `json:"force"`
	SkipAct          bool       `json:"skipAct"`
	Actor            *Actor     `json:"-"`
}

// UnassignEquipmentDTO — возврат оборудования на склад/вместо хранения.
type UnassignEquipmentDTO struct {
	ID         uuid.UUID  `json:"id"`
	RealmID    uuid.UUID  `json:"realmId"`
	SiteID     *uuid.UUID `json:"siteId"`
	Location   *string    `json:"location"`
	Comment    string     `json:"comment"`
	Actor      *Actor     `json:"-"`
}

// ChangeStatusDTO — смена статуса единицы.
type ChangeStatusDTO struct {
	ID      uuid.UUID       `json:"id"`
	RealmID uuid.UUID       `json:"realmId"`
	Status  EquipmentStatus `json:"status" binding:"required,enum"`
	Comment string          `json:"comment"`
	Actor   *Actor          `json:"-"`
}

// ComponentOpItem — элемент операции с составом комплекта.
type ComponentOpItem struct {
	ComponentID uuid.UUID `json:"componentId"`
}

// KitCompositionOpDTO — добавление/удаление/замена состава логического комплекта.
type KitCompositionOpDTO struct {
	ID          uuid.UUID         `json:"id"`
	RealmID     uuid.UUID         `json:"realmId"`
	Components  []ComponentOpItem `json:"components"`
	Reason      string            `json:"reason"`
	Actor       *Actor            `json:"-"`
}

// EquipmentActDTO — создание акта вручную.
type EquipmentActDTO struct {
	ID               uuid.UUID  `json:"id"`
	RealmID          uuid.UUID  `json:"realmId"`
	KitEquipmentID   *uuid.UUID `json:"kitEquipmentId"`
	RecipientUserID  *uuid.UUID `json:"recipientUserId"`
	RecipientName    *string    `json:"recipientName"`
	SiteID           *uuid.UUID `json:"siteId"`
	Location         *string    `json:"location"`
	Comment          string     `json:"comment"`
	Actor            *Actor     `json:"-"`
}

// GetEquipmentActByIDDTO — запрос акта по ID.
type GetEquipmentActByIDDTO struct {
	ID      uuid.UUID `json:"id"`
	RealmID uuid.UUID `json:"realmId"`
}

// EquipmentActsFilter — фильтр списка актов.
type EquipmentActsFilter struct {
	RealmID uuid.UUID  `json:"realmId"`
	KitID   *uuid.UUID `form:"kitId,parser=encoding.TextUnmarshaler" json:"kitId"`
	Limit   int        `form:"limit" json:"limit"`
	Offset  int        `form:"offset" json:"offset"`
}

// EquipmentEventsFilter — пагинация журнала событий единицы.
type EquipmentEventsFilter struct {
	Limit  int `form:"limit" json:"limit"`
	Offset int `form:"offset" json:"offset"`
}

// EquipmentShort — сокращённое представление оборудования.
type EquipmentShort struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
}
