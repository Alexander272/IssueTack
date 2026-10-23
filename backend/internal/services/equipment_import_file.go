package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// importMaxRows — предохранитель от файла в десятки тысяч строк: импорт держит
// в памяти отчёт по каждой строке, а заводить за один раз тысячи единиц всё
// равно неразумно.
const importMaxRows = 5000

// importDateLayouts — форматы дат, которые встречаются в выгрузках. Порядок
// важен: первый подходящий и определяет разбор. Русский формат — сначала день,
// потом месяц («02.01.2006» — 2 января).
var importDateLayouts = []string{
	"2006-01-02",
	"02.01.2006",
	"2.1.2006",
	"02/01/2006",
	"2/1/2006",
	"2006/01/02",
	"02-01-2006",
	"2-1-2006",
	"2006-01-02 15:04:05",
	"02.01.2006 15:04:05",
	"20060102",
	"02.01.06",
}

// importTable — разобранный файл импорта: заголовок и строки данных.
type importTable struct {
	// header — заголовки колонок (первая строка файла).
	header []string
	// rows — строки данных (без заголовка).
	rows [][]string
	// rawCell возвращает «сырое» значение ячейки (без форматирования): нужно
	// датам, у которых формат ячейки нечитаем. nil у csv — там всё и есть текст.
	rawCell func(col, row int) string
	// close освобождает ресурсы разбора (открытую книгу). nil, если нечего закрывать.
	close func()
}

// readImportTable читает файл импорта: xlsx или csv. Формат определяется по
// расширению имени и по сигнатуре файла (zip-контейнер xlsx) — расширение
// приходит от клиента и может врать.
func readImportTable(r io.Reader, fileName string) (*importTable, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file. error: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%w: файл пуст", models.ErrInvalidInput)
	}
	// Старый бинарный формат Excel — не zip и не текст: без явной проверки он
	// разобрался бы как csv и дал бы «нет колонки Наименование».
	if isLegacyXLS(data) {
		return nil, fmt.Errorf("%w: формат .xls не поддерживается — сохраните книгу как .xlsx или .csv", models.ErrInvalidInput)
	}

	if isXLSXContent(data, fileName) {
		return readXLSXTable(data)
	}
	return readCSVTable(data)
}

// isXLSXContent определяет, что файл — книга Excel: zip-сигнатура или
// расширение .xlsx/.xlsm.
func isXLSXContent(data []byte, fileName string) bool {
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return true
	}
	name := strings.ToLower(fileName)
	return strings.HasSuffix(name, ".xlsx") || strings.HasSuffix(name, ".xlsm")
}

// isLegacyXLS определяет старый бинарный формат Excel (OLE2). Отдельная
// проверка нужна ради внятной ошибки: такой файл ни книга, ни текст.
func isLegacyXLS(data []byte) bool {
	return bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
}

// readXLSXTable читает первый лист книги с данными. Значения берутся
// отформатированными (как их видит пользователь), «сырое» значение ячейки
// достаётся лениво — только когда отформатированное не разобралось как дата.
func readXLSXTable(data []byte) (*importTable, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: не удалось открыть книгу Excel: %v", models.ErrInvalidInput, err)
	}

	var (
		sheet string
		rows  [][]string
	)
	for _, name := range file.GetSheetList() {
		sheetRows, err := file.GetRows(name)
		if err != nil {
			continue
		}
		if len(sheetRows) > 0 {
			sheet, rows = name, sheetRows
			break
		}
	}
	if sheet == "" {
		file.Close()
		return nil, fmt.Errorf("%w: в книге нет ни одного листа с данными", models.ErrInvalidInput)
	}

	table := &importTable{
		header: rows[0],
		rows:   rows[1:],
		rawCell: func(col, row int) string {
			ref, err := excelize.CoordinatesToCellName(col+1, row+2)
			if err != nil {
				return ""
			}
			value, err := file.GetCellValue(sheet, ref, excelize.Options{RawCellValue: true})
			if err != nil {
				return ""
			}
			return value
		},
		close: func() { file.Close() },
	}
	return table, nil
}

