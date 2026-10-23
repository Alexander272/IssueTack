package equipment

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
)

// EquipmentImporter — импорт единиц оборудования из файла.
type EquipmentImporter interface {
	Import(ctx context.Context, dto *models.EquipmentImportDTO) (*models.EquipmentImportResult, error)
}

// importTemplateHeaders — заголовки шаблона: те же названия, которые импорт
// понимает. Порядок повторяет карточку единицы, чтобы заполнять было привычно.
var importTemplateHeaders = []string{
	"Наименование",
	"Тип",
	"Серийный номер",
	"Инвентарный номер",
	"Производитель",
	"Модель",
	"Магазин",
	"Номер заказа",
	"Получатель",
	"Площадка",
	"Локация",
	"Дата покупки",
	"Гарантия до",
	"Примечания",
	"Статус",
}

// importTemplateExample — пример строки: показывает формат даты и то, что
// получателя можно указать ФИО (сотрудник найдётся сам, иначе запишется
// свободным значением).
var importTemplateExample = []string{
	"Ноутбук Dell Latitude 5450",
	"Ноутбук",
	"ABC123",
	"",
	"Dell",
	"Latitude 5450",
	"Ситилинк",
	"И-2026-15",
	"Иванов Иван",
	"Офис на Мира 10",
	"Кабинет 312",
	"15.01.2026",
	"15.01.2029",
	"Выдан сотруднику",
	"в эксплуатации",
}

// importEquipment принимает файл (xlsx или csv) и создаёт единицы оборудования.
//
// Отчёт по строкам возвращается вместе с результатом: импорт разовый, и
// пользователю сразу видно, какие строки не прошли и почему.
func (h *Handler) importEquipment(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	// К пределу добавляем запас на multipart-служебные поля (~64 КБ), чтобы тело
	// с файлом допустимого размера не отбрасывалось на парсинге.
	if h.maxSize > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxSize+(64<<10))
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: не удалось прочитать файл: %v", models.ErrInvalidInput, err))
		return
	}
	defer file.Close()

	result, err := h.importer.Import(c, &models.EquipmentImportDTO{
		RealmID:  realmID,
		FileName: header.Filename,
		File:     file,
		Actor:    actor,
	})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, result, result.Total)
}

// importTemplate отдаёт шаблон файла импорта (csv с примером строки).
// Заголовок — attachment: шаблон сохраняют и заполняют.
func (h *Handler) importTemplate(c *gin.Context) {
	content, err := buildImportTemplate()
	if err != nil {
		response.SendError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="equipment-import-template.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", content)
}

// buildImportTemplate собирает csv-шаблон. В начало файла кладётся BOM:
// без него русский Excel открывает csv в кракозябрах.
func buildImportTemplate() ([]byte, error) {
	var buf bytes.Buffer
	// BOM — три сырых байта, а не строка: в Go строка с этими кодами записалась
	// бы как utf-8-последовательность из шести байт, и Excel её не увидел бы.
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)
	writer.Comma = ';'
	if err := writer.Write(importTemplateHeaders); err != nil {
		return nil, fmt.Errorf("failed to build import template. error: %w", err)
	}
	if err := writer.Write(importTemplateExample); err != nil {
		return nil, fmt.Errorf("failed to build import template. error: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to build import template. error: %w", err)
	}
	return buf.Bytes(), nil
}
