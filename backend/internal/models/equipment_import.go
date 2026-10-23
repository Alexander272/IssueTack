package models

import (
	"io"

	"github.com/google/uuid"
)

// Статусы строки отчёта об импорте.
const (
	// EquipmentImportRowCreated — единица создана.
	EquipmentImportRowCreated = "created"
	// EquipmentImportRowFailed — строку не удалось импортировать, причина в Error.
	EquipmentImportRowFailed = "failed"
)

// EquipmentImportDTO — параметры импорта оборудования из файла.
//
// Realm приходит из запроса, актор — из сессии: тело файла не может ни
// назначить область, ни подделать автора операции.
type EquipmentImportDTO struct {
	RealmID  uuid.UUID
	FileName string
	File     io.Reader
	Actor    *Actor
}

// EquipmentImportResult — итог импорта: сколько единиц создано, сколько строк
// не прошло и почему. Отчёт по строкам — часть результата, а не отдельный
// запрос: импорт операция разовая, а пользователю сразу видно, что чинить.
type EquipmentImportResult struct {
	// Total — число обработанных строк (пустые строки не считаются).
	Total int `json:"total"`
	// Created — число созданных единиц.
	Created int `json:"created"`
	// Failed — число строк, которые не удалось импортировать.
	Failed int `json:"failed"`
	// Rows — результат по каждой строке файла. Всегда массив, даже пустой.
	Rows []*EquipmentImportRowResult `json:"rows"`
}

// EquipmentImportRowResult — результат импорта одной строки файла.
type EquipmentImportRowResult struct {
	// Row — номер строки в файле (как в Excel: первая строка данных — 2).
	Row int `json:"row"`
	// Name — наименование из файла (пусто, если его не удалось прочитать).
	Name string `json:"name"`
	// Status — created или failed.
	Status string `json:"status"`
	// Error — причина отказа по строке (пусто у созданных).
	Error string `json:"error"`
	// Note — предупреждение: единица создана, но что-то пришлось поправить
	// (например, получатель не найден среди пользователей).
	Note string `json:"note"`
	// EquipmentID — идентификатор созданной единицы.
	EquipmentID *uuid.UUID `json:"equipmentId"`
	// InventoryNumber — присвоенный инвентарный номер (пустой — номер не
	// генерировался, например, номер был в файле).
	InventoryNumber *string `json:"inventoryNumber"`
	// Holder — получатель из файла.
	Holder string `json:"holder"`
	// HolderFound — нашёлся ли получатель среди пользователей реалма. false
	// означает свободное значение (holder_name).
	HolderFound bool `json:"holderFound"`
}
