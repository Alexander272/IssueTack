package services

import (
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeModifierKeywords(t *testing.T) {
	got := normalizeModifierKeywords([]string{
		"  Принтер  ", "", "   ", "принтер", "Ёлка", "елка", "Wi-Fi", "wi fi",
		"Ошибка", "ошибки", "ошибку", "ошибкой",
	})
	assert.Equal(t, []string{"Принтер", "Ёлка", "Wi-Fi", "Ошибка"}, got)
	assert.NotNil(t, normalizeModifierKeywords(nil), "колонка keywords NOT NULL — nil-слайс писать нельзя")
}

func TestStemModifierText(t *testing.T) {
	assert.Equal(t, "ошибк ошибк лага", stemModifierText("ошибки ошибку лагать"))
	// Часть форм даёт другую основу: такие слова добавляются в ключи отдельно.
	assert.Equal(t, "ошибок", stemModifierText("ошибок"))
}

func TestMatchModifierByKeywords(t *testing.T) {
	cases := []struct {
		name        string
		title       string
		description string
		modifiers   []*models.CategoryModifier
		wantID      string
		wantKeyword string
	}{
		{
			name:        "count_beats_longest_keyword",
			title:       "Заменил картридж и тонер в принтере",
			description: "Нужна замена расходников",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("print"), Name: "Печать", IsActive: true, Keywords: []string{"принтер", "картридж", "тонер"}},
				{ID: idForTest("hp"), Name: "Расходники HP", IsActive: true, Keywords: []string{"тонер картридж для hp"}},
			},
			wantID:      "print",
			wantKeyword: "картридж",
		},
		{
			name:  "longest_keyword_tie_break",
			title: "Поменять тонер картридж",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("short"), Name: "Короткий", IsActive: true, Keywords: []string{"картридж"}},
				{ID: idForTest("long"), Name: "Длинный", IsActive: true, Keywords: []string{"тонер картридж"}},
			},
			wantID:      "long",
			wantKeyword: "тонер картридж",
		},
		{
			name:  "sort_order_tie_break",
			title: "Не работает принтер",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("second"), Name: "Б", IsActive: true, SortOrder: 2, Keywords: []string{"принтер"}},
				{ID: idForTest("first"), Name: "А", IsActive: true, SortOrder: 1, Keywords: []string{"принтер"}},
			},
			wantID:      "first",
			wantKeyword: "принтер",
		},
		{
			name:        "normalization_and_word_boundaries",
			title:       "Ёлка WI-FI в принтерной",
			description: "Принтер, МФУ — не отвечает",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("wifi"), Name: "Сеть", IsActive: true, Keywords: []string{"елка wi fi"}},
				{ID: idForTest("print"), Name: "Печать", IsActive: true, Keywords: []string{"принтер"}},
			},
			wantID:      "wifi",
			wantKeyword: "елка wi fi",
		},
		{
			name:        "inactive_and_empty_ignored",
			title:       "Принтер не печатает",
			description: "Картридж",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("inactive"), Name: "Выключенная", IsActive: false, Keywords: []string{"принтер", "картридж"}},
				{ID: idForTest("empty"), Name: "Без ключей", IsActive: true},
			},
		},
		{
			name:        "word_forms_noun",
			title:       "Ошибки при печати",
			description: "Вижу ошибку с утра",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("errors"), Name: "Ошибки", IsActive: true, Keywords: []string{"ошибка"}},
			},
			wantID:      "errors",
			wantKeyword: "ошибка",
		},
		{
			name:        "word_forms_verb",
			title:       "База лагала",
			description: "И будет лагать",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("lag"), Name: "Лаги", IsActive: true, Keywords: []string{"лагает"}},
			},
			wantID:      "lag",
			wantKeyword: "лагает",
		},
		{
			name:  "stem_dedup_does_not_inflate_count",
			title: "Ошибки и печать картриджа",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("forms"), Name: "Формы", IsActive: true, Keywords: []string{"ошибка", "ошибки", "ошибку"}},
				{ID: idForTest("two"), Name: "Два ключа", IsActive: true, Keywords: []string{"ошибка", "печать картриджа"}},
			},
			wantID:      "two",
			wantKeyword: "печать картриджа",
		},
		{
			name:        "no_match",
			title:       "Принтерная лента",
			description: "Заказ на поставку",
			modifiers: []*models.CategoryModifier{
				{ID: idForTest("print"), Name: "Печать", IsActive: true, Keywords: []string{"принтер"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, keyword := matchModifierByKeywords(tc.title, tc.description, tc.modifiers)
			if tc.wantID == "" {
				assert.Nil(t, got)
				assert.Empty(t, keyword)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, idForTest(tc.wantID), got.ID)
			assert.Equal(t, tc.wantKeyword, keyword)
		})
	}
}

// idForTest — детерминированный uuid из короткой метки, чтобы в таблице кейсов
//
//	idForTest("print") === ожидаемый ID выбранной подкатегории.
func idForTest(label string) (id uuid.UUID) {
	for i := 0; i < len(label); i++ {
		id[i%len(id)] = byte(label[i])
	}
	return id
}
