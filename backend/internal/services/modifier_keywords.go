package services

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/kljensen/snowball/russian"
)

// normalizeModifierKeywordText приводит текст и ключ к одному виду для поиска:
// нижний регистр, ё→е, все разделители (пробелы, дефисы, знаки препинания)
// превращаются в один пробел. Поэтому «Wi-Fi» совпадает с «wi fi», а фраза
// «тонер картридж» — с «тонер-картридж». Границы слова проверяются отдельно,
// чтобы «принтер» не совпал с «принтерная».
func normalizeModifierKeywordText(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), "ё", "е")
	var b strings.Builder
	pendingSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false
			b.WriteRune(r)
			continue
		}
		pendingSpace = true
	}
	return b.String()
}

// stemModifierText приводит каждое слово нормализованного текста к основе
// (снежный ком для русского). Это позволяет ключу «ошибка» совпасть с
// «ошибки/ошибку/ошибкой», а «лагает» — с «лагал/лагать». Стемминг, а не
// полная лемматизация: часть форм (например, «ошибок») даёт другую основу,
// поэтому такие слова при необходимости добавляются в ключи отдельно.
func stemModifierText(text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = russian.Stem(word, false)
	}
	return strings.Join(words, " ")
}

// normalizeModifierKeywords чистит ключи при сохранении подкатегории:
// обрезка, отсев пустых и дублей (сравнение по основе слова, поэтому
// «ошибка» и «ошибки» — один ключ). В БД остаётся первый введённый вариант.
// Возвращает не-nil срез — колонка keywords объявлена NOT NULL.
func normalizeModifierKeywords(keywords []string) []string {
	out := make([]string, 0, len(keywords))
	seen := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		raw := strings.TrimSpace(keyword)
		normalized := normalizeModifierKeywordText(raw)
		if normalized == "" {
			continue
		}
		stemmed := stemModifierText(normalized)
		if stemmed == "" {
			continue
		}
		if _, exists := seen[stemmed]; exists {
			continue
		}
		seen[stemmed] = struct{}{}
		out = append(out, raw)
	}
	return out
}

// modifierTextHasKeyword ищет ключ по границам «слова». В нормализованном
// тексте разделитель — только ASCII-пробел, поэтому проверка байтов по краям
// корректна и для кириллицы (ASCII-\b не понимает не-латиницу).
func modifierTextHasKeyword(text, keyword string) bool {
	if keyword == "" {
		return false
	}
	for offset := 0; offset < len(text); {
		i := strings.Index(text[offset:], keyword)
		if i < 0 {
			return false
		}
		start := offset + i
		end := start + len(keyword)
		if (start == 0 || text[start-1] == ' ') && (end == len(text) || text[end] == ' ') {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[start:])
		offset = start + size
	}
	return false
}

// modifierKeywordMatch — оценка одной подкатегории: сколько её ключей вошло
// в текст и какой из совпавших ключей самый длинный.
type modifierKeywordMatch struct {
	modifier *models.CategoryModifier
	keyword  string
	count    int
	longest  int
}

func modifierMatchBetter(a, b *modifierKeywordMatch) bool {
	if b == nil {
		return true
	}
	if a.count != b.count {
		return a.count > b.count
	}
	if a.longest != b.longest {
		return a.longest > b.longest
	}
	if a.modifier.SortOrder != b.modifier.SortOrder {
		return a.modifier.SortOrder < b.modifier.SortOrder
	}
	return a.modifier.Name < b.modifier.Name
}

// matchModifierByKeywords выбирает подкатегорию по словам заголовка и описания:
// сначала больше уникальных совпавших ключей, затем самый длинный ключ,
// затем sort_order и имя. Ключ и текст сравниваются по основам слов, поэтому
// «ошибка» ловит «ошибки/ошибку/ошибкой». Для сравнения длин и в результат
// возвращается исходная (нормализованная) формулировка ключа, а не основа.
// Неактивные подкатегории и чужие категории отфильтровываются вызывающим.
func matchModifierByKeywords(title, description string, modifiers []*models.CategoryModifier) (*models.CategoryModifier, string) {
	text := normalizeModifierKeywordText(title + " " + description)
	if text == "" {
		return nil, ""
	}
	stemmedText := stemModifierText(text)
	if stemmedText == "" {
		return nil, ""
	}

	var best *modifierKeywordMatch
	for _, modifier := range modifiers {
		if modifier == nil || !modifier.IsActive || len(modifier.Keywords) == 0 {
			continue
		}
		count, longest := 0, ""
		seen := make(map[string]struct{}, len(modifier.Keywords))
		for _, keyword := range modifier.Keywords {
			normalized := normalizeModifierKeywordText(keyword)
			if normalized == "" {
				continue
			}
			stemmed := stemModifierText(normalized)
			if stemmed == "" {
				continue
			}
			if _, exists := seen[stemmed]; exists {
				continue
			}
			seen[stemmed] = struct{}{}
			if !modifierTextHasKeyword(stemmedText, stemmed) {
				continue
			}
			count++
			if utf8.RuneCountInString(normalized) > utf8.RuneCountInString(longest) {
				longest = normalized
			}
		}
		if count == 0 {
			continue
		}
		candidate := &modifierKeywordMatch{
			modifier: modifier,
			keyword:  longest,
			count:    count,
			longest:  utf8.RuneCountInString(longest),
		}
		if modifierMatchBetter(candidate, best) {
			best = candidate
		}
	}
	if best == nil {
		return nil, ""
	}
	return best.modifier, best.keyword
}
