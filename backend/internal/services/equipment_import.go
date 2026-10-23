package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// EquipmentImportService — импорт единиц оборудования из файла (xlsx или csv).
//
// Импорт только создаёт: уже заведённые единицы не трогаются. Ошибка в строке
// не отменяет импорт остальных — в отчёте видно, что именно не получилось и
// почему, поэтому файл можно поправить и загрузить заново.
//
// Единицы создаются через обычный сервис оборудования, поэтому действуют все
// его правила: нормализация полей, вывод статуса из держателя, генерация
// инвентарного номера, добавление магазина в справочник подсказок.
type EquipmentImportService struct {
	// equipment — создание единиц (тот же сервис, что и у формы).
	equipment Equipment
	// sites — справочник площадок: в файле площадка приходит названием.
	sites repository.Sites
	// users — пользователи реалма: в файле получатель приходит ФИО.
	users repository.Users
}

// NewEquipmentImportService создаёт сервис импорта оборудования.
func NewEquipmentImportService(equipment Equipment, sites repository.Sites, users repository.Users) *EquipmentImportService {
	return &EquipmentImportService{equipment: equipment, sites: sites, users: users}
}

// Import читает файл и создаёт единицы оборудования.
func (s *EquipmentImportService) Import(ctx context.Context, dto *models.EquipmentImportDTO) (*models.EquipmentImportResult, error) {
	table, err := readImportTable(dto.File, dto.FileName)
	if err != nil {
		return nil, err
	}
	if table.close != nil {
		defer table.close()
	}

	if len(table.rows) > importMaxRows {
		return nil, fmt.Errorf("%w: в файле %d строк, максимум %d", models.ErrInvalidInput, len(table.rows), importMaxRows)
	}

	columns, err := mapImportColumns(table.header)
	if err != nil {
		return nil, err
	}

	refs, err := s.loadRefs(ctx, dto.RealmID)
	if err != nil {
		return nil, err
	}

	result := &models.EquipmentImportResult{Rows: []*models.EquipmentImportRowResult{}}
	for i, row := range table.rows {
		if isEmptyImportRow(row) {
			continue
		}
		rowResult := s.importRow(ctx, dto, columns, table, row, i, refs)
		result.Total++
		if rowResult.Status == models.EquipmentImportRowCreated {
			result.Created++
		} else {
			result.Failed++
		}
		result.Rows = append(result.Rows, rowResult)
	}
	return result, nil
}

// importRefs — справочники реалма, разрешённые один раз на весь файл: иначе на
// каждую строку шли бы запросы за типами, площадками и пользователями.
type importRefs struct {
	// types — типы оборудования по нормализованному названию.
	types map[string]*models.EquipmentType
	// sites — площадки по нормализованному названию.
	sites map[string]*models.Site
	// holders — пользователи реалма для поиска получателя по ФИО/логину.
	holders *importHolderIndex
}

