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
		assert.Equal(t, "c0a0", actions[0]["id"])
		integration, ok := actions[0]["integration"].(model.StringInterface)
		require.True(t, ok)
		assert.Equal(t, "https://app/api/dialog", integration["url"])
		assert.Equal(t, map[string]string{"realm_id": "r1"}, integration["context"])

		assert.Equal(t, "Мои заявки", actions[1]["name"])
		assert.Equal(t, "button", actions[1]["type"])
		assert.Equal(t, "c0a1", actions[1]["id"])
		_, hasStyle := actions[1]["style"]
		assert.False(t, hasStyle, "button without style must not send an empty style")
	})
}

func TestPostProps_Cards(t *testing.T) {
	t.Run("no cards means no props", func(t *testing.T) {
		assert.Nil(t, postProps(nil))
	})

	t.Run("card fields, title link and link action", func(t *testing.T) {
		props := postProps([]Attachment{{
			Title:     "№7 — Сломалась дверь",
			Color:     "#01579B",
			TitleLink: "https://app/tasks/abc",
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
		assert.Equal(t, "https://app/tasks/abc", card["title_link"])
		// url в схеме вложений нет: клиенты его не рисуют, карточка открывается
		// только по title_link.
		_, hasURL := card["url"]
		assert.False(t, hasURL, "attachment url не отправляем — его не знает ни один клиент")

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
		assert.Equal(t, "c0a0", links[0]["id"])
	})

	t.Run("empty optional fields are omitted", func(t *testing.T) {
		props := postProps([]Attachment{{Title: "Без ссылок"}})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 1)

		card := attachments[0]
		for _, key := range []string{"text", "color", "title_link", "url", "fields", "actions"} {
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

func TestPostProps_ActionIDs(t *testing.T) {
	// Регресс-гард: Mattermost ищет действие по id в attachments поста и берёт
	// первое совпадение, а роут /posts/{id}/actions/{action_id} принимает только
	// [A-Za-z0-9]+. Пустой или повторяющийся id ломает нажатие (404 на текущих
	// серверах, кнопка исчезает на 11.10+), поэтому проверяем оба свойства на
	// любом наборе карточек.
	t.Run("every action of every card has a unique id", func(t *testing.T) {
		props := postProps([]Attachment{
			{Title: "Первая", Buttons: []InteractiveButton{
				{Text: "Отменить заявку", URL: "https://app/api/action", ID: "t0123456789abcancel"},
				{Text: "Вернуть в работу", URL: "https://app/api/action"},
			}},
			{Title: "Вторая", Links: []LinkAction{{Text: "Открыть заявку", URL: "https://app/tasks/b"}}},
			{Title: "Третья"},
		})

		seen := map[string]bool{}
		for i, card := range propsAttachments(t, props) {
			raw, ok := card["actions"].([]model.StringInterface)
			if !ok {
				continue
			}
			for j, action := range raw {
				id, _ := action["id"].(string)
				require.NotEmptyf(t, id, "card %d action %d must have an id", i, j)
				assert.Regexp(t, `^[A-Za-z0-9]+$`, id, "id must match the action_id route")
				assert.Falsef(t, seen[id], "id %q must be unique within the post", id)
				seen[id] = true
			}
		}
		require.Len(t, seen, 3, "кнопки первой карточки, её ссылка и ссылка второй")
	})

	t.Run("explicit id wins and is stripped to the route charset", func(t *testing.T) {
		props := postProps([]Attachment{{
			Title: "Заявка",
			Buttons: []InteractiveButton{
				{Text: "Отменить заявку", URL: "https://app/api/action", ID: "t0123456789ab-cancel"},
			},
		}})

		actions, ok := propsAttachments(t, props)[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		require.Len(t, actions, 1)
		assert.Equal(t, "t0123456789abcancel", actions[0]["id"])
	})

	t.Run("positional id separates cards of one post", func(t *testing.T) {
		props := postProps([]Attachment{
			{Buttons: []InteractiveButton{{Text: "Первая", URL: "https://app/api/action"}}},
			{Buttons: []InteractiveButton{{Text: "Вторая", URL: "https://app/api/action"}}},
		})

		attachments := propsAttachments(t, props)
		require.Len(t, attachments, 2)

		first, ok := attachments[0]["actions"].([]model.StringInterface)
		require.True(t, ok)
		second, ok := attachments[1]["actions"].([]model.StringInterface)
		require.True(t, ok)
		assert.Equal(t, "c0a0", first[0]["id"])
		assert.Equal(t, "c1a0", second[0]["id"])
		assert.NotEqual(t, first[0]["id"], second[0]["id"])
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