// readCSVTable читает csv-файл. Разделитель определяется по первой строке
// (русские выгрузки чаще приходят с «;»), кодировка — utf-8 либо windows-1251,
// в которой русский Excel сохраняет csv по умолчанию.
func readCSVTable(data []byte) (*importTable, error) {
	text, err := decodeImportText(data)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = detectCSVDelimiter(text)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: не удалось разобрать csv: %v", models.ErrInvalidInput, err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: в файле нет ни одной строки", models.ErrInvalidInput)
	}

	table := &importTable{header: records[0], rows: records[1:]}
	return table, nil
}

// decodeImportText приводит содержимое файла к тексту: снимает BOM, а если файл
// не utf-8 — декодирует как windows-1251.
func decodeImportText(data []byte) (string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(data) {
		return string(data), nil
	}
	decoded, _, err := transform.Bytes(charmap.Windows1251.NewDecoder(), data)
	if err != nil {
		return "", fmt.Errorf("%w: не удалось определить кодировку файла", models.ErrInvalidInput)
	}
	return string(decoded), nil
}

// detectCSVDelimiter выбирает разделитель csv по первой непустой строке:
// побеждает тот знак, которого в строке больше.
func detectCSVDelimiter(text string) rune {
	line := text
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		line = text[:idx]
	}
	best, bestCount := ';', 0
	for _, candidate := range []rune{';', ',', '\t', '|'} {
		if count := strings.Count(line, string(candidate)); count > bestCount {
			best, bestCount = candidate, count
		}
	}
	return best
}

// Ключи полей единицы, которые распознаются в заголовках файла.
const (
	importColName      = "name"
	importColType      = "type"
	importColSerial    = "serial"
	importColInventory = "inventory"
	importColVendor    = "vendor"
	importColModel     = "model"
	importColShop      = "shop"
	importColOrder     = "order"
	importColHolder    = "holder"
	importColSite      = "site"
	importColLocation  = "location"
	importColPurchase  = "purchase"
	importColWarranty  = "warranty"
	importColNotes     = "notes"
	importColStatus    = "status"
)

// importHeaderAliases — синонимы заголовков колонок. Ключи нормализованы
// (см. normalizeImportText), поэтому «Серийный №» и «серийный номер» —
// один и тот же ключ.
var importHeaderAliases = map[string]string{
	// наименование
	"наименование": importColName, "наименование оборудования": importColName,
	"наименование единицы": importColName, "наименование товара": importColName,
	"название": importColName, "имя": importColName, "оборудование": importColName,
	"equipment": importColName, "name": importColName, "предмет": importColName,
	// тип
	"тип": importColType, "тип оборудования": importColType, "тип техники": importColType,
	"equipment type": importColType, "type": importColType, "вид": importColType,
	// серийный номер
	"серийный": importColSerial, "серийный номер": importColSerial,
	"серийник": importColSerial, "заводской номер": importColSerial,
	"s n": importColSerial, "s n number": importColSerial, "sn": importColSerial,
	"serial": importColSerial, "serial number": importColSerial,
	// инвентарный номер
	"инвентарный": importColInventory, "инвентарный номер": importColInventory,
	"инвентарник": importColInventory, "инв": importColInventory,
	"inventory": importColInventory, "inventory number": importColInventory,
	// производитель
	"производитель": importColVendor, "вендор": importColVendor, "поставщик": importColVendor,
	"торговая марка": importColVendor, "бренд": importColVendor,
	"vendor": importColVendor, "manufacturer": importColVendor,
	// модель
	"модель": importColModel, "model": importColModel,
	// магазин
	"магазин": importColShop, "shop": importColShop,
	// номер заказа
	"заказ": importColOrder, "заказа": importColOrder, "номер заказа": importColOrder,
	"номер закупки": importColOrder, "order": importColOrder, "order number": importColOrder,
	// получатель
	"получатель": importColHolder, "получатель фио": importColHolder,
	"владелец": importColHolder, "держатель": importColHolder,
	"ответственный": importColHolder, "сотрудник": importColHolder,
	"кому выдан": importColHolder, "выдан": importColHolder,
	"holder": importColHolder, "recipient": importColHolder,
	// площадка
	"площадка": importColSite, "площадка объект": importColSite, "объект": importColSite,
	"site": importColSite, "местонахождение": importColSite,
	// локация
	"локация": importColLocation, "место": importColLocation, "адрес": importColLocation,
	"рабочее место": importColLocation, "кабинет": importColLocation,
	"место размещения": importColLocation, "location": importColLocation,
	// даты
	"дата покупки": importColPurchase, "дата приобретения": importColPurchase,
	"дата закупки": importColPurchase, "дата поступления": importColPurchase,
	"дата": importColPurchase, "purchase": importColPurchase, "purchase date": importColPurchase,
	"гарантия": importColWarranty, "гарантия до": importColWarranty,
	"гарантийный срок": importColWarranty, "срок гарантии": importColWarranty,
	"warranty": importColWarranty, "warranty until": importColWarranty,
	// примечания
	"примечания": importColNotes, "комментарий": importColNotes, "заметки": importColNotes,
	"описание": importColNotes, "notes": importColNotes, "comment": importColNotes,
	"description": importColNotes,
	// статус
	"статус": importColStatus, "status": importColStatus, "состояние": importColStatus,
}

