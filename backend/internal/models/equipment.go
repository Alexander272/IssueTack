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
	CreatedBy       *uuid.UUID      `json:"-" db:"created_by"`
}

type DelEquipmentDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}
