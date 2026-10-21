package services

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/gomutex/godocx"
	docs "github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/wml/ctypes"
	"github.com/gomutex/godocx/wml/stypes"
)

// actDocParties — подписи сторон для документа акта. Площадки здесь нет: место
// в акте не печатают (реквизит единицы, а не передачи), поэтому и не резолвим.
type actDocParties struct {
	Issuer    string
	Recipient string
}

// Таблица состава акта: колонки те же, что в снимке (см. actPosition), плюс
// порядковый номер — по нему сверяют позиции при подписании.
var actDocTableHeader = []string{"№", "Наименование", "Тип", "Серийный номер", "Инв. номер"}

// buildActDocx собирает docx акта по снимку состава. Документ не хранится на
// диске: он собирается по запросу, поэтому правка шаблона не требует пересборки
// старых актов. Библиотека godocx (MIT) закрывает OOXML, а не сам шаблон
// документа — состав документа задан здесь.
func buildActDocx(act *models.EquipmentAct, positions []actPosition, parties actDocParties) ([]byte, error) {
	if act == nil {
		return nil, models.ErrEquipmentActGeneration
	}

	doc, err := godocx.NewDocument()
	if err != nil {
		return nil, err
	}
	actDocArial(doc)

	actDocCentered(doc, "АКТ приёма-передачи оборудования", 14)
	actDocCentered(doc, "№ "+act.Number, 14)
	doc.AddEmptyParagraph()

	actDocLine(doc, "Дата составления:", act.IssuedAt.Format("02.01.2006"))
	actDocLine(doc, "Выдал:", actDocOrDash(parties.Issuer))
	actDocLine(doc, "Получил:", actDocOrDash(parties.Recipient))

	doc.AddEmptyParagraph()
	actDocLine(doc, "Состав передаваемого оборудования:", "")
	actDocTable(doc, positions)

	if act.Comment != "" {
		doc.AddEmptyParagraph()
		actDocMultiLine(doc, "Комментарий: "+act.Comment)
	}

	doc.AddEmptyParagraph()
	actDocBold(doc, "Подписи сторон")
	actDocLine(doc, "Выдал:", "______________________ / "+actDocOrDash(parties.Issuer))
	actDocLine(doc, "Получил:", "____________________ / "+actDocOrDash(parties.Recipient))

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// actDocArial ставит Arial шрифтом по умолчанию для всего документа. Шаблон
// godocx задаёт шрифт через тему (docDefaults → rFonts с *Theme=minorHAnsi),
// поэтому одной подмены имени мало: тематические ссылки нужно снять, иначе XML
// выведет и ascii, и asciiTheme, а тема в Word имеет приоритет. Стили заголовков
// (Heading*/Title) акт не использует, а TableGrid наследует обычный текст, так
// что правки docDefaults достаточно.
func actDocArial(doc *docs.RootDoc) {
	styles := doc.DocStyles
	if styles == nil || styles.DocDefaults == nil || styles.DocDefaults.RunProp == nil {
		return
	}
	runProp := styles.DocDefaults.RunProp.RunProp
	if runProp == nil {
		return
	}
	if runProp.Fonts == nil {
		runProp.Fonts = &ctypes.RunFonts{}
	}
	fonts := runProp.Fonts
	fonts.Ascii, fonts.HAnsi, fonts.CS = "Arial", "Arial", "Arial"
	fonts.AsciiTheme, fonts.HAnsiTheme, fonts.CSTheme, fonts.EastAsiaTheme = "", "", "", ""
}

// actDocCentered добавляет абзац по центру.
func actDocCentered(doc *docs.RootDoc, text string, size uint64) {
	p := doc.AddParagraph("")
	p.AddText(text).Bold(true).Size(size)
	p.Justification(stypes.JustificationCenter)
}

// actDocBold добавляет абзац с выделенным текстом.
func actDocBold(doc *docs.RootDoc, text string) {
	p := doc.AddParagraph("")
	p.AddText(text).Bold(true)
}

// actDocLine добавляет строку «Подпись: значение». Пустое значение даёт только
// подпись — снимок состава акта может быть без места и комментария.
func actDocLine(doc *docs.RootDoc, label, value string) {
	p := doc.AddParagraph("")
	p.AddText(label).Bold(true)
	if value != "" {
		p.AddText(" " + value)
	}
}

// actDocMultiLine добавляет абзац с переводами строк: в docx перенос — это
// отдельный элемент <w:br/>, а не символ в тексте.
func actDocMultiLine(doc *docs.RootDoc, text string) {
	lineBreak := stypes.BreakTypeTextWrapping
	p := doc.AddParagraph("")
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if i > 0 {
			p.AddRun().AddBreak(&lineBreak)
		}
		p.AddText(line)
	}
}

// actDocTable рисует таблицу состава. Стиль TableGrid берём из шаблона, который
// библиотека кладёт в новый документ, — иначе таблица вышла бы без рамок.
func actDocTable(doc *docs.RootDoc, positions []actPosition) {
	table := doc.AddTable()
	table.Style("TableGrid")

	header := table.AddRow()
	for _, title := range actDocTableHeader {
		p := header.AddCell().AddEmptyPara()
		p.AddText(title).Bold(true)
	}

	for i, position := range positions {
		row := table.AddRow()
		for _, cell := range []string{
			fmt.Sprintf("%d", i+1),
			position.Name,
			position.TypeName,
			actDocPtr(position.SerialNumber),
			actDocPtr(position.InventoryNumber),
		} {
			row.AddCell().AddParagraph(cell)
		}
	}
}

// actDocPtr разыменовывает необязательную подпись позиции.
func actDocPtr(value *string) string {
	if value == nil {
		return "—"
	}
	return actDocOrDash(*value)
}

// actDocOrDash заменяет пустое значение прочерком: пустая клетка в акте
// читается как «не заполнили», а не «нечего указать».
func actDocOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