// loadRefs загружает справочники, нужные импорту.
func (s *EquipmentImportService) loadRefs(ctx context.Context, realmID uuid.UUID) (*importRefs, error) {
	refs := &importRefs{
		types:   map[string]*models.EquipmentType{},
		sites:   map[string]*models.Site{},
		holders: newImportHolderIndex(),
	}

	types, err := s.equipment.GetTypes(ctx, &models.GetEquipmentTypesDTO{RealmID: realmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment types. error: %w", err)
	}
	for _, item := range types {
		key := normalizeImportText(item.Name)
		if key == "" {
			continue
		}
		if _, exists := refs.types[key]; !exists {
			refs.types[key] = item
		}
	}

	sites, err := s.sites.Get(ctx, &models.GetSitesDTO{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sites. error: %w", err)
	}
	for _, item := range sites {
		key := normalizeImportText(item.Name)
		if key == "" {
			continue
		}
		if _, exists := refs.sites[key]; !exists {
			refs.sites[key] = item
		}
	}

	users, err := s.users.GetAll(ctx, &realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users. error: %w", err)
	}
	for _, item := range users {
		refs.holders.add(item)
	}

	return refs, nil
}

// importRow импортирует одну строку файла. Любая проблема строки — это отказ по
// строке, а не отказ всего импорта: остальные строки импортируются.
func (s *EquipmentImportService) importRow(
	ctx context.Context,
	dto *models.EquipmentImportDTO,
	columns *importColumns,
	table *importTable,
	row []string,
	rowIndex int,
	refs *importRefs,
) *models.EquipmentImportRowResult {
	result := &models.EquipmentImportRowResult{
		Row:    rowIndex + 2, // +1 — заголовок, +1 — нумерация с единицы
		Status: models.EquipmentImportRowFailed,
	}
	result.Name = columns.cell(row, importColName)
	if result.Name == "" {
		result.Error = "не указано наименование"
		return result
	}

	notes := []string{}

	// Тип. Неизвестный тип — отказ по строке: создать тип «на лету» нельзя,
	// у типа есть вид (устройство/комплектующее/расходник/периферия), а из
	// названия он не выводится.
	typeName := columns.cell(row, importColType)
	if typeName == "" {
		result.Error = "не указан тип оборудования"
		return result
	}
	eqType, ok := refs.types[normalizeImportText(typeName)]
	if !ok {
		result.Error = fmt.Sprintf("тип «%s» не найден в справочнике", typeName)
		return result
	}

	// Площадка: в файле она названием, в базе — ссылкой. Не найдена — отказ:
	// молча создавать площадки из файла нельзя, это справочник другой ветки.
	var siteID *uuid.UUID
	if siteName := columns.cell(row, importColSite); siteName != "" {
		site, ok := refs.sites[normalizeImportText(siteName)]
		if !ok {
			result.Error = fmt.Sprintf("площадка «%s» не найдена", siteName)
			return result
		}
		siteID = &site.ID
	}

	// Получатель. Если сотрудник нашёлся — ссылка на пользователя; если нет —
	// свободное значение: человек существует, но в системе его нет.
	var holderUserID *uuid.UUID
	var holderName *string
	result.Holder = columns.cell(row, importColHolder)
	if result.Holder != "" {
		if user, found := refs.holders.find(result.Holder); found {
			holderUserID = &user.ID
			result.HolderFound = true
		} else {
			holderName = &result.Holder
			notes = append(notes, "получатель не найден среди пользователей — записан свободным значением")
		}
	}

	// Статус. «На складе»/«в эксплуатации» — не состояние, а следствие
	// выдачи, поэтому вручную их не задать: статус выводится из держателя.
	status, err := importStatus(columns.cell(row, importColStatus), holderUserID != nil || holderName != nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if status == models.EquipmentStatusInStock && (holderUserID != nil || holderName != nil) {
		notes = append(notes, "указан получатель — единица числится в эксплуатации")
	}

	purchaseDate, err := columns.date(row, table, rowIndex, importColPurchase)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	warrantyUntil, err := columns.date(row, table, rowIndex, importColWarranty)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	location := columns.cell(row, importColLocation)
	// Единица должна быть физически учтена: держатель, площадка или локация.
	// Проверяем до вставки, чтобы пользователь получил внятную причину, а не
	// ошибку CHECK-ограничения из базы.
	if holderUserID == nil && holderName == nil && siteID == nil && location == "" {
		result.Error = "не указан получатель, площадка или локация — единицу негде учесть"
		return result
	}

	createDTO := &models.CreateEquipmentDTO{}
	createDTO.RealmID = dto.RealmID
	createDTO.TypeID = &eqType.ID
	createDTO.Name = result.Name
	createDTO.Vendor = columns.cell(row, importColVendor)
	createDTO.Model = columns.cell(row, importColModel)
	createDTO.SerialNumber = optionalImportText(columns.cell(row, importColSerial))
	createDTO.InventoryNumber = optionalImportText(columns.cell(row, importColInventory))
	createDTO.Shop = optionalImportText(columns.cell(row, importColShop))
	createDTO.OrderNumber = optionalImportText(columns.cell(row, importColOrder))
	createDTO.Status = status
	createDTO.SiteID = siteID
	createDTO.HolderUserID = holderUserID
	createDTO.HolderName = holderName
	createDTO.Location = optionalImportText(location)
	createDTO.PurchaseDate = purchaseDate
	createDTO.WarrantyUntil = warrantyUntil
	createDTO.Notes = columns.cell(row, importColNotes)
	createDTO.Specs = columns.specs(row)
	if dto.Actor != nil && dto.Actor.ID != uuid.Nil {
		createDTO.CreatedBy = &dto.Actor.ID
		createDTO.Actor = dto.Actor
	}

	created, err := s.equipment.Create(ctx, createDTO)
	if err != nil {
		result.Error = importErrorMessage(err)
		return result
	}

	result.Status = models.EquipmentImportRowCreated
	result.EquipmentID = &created.ID
	result.InventoryNumber = created.InventoryNumber
	result.Note = strings.Join(notes, "; ")
	return result
}

// importStatus переводит статус из файла в статус единицы.
//
// in_stock/in_use — выводимые из держателя, поэтому в файле их быть не
// должно: «в эксплуатации» без получателя противоречит правилу «выданная
// единица имеет держателя» — это отказ по строке.
func importStatus(raw string, hasHolder bool) (models.EquipmentStatus, error) {
	switch normalizeImportText(raw) {
	case "", "на складе", "склад", "в наличии", "in_stock", "stock":
		return models.EquipmentStatusInStock, nil
	case "в эксплуатации", "выдан", "на руках", "in_use":
		if !hasHolder {
			return "", fmt.Errorf("указан статус «в эксплуатации», но получатель не задан")
		}
		return models.EquipmentStatusInUse, nil
	case "ремонт", "в ремонте", "repair":
		return models.EquipmentStatusRepair, nil
	case "дефект", "дефектное", "неисправно", "defective":
		return models.EquipmentStatusDefective, nil
	case "резерв", "зарезервировано", "reserved":
		return models.EquipmentStatusReserved, nil
	case "списано", "списано с учета", "written_off":
		return models.EquipmentStatusWrittenOff, nil
	default:
		return "", fmt.Errorf("неизвестный статус «%s»", raw)
	}
}

// importErrorMessage переводит ошибку в текст для отчёта: доменная ошибка
// знает своё человеческое сообщение, остальное — как есть.
func importErrorMessage(err error) string {
	var domainErr *models.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Message()
	}
	return err.Error()
}

// optionalImportText превращает пустую строку в nil: пустое поле — это
// «не задано», а не пустое значение.
func optionalImportText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// importHolderIndex — поиск получателя среди пользователей реалма по ФИО,
// логину или почте. Ключи нормализованы, поэтому «Иванов  Иван» и «иванов
// иван» — один ключ.
type importHolderIndex struct {
	byKey map[string][]*models.UserData
}

// newImportHolderIndex создаёт пустой индекс получателей.
func newImportHolderIndex() *importHolderIndex {
	return &importHolderIndex{byKey: map[string][]*models.UserData{}}
}

// add добавляет пользователя в индекс под всеми его ключами.
func (i *importHolderIndex) add(user *models.UserData) {
	for _, key := range importHolderKeys(user) {
		if key == "" {
			continue
		}
		i.byKey[key] = append(i.byKey[key], user)
	}
}

// find ищет получателя по значению из файла.
//
// Активные пользователи предпочтительнее: однофамилец отключённый не должен
// перехватывать выдачу у живого сотрудника. Неоднозначность (несколько
// совпадений) считается «не нашли»: имя записывается свободным значением,
// а не приписывается случайному сотруднику.
func (i *importHolderIndex) find(name string) (*models.UserData, bool) {
	candidates := i.byKey[normalizeImportText(name)]
	if len(candidates) == 0 {
		return nil, false
	}
	active := make([]*models.UserData, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.IsActive {
			active = append(active, candidate)
		}
	}
	if len(active) > 0 {
		candidates = active
	}
	if len(candidates) != 1 {
		return nil, false
	}
	return candidates[0], true
}

// importHolderKeys — ключи пользователя: логин, ФИО в обоих порядках и почта.
func importHolderKeys(user *models.UserData) []string {
	first := strings.TrimSpace(user.FirstName)
	last := strings.TrimSpace(user.LastName)
	return []string{
		normalizeImportText(user.Username),
		normalizeImportText(last + " " + first),
		normalizeImportText(first + " " + last),
		normalizeImportText(user.Email),
	}
}

// date разбирает дату в колонке: пустая ячейка — дата не задана.
func (c *importColumns) date(row []string, table *importTable, rowIndex int, key string) (*string, error) {
	idx, ok := c.byKey[key]
	if !ok {
		return nil, nil
	}
	text := ""
	if idx < len(row) {
		text = strings.TrimSpace(row[idx])
	}
	if text == "" {
		return nil, nil
	}
	raw := ""
	if table.rawCell != nil {
		raw = table.rawCell(idx, rowIndex)
	}
	value, ok := parseImportDate(text, raw)
	if !ok {
		return nil, fmt.Errorf("не удалось разобрать дату «%s»", text)
	}
	return &value, nil
}

// specs собирает характеристики единицы из колонок, которые не стали полями:
// характеристики — свободный словарь, поэтому лишняя колонка не теряется.
func (c *importColumns) specs(row []string) json.RawMessage {
	specs := map[string]string{}
	for i, idx := range c.specColumns {
		if idx >= len(row) {
			continue
		}
		value := strings.TrimSpace(row[idx])
		if value == "" {
			continue
		}
		header := c.specHeaders[i]
		if header == "" {
			continue
		}
		specs[header] = value
	}
	if len(specs) == 0 {
		return json.RawMessage("{}")
	}
	data, err := json.Marshal(specs)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}
