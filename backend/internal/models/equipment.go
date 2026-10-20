package models

import (
	"fmt"
	"io"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// EquipmentStatus — статус единицы оборудования.
type EquipmentStatus string

const (
	EquipmentStatusInStock    EquipmentStatus = "in_stock"
	EquipmentStatusInUse      EquipmentStatus = "in_use"
	EquipmentStatusRepair     EquipmentStatus = "repair"
	EquipmentStatusDefective  EquipmentStatus = "defective"
	EquipmentStatusReserved   EquipmentStatus = "reserved"
	EquipmentStatusWrittenOff EquipmentStatus = "written_off"
)

// IsValid проверяет, что статус принадлежит перечислению (для enum-валидатора).
func (s EquipmentStatus) IsValid() bool {
	switch s {
	case EquipmentStatusInStock, EquipmentStatusInUse, EquipmentStatusRepair,
		EquipmentStatusDefective, EquipmentStatusReserved, EquipmentStatusWrittenOff:
		return true
	}
	return false
}

// EquipmentTypeKind — категория типа оборудования: задаёт группу в справочнике
// и допустимость быть частью физического состава (parent_id).
type EquipmentTypeKind string

const (
	EquipmentTypeDevice     EquipmentTypeKind = "device"
	EquipmentTypeComponent  EquipmentTypeKind = "component"
	EquipmentTypePeriphery  EquipmentTypeKind = "periphery"
	EquipmentTypeConsumable EquipmentTypeKind = "consumable"
)

// IsValid проверяет, что категория принадлежит перечислению (для enum-валидатора).
func (k EquipmentTypeKind) IsValid() bool {
	switch k {
	case EquipmentTypeDevice, EquipmentTypeComponent, EquipmentTypePeriphery, EquipmentTypeConsumable:
		return true
	}
	return false
}

// CanBePart сообщает, может ли единица такого типа быть частью физического
// состава (parent_id): комплектующее или расходник.
func (k EquipmentTypeKind) CanBePart() bool {
	return k == EquipmentTypeComponent || k == EquipmentTypeConsumable
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

// EquipmentInventorySeries — серия автоинвентарного номера: единицы и комплекты
// нумеруются независимо, у каждой серии свой счётчик в БД
// (equipment_inventory_counters). Единица получает только цифры, комплект —
// значение того же вида, но с префиксом «KIT-», чтобы номера двух серий не
// путались при вводе/сканировании.
type EquipmentInventorySeries string

const (
	InventorySeriesUnit EquipmentInventorySeries = "unit"
	InventorySeriesKit  EquipmentInventorySeries = "kit"
)

// InventoryNumber собирает инвентарный номер из серии и порядкового значения
// счётчика: «000123» для единицы и «KIT-000123» для комплекта. Паддинг в шесть
// цифр держит одинаковую длину номера (и этикетки со штрихкодом).
func InventoryNumber(series EquipmentInventorySeries, seq int) string {
	if series == InventorySeriesKit {
		return fmt.Sprintf("KIT-%06d", seq)
	}
	return fmt.Sprintf("%06d", seq)
}

// EquipmentType — тип оборудования (справочник реалма): ноутбук, RAM, монитор и т.д.
type EquipmentType struct {
	ID      uuid.UUID         `json:"id" db:"id"`
	RealmID uuid.UUID         `json:"realmId" db:"realm_id"`
	Name    string            `json:"name" db:"name"`
	Kind    EquipmentTypeKind `json:"kind" db:"kind"`
	// Icon — ключ иконки из закрытого набора фронтового реестра; пустая
	// строка = «Авто» (фронт подбирает иконку по названию типа).
	Icon      string    `json:"icon" db:"icon"`
	SortOrder int       `json:"sortOrder" db:"sort_order"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type GetEquipmentTypesDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type GetEquipmentTypeByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type EquipmentTypeDTO struct {
	ID      *uuid.UUID        `json:"id" db:"id"`
	RealmID uuid.UUID         `json:"realmId" db:"realm_id"`
	Name    string            `json:"name" validate:"required" db:"name"`
	Kind    EquipmentTypeKind `json:"kind" db:"kind" binding:"required,enum"`
	// Icon пустой допустим (= «Авто»): набор ключей живёт во фронтовом
	// реестре, бэкенд его не валидирует.
	Icon      string `json:"icon" db:"icon"`
	SortOrder int    `json:"sortOrder" db:"sort_order"`
}

type DelEquipmentTypeDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// Equipment — единица оборудования (или компонент комплекта).
type Equipment struct {
	ID              uuid.UUID         `json:"id" db:"id"`
	RealmID         uuid.UUID         `json:"realmId" db:"realm_id"`
	TypeID          *uuid.UUID        `json:"typeId" db:"type_id"`
	TypeName        string            `json:"typeName" db:"type_name"`
	TypeKind        EquipmentTypeKind `json:"typeKind" db:"type_kind"`
	TypeIcon        string            `json:"typeIcon" db:"type_icon"`
	Name            string            `json:"name" db:"name"`
	Vendor          string            `json:"vendor" db:"vendor"`
	Model           string            `json:"model" db:"model"`
	SerialNumber    *string           `json:"serialNumber" db:"serial_number"`
	InventoryNumber *string           `json:"inventoryNumber" db:"inventory_number"`
	Specs           json.RawMessage   `json:"specs" db:"specs"`
	Status          EquipmentStatus   `json:"status" db:"status"`
	ParentID        *uuid.UUID        `json:"parentId" db:"parent_id"`
	SiteID          *uuid.UUID        `json:"siteId" db:"site_id"`
	SiteName        string            `json:"siteName" db:"site_name"`
	HolderUserID    *uuid.UUID        `json:"holderUserId" db:"holder_user_id"`
	HolderName      *string           `json:"holderName" db:"holder_name"`
	HolderFullName  string            `json:"holderFullName" db:"holder_user_full_name"`
	// ReturnDueAt — срок возврата текущей (открытой) выдачи. nil — держателя нет
	// или срок не задавали. Нужен карточке, чтобы показать «просрочен возврат».
	ReturnDueAt   *time.Time `json:"returnDueAt" db:"return_due_at"`
	Location      *string    `json:"location" db:"location"`
	PurchaseDate  *string    `json:"purchaseDate" db:"purchase_date"`
	WarrantyUntil *string    `json:"warrantyUntil" db:"warranty_until"`
	Notes         string     `json:"notes" db:"notes"`
	// ContainerName — имя контейнера для подписи «внутри …» в плоском списке:
	// физический родитель (parent_id) или логический комплект (kit_id).
	ContainerName *string    `json:"containerName" db:"container_name"`
	CreatedBy     *uuid.UUID `json:"createdBy" db:"created_by"`
	IsKit         bool       `json:"isKit" db:"is_kit"`
	KitID         *uuid.UUID `json:"kitId" db:"kit_id"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// EquipmentTreeNode — узел дерева оборудования для страницы-дерева: единица со
// своими детьми — текущими членами логического комплекта и/или физического состава
// (parent_id). Поля самой единицы разворачиваются в JSON рядом с children.
//
// Дерево грузится уровнями (лениво), поэтому children заполняются только у того
// уровня, который запрошен, а HasChildren/ChildCount приходят с сервера заранее:
// по ним рисуется шеврон раскрытия и счётчик «(N)» без загрузки всего поддерева.
type EquipmentTreeNode struct {
	*Equipment
	Children    []*EquipmentTreeNode `json:"children"`
	HasChildren bool                 `json:"hasChildren"`
	ChildCount  int                  `json:"childCount"`
}

// EquipmentFilter — фильтры и пагинация списка единиц.
type EquipmentFilter struct {
	RealmID   uuid.UUID        `json:"realmId" db:"realm_id"`
	TypeID    *uuid.UUID       `form:"typeId,parser=encoding.TextUnmarshaler" json:"typeId" db:"type_id"`
	Status    *EquipmentStatus `form:"status" json:"status" db:"status" binding:"omitempty,enum"`
	IsKit     *bool            `form:"isKit" json:"isKit" db:"is_kit"`
	SiteID    *uuid.UUID       `form:"siteId,parser=encoding.TextUnmarshaler" json:"siteId" db:"site_id"`
	HolderID  *uuid.UUID       `form:"holderId,parser=encoding.TextUnmarshaler" json:"holderId" db:"holder_user_id"`
	Serial    *string          `form:"serial" json:"serial"`
	Inventory *string          `form:"inventory" json:"inventory"`
	Search    *string          `form:"search" json:"search"`
	Limit     int              `form:"limit" json:"limit"`
	Offset    int              `form:"offset" json:"offset"`
}

// GetEquipmentTreeDTO — параметры ленивого дерева: ParentID == nil — корни реалма,
// иначе — непосредственные дети узла (физсостав + члены комплекта). Limit/Offset
// пагинируют уровни, чтобы даже сотни корней не приезжали одним ответом. Status и
// IsKit сужают только корни: дети раскрытого узла показываются целиком (как раньше
// на клиенте — фильтр по статусу применялся к верхнему уровню).
type GetEquipmentTreeDTO struct {
	RealmID  uuid.UUID        `json:"realmId" db:"realm_id" form:"-"`
	ParentID *uuid.UUID       `form:"parentId,parser=encoding.TextUnmarshaler" json:"parentId" db:"parent_id"`
	Status   *EquipmentStatus `form:"status" json:"status" binding:"omitempty,enum"`
	// IsKit: nil — все корни, true — комплекты, false — единицы.
	IsKit  *bool `form:"isKit" json:"isKit" db:"is_kit"`
	Limit  int   `form:"limit" json:"limit"`
	Offset int   `form:"offset" json:"offset"`
}

// EquipmentStats — счётчики единиц реалма для подписи в шапке страницы.
type EquipmentStats struct {
	// Units — число неустановленных единиц (не комплектов): устройства, периферия,
	// комплектующие и расходники вместе.
	Units int `json:"units"`
	// Kits — число логических комплектов (is_kit).
	Kits int `json:"kits"`
	// StockParts — комплектующие/расходники на складе (status = in_stock).
	StockParts int `json:"stockParts"`
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
	TypeID          *uuid.UUID      `json:"typeId" db:"type_id"`
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
	Notes           string          `json:"notes" db:"notes"`
	IsKit           bool            `json:"isKit" db:"is_kit"`
	KitID           *uuid.UUID      `json:"kitId" db:"kit_id"`
	CreatedBy       *uuid.UUID      `json:"-" db:"created_by"`
	// Actor — пользователь, выполняющий операцию. Нужен журналу: автор события
	// «создано»/«перемещено». Из запроса не приходит — его проставляет обработчик.
	Actor *Actor `json:"-"`
}

type DelEquipmentDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// CreateEquipmentDTO — тело POST /equipment: единица/комплект и, для комплекта,
// необязательный стартовый состав. Поля единицы разворачиваются в JSON рядом с
// components (встроенная структура), поэтому существующий контракт не меняется.
// Состав принимается только при создании: в EquipmentDTO его нет, чтобы PUT не
// мог молча переопределить состав комплекта.
type CreateEquipmentDTO struct {
	EquipmentDTO
	Components []ComponentOpItem `json:"components"`
}

// OpenComponentsDTO — параметры добавления состава в рамках транзакции. Один
// набор правил на два входа: Create (стартовый состав) и AddComponents.
type OpenComponentsDTO struct {
	RealmID    uuid.UUID
	KitID      uuid.UUID
	Components []ComponentOpItem
	Reason     string
	Actor      *Actor
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

	// Подписи ссылок резолвит SQL (см. compositionSelectBase), чтобы клиенту не
	// приходилось тянуть справочники реалма ради одной строки истории.
	ComponentName string `json:"componentName" db:"component_name"`
	StartedByName string `json:"startedByName" db:"started_by_name"`
	EndedByName   string `json:"endedByName" db:"ended_by_name"`
}

// EquipmentKitTemplate — шаблон комплекта: именованный набор типов оборудования,
// из которого собирают логический комплект (например, «Стандартное рабочее место»).
type EquipmentKitTemplate struct {
	ID          uuid.UUID                   `json:"id" db:"id"`
	RealmID     uuid.UUID                   `json:"realmId" db:"realm_id"`
	Name        string                      `json:"name" db:"name"`
	Description string                      `json:"description" db:"description"`
	SortOrder   int                         `json:"sortOrder" db:"sort_order"`
	Items       []*EquipmentKitTemplateItem `json:"items" db:"-"`
	CreatedAt   time.Time                   `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time                   `json:"updatedAt" db:"updated_at"`
}

// EquipmentKitTemplateItem — позиция шаблона: требуемый тип оборудования.
// TypeName/TypeKind резолвит SQL-джойн, чтобы клиенту не тянуть справочник типов.
type EquipmentKitTemplateItem struct {
	ID         uuid.UUID         `json:"id" db:"id"`
	TemplateID uuid.UUID         `json:"templateId" db:"template_id"`
	TypeID     uuid.UUID         `json:"typeId" db:"type_id"`
	TypeName   string            `json:"typeName" db:"type_name"`
	TypeKind   EquipmentTypeKind `json:"typeKind" db:"type_kind"`
	SortOrder  int               `json:"sortOrder" db:"sort_order"`
}

type GetEquipmentKitTemplatesDTO struct {
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

type GetEquipmentKitTemplateByIdDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
}

// EquipmentKitTemplateDTO — создание/правка шаблона. Items передаются целиком:
// правка заменяет состав шаблона, а не патчит отдельные позиции.
type EquipmentKitTemplateDTO struct {
	ID          *uuid.UUID                    `json:"id" db:"id"`
	RealmID     uuid.UUID                     `json:"realmId" db:"realm_id"`
	Name        string                        `json:"name" db:"name" binding:"required"`
	Description string                        `json:"description" db:"description"`
	SortOrder   int                           `json:"sortOrder" db:"sort_order"`
	Items       []EquipmentKitTemplateItemDTO `json:"items"`
}

type EquipmentKitTemplateItemDTO struct {
	TypeID    uuid.UUID `json:"typeId" binding:"required,uuid"`
	SortOrder int       `json:"sortOrder"`
}

type DelEquipmentKitTemplateDTO struct {
	ID      uuid.UUID `json:"id" db:"id"`
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
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
	// ReturnDueAt — ожидаемый срок возврата (nil = срок не задавали). Реквизит
	// конкретной выдачи: у следующей он свой, а закрытая строка хранит свой срок.
	ReturnDueAt *time.Time `json:"returnDueAt" db:"return_due_at"`
	AssignedBy  *uuid.UUID `json:"assignedBy" db:"assigned_by"`
	Comment     string     `json:"comment" db:"comment"`

	// Подписи ссылок резолвит SQL (см. assignmentSelectBase): ФИО держателя и того,
	// кто выдал (fallback — свободный текст holder_name), название площадки.
	HolderFullName string `json:"holderFullName" db:"holder_full_name"`
	SiteName       string `json:"siteName" db:"site_name"`
	AssignedByName string `json:"assignedByName" db:"assigned_by_name"`
}

// EquipmentDueReturn — открытая выдача, у которой наступил срок возврата. Данных
// ровно столько, сколько нужно напоминанию: кому шлём (IssuedBy), о чём (имя и
// номер единицы) и на какой срок ссылаться (ReturnDueAt).
type EquipmentDueReturn struct {
	AssignmentID uuid.UUID `json:"assignmentId" db:"assignment_id"`
	EquipmentID  uuid.UUID `json:"equipmentId" db:"equipment_id"`
	// RealmID — реалм единицы: по нему выбирается Mattermost-интеграция для
	// отправки напоминания.
	RealmID uuid.UUID `json:"realmId" db:"realm_id"`
	// IssuedBy — тот, кто выдал: напоминание уходит ему. Пусто быть не может:
	// выдачи без автора (старые/миграции) в выборку не попадают.
	IssuedBy        uuid.UUID `json:"issuedBy" db:"issued_by"`
	HolderName      string    `json:"holderName" db:"holder_name"`
	SiteName        string    `json:"siteName" db:"site_name"`
	Name            string    `json:"name" db:"name"`
	TypeName        string    `json:"typeName" db:"type_name"`
	InventoryNumber *string   `json:"inventoryNumber" db:"inventory_number"`
	ReturnDueAt     time.Time `json:"returnDueAt" db:"return_due_at"`
}

// EquipmentEvent — событие жизненного цикла оборудования.
type EquipmentEvent struct {
	ID          uuid.UUID          `json:"id" db:"id"`
	EquipmentID uuid.UUID          `json:"equipmentId" db:"equipment_id"`
	EventType   EquipmentEventType `json:"eventType" db:"event_type"`
	OperationID uuid.UUID          `json:"operationId" db:"operation_id"`
	ComponentID *uuid.UUID         `json:"componentId" db:"component_id"`
	OldValue    json.RawMessage    `json:"oldValue" db:"old_value"`
	NewValue    json.RawMessage    `json:"newValue" db:"new_value"`
	UserID      *uuid.UUID         `json:"userId" db:"user_id"`
	// UserName — ФИО актора, резолвит SQL по user_id (см. GetEvents). Журнал
	// иммутабелен, но актор — это ссылка на живого пользователя: подпись читается
	// на момент чтения, а не пишется снимком.
	UserName  string    `json:"userName" db:"user_name"`
	Comment   string    `json:"comment" db:"comment"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// EquipmentActStatus — состояние акта. Акт рождается «сформированным» (docx
// собран и отдан пользователю) и становится «подписанным», когда подгружен скан
// или фото подписанного документа.
type EquipmentActStatus string

const (
	ActStatusFormed EquipmentActStatus = "formed"
	ActStatusSigned EquipmentActStatus = "signed"
)

// EquipmentAct — акт выдачи/приёма.
type EquipmentAct struct {
	ID              uuid.UUID          `json:"id" db:"id"`
	RealmID         uuid.UUID          `json:"realmId" db:"realm_id"`
	Number          string             `json:"number" db:"number"`
	Status          EquipmentActStatus `json:"status" db:"status"`
	KitEquipmentID  *uuid.UUID         `json:"kitEquipmentId" db:"kit_equipment_id"`
	RecipientUserID *uuid.UUID         `json:"recipientUserId" db:"recipient_user_id"`
	RecipientName   *string            `json:"recipientName" db:"recipient_name"`
	SiteID          *uuid.UUID         `json:"siteId" db:"site_id"`
	Location        *string            `json:"location" db:"location"`
	IssuedBy        *uuid.UUID         `json:"issuedBy" db:"issued_by"`
	IssuedAt        time.Time          `json:"issuedAt" db:"issued_at"`
	Content         json.RawMessage    `json:"content" db:"content"`
	Comment         string             `json:"comment" db:"comment"`
	SignedFileName  *string            `json:"signedFileName" db:"signed_file_name"`
	SignedMimeType  *string            `json:"signedMimeType" db:"signed_mime_type"`
	SignedFileSize  *int64             `json:"signedFileSize" db:"signed_file_size"`
	SignedAt        *time.Time         `json:"signedAt" db:"signed_at"`
	SignedBy        *uuid.UUID         `json:"signedBy" db:"signed_by"`
	// SignedFilePath — путь файла на диске. Наружу не уходит: клиенту нужны имя,
	// тип и размер, а раскладку хранилища не показываем.
	SignedFilePath *string   `json:"-" db:"signed_file_path"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

// AssignEquipmentDTO — выдача оборудования.
type AssignEquipmentDTO struct {
	ID         uuid.UUID  `json:"id"`
	RealmID    uuid.UUID  `json:"realmId"`
	UserID     *uuid.UUID `json:"userId"`
	HolderName *string    `json:"holderName"`
	// SiteID и Location — место единицы. Пусто в обоих полях = «единица не
	// переезжает»: место берётся у самой единицы. Указанное место важнее и
	// записывается как есть (выдача сразу на другой объект).
	SiteID   *uuid.UUID `json:"siteId"`
	Location *string    `json:"location"`
	Comment  string     `json:"comment"`
	Force    bool       `json:"force"`
	SkipAct  bool       `json:"skipAct"`
	// AssignedAt — момент выдачи из формы (RFC3339). nil = сейчас. Будущее
	// время отклоняется (EQ017): оформлять выдачу заранее нельзя, а задним
	// числом — можно, акт подписывают позже факта.
	AssignedAt *time.Time `json:"assignedAt"`
	// ReturnDueAt — ожидаемый срок возврата (RFC3339). nil = не задан (выдают и
	// бессрочно). Раньше момента выдачи быть не может (EQ019).
	ReturnDueAt *time.Time `json:"returnDueAt"`
	Actor       *Actor     `json:"-"`
}

// UnassignEquipmentDTO — возврат оборудования на склад/вместо хранения.
type UnassignEquipmentDTO struct {
	ID      uuid.UUID `json:"id"`
	RealmID uuid.UUID `json:"realmId"`
	// SiteID и Location — куда вернуть. Пусто в обоих полях = «дом»: место
	// последнего складского периода, а если его не было — текущее место единицы.
	SiteID   *uuid.UUID `json:"siteId"`
	Location *string    `json:"location"`
	Comment  string     `json:"comment"`
	Actor    *Actor     `json:"-"`
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
	ID         uuid.UUID         `json:"id"`
	RealmID    uuid.UUID         `json:"realmId"`
	Components []ComponentOpItem `json:"components"`
	Reason     string            `json:"reason"`
	Actor      *Actor            `json:"-"`
}

// EquipmentActDTO — создание акта вручную.
type EquipmentActDTO struct {
	ID              uuid.UUID  `json:"id"`
	RealmID         uuid.UUID  `json:"realmId"`
	KitEquipmentID  *uuid.UUID `json:"kitEquipmentId"`
	RecipientUserID *uuid.UUID `json:"recipientUserId"`
	RecipientName   *string    `json:"recipientName"`
	SiteID          *uuid.UUID `json:"siteId"`
	Location        *string    `json:"location"`
	Comment         string     `json:"comment"`
	// IssuedAt — момент акта из формы выдачи (RFC3339), nil = сейчас.
	// Внутреннее поле: наружу через этот DTO не уходит, поэтому json "-".
	IssuedAt *time.Time `json:"-"`
	Actor    *Actor     `json:"-"`
}

// GetEquipmentActByIDDTO — запрос акта по ID.
type GetEquipmentActByIDDTO struct {
	ID      uuid.UUID `json:"id"`
	RealmID uuid.UUID `json:"realmId"`
}

// SignEquipmentActDTO — загрузка подписанного акта (скан/фото с телефона).
type SignEquipmentActDTO struct {
	ID       uuid.UUID
	RealmID  uuid.UUID
	FileName string
	FileSize int64
	MimeType string
	// FilePath заполняет сервис после сохранения файла на диск.
	FilePath string
	File     io.Reader
	Actor    *Actor
}

// EquipmentActDocument — сформированный docx акта: имя файла для
// Content-Disposition и готовое содержимое. Собирается по запросу из снимка
// состава, поэтому на диске не хранится.
type EquipmentActDocument struct {
	FileName string
	Content  []byte
}

// EquipmentActsFilter — фильтр списка актов.
type EquipmentActsFilter struct {
	RealmID uuid.UUID `json:"realmId"`
	// KitID — акты конкретного комплекта.
	KitID *uuid.UUID `form:"kitId,parser=encoding.TextUnmarshaler" json:"kitId"`
	// EquipmentID — акты единицы: и те, где она комплект, и те, где она позиция
	// состава. Фильтр серверный: тянуть весь список реалма ради одной карточки
	// нельзя — за пагинацией его не видно.
	EquipmentID *uuid.UUID `form:"equipmentId,parser=encoding.TextUnmarshaler" json:"equipmentId"`
	Limit       int        `form:"limit" json:"limit"`
	Offset      int        `form:"offset" json:"offset"`
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
