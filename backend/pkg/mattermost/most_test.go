package mattermost

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// propsAttachments достаёт карточки из props поста.
func propsAttachments(t *testing.T, props model.StringInterface) []model.StringInterface {
	t.Helper()
	if props == nil {
		return nil
	}
	raw, ok := props["attachments"]
	require.True(t, ok, "props must contain attachments")
	attachments, ok := raw.([]model.StringInterface)
	require.True(t, ok, "attachments must be a slice")
	return attachments
}

func TestButtonProps_SingleAndSeveral(t *testing.T) {
	t.Run("no buttons means no props", func(t *testing.T) {
		assert.Nil(t, buttonProps(nil))
		assert.Nil(t, buttonProps([]InteractiveButton{}))
	})

	t.Run("two buttons land in one action row", func(t *testing.T) {
		props := buttonProps([]InteractiveButton{
			{Text: "Создать заявку", Style: "primary", URL: "https://app/api/dialog", Context: map[string]string{"realm_id": "r1"}},
			{Text: "Мои заявки", URL: "https://app/api/action", Context: map[string]string{"action": "my_tickets", "realm_id": "r1"}},
		})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 1, "actions must not be duplicated into a card per button")

		actions, ok := attachments[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, actions, 2)

		assert.Equal(t, "Создать заявку", actions[0]["name"])
		assert.Equal(t, "button", actions[0]["type"])
		assert.Equal(t, "primary", actions[0]["style"])
		integration, ok := actions[0]["integration"].(model.StringInterface)
		require.True(t, ok)
		assert.Equal(t, "https://app/api/dialog", integration["url"])
		assert.Equal(t, map[string]string{"realm_id": "r1"}, integration["context"])

		assert.Equal(t, "Мои заявки", actions[1]["name"])
		assert.Equal(t, "button", actions[1]["type"])
		_, hasStyle := actions[1]["style"]
		assert.False(t, hasStyle, "button without style must not send an empty style")
	})
}

func TestPostProps_Cards(t *testing.T) {
	t.Run("no cards means no props", func(t *testing.T) {
		assert.Nil(t, postProps(nil))
	})

	t.Run("card fields, url and link action", func(t *testing.T) {
		props := postProps([]Attachment{{
			Title: "№7 — Сломалась дверь",
			Color: "#01579B",
			URL:   "https://app/tasks/abc",
			Fields: []AttachmentField{
				{Title: "Статус", Value: "Новая", Short: true},
				{Title: "Создана", Value: "05.09.2026", Short: true},
			},
			Links: []LinkAction{{Text: "Открыть заявку", URL: "https://app/tasks/abc"}},
		}})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 1)

		card := attachments[0]
		assert.Equal(t, "№7 — Сломалась дверь", card["title"])
		assert.Equal(t, "#01579B", card["color"])
		assert.Equal(t, "https://app/tasks/abc", card["url"])

		fields, ok := card["fields"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, fields, 2)
		assert.Equal(t, "Статус", fields[0]["title"])
		assert.Equal(t, "Новая", fields[0]["value"])
		assert.Equal(t, true, fields[0]["short"])

		links, ok := card["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, links, 1)
		assert.Equal(t, "Открыть заявку", links[0]["name"])
		assert.Equal(t, "link", links[0]["type"])
		assert.Equal(t, "https://app/tasks/abc", links[0]["url"])
	})

	t.Run("empty optional fields are omitted", func(t *testing.T) {
		props := postProps([]Attachment{{Title: "Без ссылок"}})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 1)

		card := attachments[0]
		for _, key := range []string{"text", "color", "url", "fields", "actions"} {
			_, has := card[key]
			assert.Falsef(t, has, "key %q must be omitted", key)
		}
	})

	t.Run("card buttons stay on their own card", func(t *testing.T) {
		props := postProps([]Attachment{
			{Title: "Первая", Buttons: []InteractiveButton{{Text: "Создать заявку", URL: "https://app/api/dialog", Context: map[string]string{"realm_id": "r1"}}}},
			{Title: "Вторая"},
		})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 2, "кнопки карточки не превращаются в отдельный ряд")

		actions, ok := attachments[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, actions, 1)
		assert.Equal(t, "Создать заявку", actions[0]["name"])
		assert.Equal(t, "button", actions[0]["type"])
		integration, ok := actions[0]["integration"].(model.StringInterface)
		require.True(t, ok)
		assert.Equal(t, map[string]string{"realm_id": "r1"}, integration["context"])

		_, hasActions := attachments[1]["actions"]
		assert.False(t, hasActions, "у карточки без кнопок actions не отправляем")
		assert.Equal(t, "Вторая", attachments[1]["title"])
	})

	t.Run("card buttons and links share one action row", func(t *testing.T) {
		props := postProps([]Attachment{{
			Title:   "С кнопками",
			Buttons: []InteractiveButton{{Text: "Отменить заявку", Style: "danger", URL: "https://app/api/action"}},
			Links:   []LinkAction{{Text: "Открыть заявку", URL: "https://app/tasks/abc"}},
		}})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 1)

		actions, ok := attachments[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, actions, 2)
		assert.Equal(t, "Отменить заявку", actions[0]["name"])
		assert.Equal(t, "danger", actions[0]["style"])
		assert.Equal(t, "Открыть заявку", actions[1]["name"])
		assert.Equal(t, "link", actions[1]["type"])
	})
}

func TestPostReply(t *testing.T) {
	t.Run("message only", func(t *testing.T) {
		post := (&Post{}).Reply("просто текст")
		assert.Equal(t, "просто текст", post.Message)
		assert.Nil(t, post.Props)
	})

	t.Run("one button", func(t *testing.T) {
		post := (&Post{}).Reply("нажмите", InteractiveButton{Text: "Открыть", URL: "https://app/api/action"})

		attachments := propsAttachments(t, post.Props)
		require.Len(t, attachments, 1)
		actions, ok := attachments[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, actions, 1)
		assert.Equal(t, "Открыть", actions[0]["name"])
	})
}

func TestPostProps(t *testing.T) {
	props := postProps([]Attachment{{Title: "№1 — Заявка", Text: "Описание"}})

	attachments := propsAttachments(t, props)
	require.Len(t, attachments, 1)
	assert.Equal(t, "№1 — Заявка", attachments[0]["title"])
	assert.Equal(t, "Описание", attachments[0]["text"])
}