// importIgnoredHeaders — колонки, которые не превращаются в характеристики:
// служебные (номер строки в выгрузке, идентификатор) — они к единице ничего
// не относят, а в характеристиках только засоряют карточку.
var importIgnoredHeaders = map[string]bool{
	"номер": true, "номер строки": true, "п п": true, "id": true,
	"индекс": true, "порядок": true, "порядковый номер": true,
	"row": true, "index": true, "uuid": true, "guid": true,
}

// importColumns — раскладка колонок файла по полям единицы.
type importColumns struct {
	// byKey — индекс колонки по ключу поля.
	byKey map[string]int
	// specColumns — индексы колонок, которые не опознаны как поля: их значения
	// становятся характеристиками единицы (свободный словарь).
	specColumns []int
	// specHeaders — заголовки тех же колонок (по индексам specColumns).
	specHeaders []string
}

// mapImportColumns сопоставляет заголовки файла с полями единицы. Колонка без
// наименования — ошибка всего импорта: без неё файл бессмысленен.
func mapImportColumns(header []string) (*importColumns, error) {
	columns := &importColumns{byKey: map[string]int{}}

	for i, title := range header {
		key := normalizeImportText(title)
		if key == "" {
			continue
		}
		if field, ok := importHeaderAliases[key]; ok {
			// Первая колонка с таким именем побеждает: повтор в шапке — не
			// повод молча терять вторую.
			if _, exists := columns.byKey[field]; !exists {
				columns.byKey[field] = i
			}
			continue
		}
		if importIgnoredHeaders[key] {
			continue
		}
		columns.specColumns = append(columns.specColumns, i)
		columns.specHeaders = append(columns.specHeaders, strings.TrimSpace(title))
	}

	if _, ok := columns.byKey[importColName]; !ok {
		return nil, fmt.Errorf("%w: в файле нет колонки «Наименование»", models.ErrInvalidInput)
	}
	return columns, nil
}

// cell возвращает значение колонки строки (пустая строка, если колонки нет).
func (c *importColumns) cell(row []string, key string) string {
	idx, ok := c.byKey[key]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// normalizeImportText приводит значение к сравнимому виду: нижний регистр,
// «ё» вместо «е», без лишних пробелов и знака «№».
func normalizeImportText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "№", " ")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "(", " ")
	s = strings.ReplaceAll(s, ")", " ")
	return strings.Join(strings.Fields(s), " ")
}

// parseImportDate разбирает дату из ячейки. Сначала пробуется отформатированное
// значение (его видит пользователь), затем «сырое» — в нём может лежать
// серийный номер Excel, если формат ячейки нечитаем.
func parseImportDate(text, raw string) (string, bool) {
	text = strings.TrimSpace(text)
	for _, layout := range importDateLayouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}

	serial, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return "", false
	}
	parsed, err := excelize.ExcelDateToTime(serial, false)
	if err != nil {
		return "", false
	}
	// Серийный номер Excel — это число дней с 1899 года. Отсекаем неправдоподобные
	// значения, чтобы «45292» в текстовой ячейке не превратилось в дату.
	if parsed.Year() < 1970 || parsed.Year() > 2100 {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

// isEmptyImportRow сообщает, что строка файла пустая: такие строки не считаются
// ни созданными, ни ошибочными — их просто нет.
func isEmptyImportRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
