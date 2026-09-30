package mattermost

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// Most is the Mattermost messaging capability layer: it knows how to send
// messages, open dialogs and communicate with users, without any domain logic.
// It composes the low-level *Client into ergonomic primitives used by services.
type Most struct {
	*Post
	*Dialog
	*DM
	Client *Client
}

type MostConfig struct {
	ServerURL string // Mattermost server URL for the low-level client
	BaseURL   string // external base URL used to build interactive callback URLs
}

func NewMost(cfg MostConfig) *Most {
	client := NewClient(cfg.ServerURL)
	return &Most{
		Post:   NewPost(client),
		Dialog: NewDialog(client, cfg.BaseURL),
		DM:     NewDM(client),
		Client: client,
	}
}

// InteractiveButton describes an action button attached to a post. URL points
// to our backend: Mattermost posts the press there with the Context, so the
// action is dispatched server-side.
//
// ID is the action_id Mattermost uses to route the press: the server looks the
// action up in the post's own attachments (Post.GetAction) and sends that
// action's integration URL and Context. It is therefore mandatory and must be
// unique inside the post. Left empty, postProps assigns a positional one.
type InteractiveButton struct {
	Text    string
	Style   string
	URL     string
	Context map[string]string
	ID      string
}

// LinkAction describes a plain link attached to a card: unlike InteractiveButton
// it posts nothing back, it just opens the URL (a page of the web app). ID is
// handled exactly like InteractiveButton.ID.
type LinkAction struct {
	Text string
	URL  string
	ID   string
}

// AttachmentField is a short "label: value" line of a card.
type AttachmentField struct {
	Title string
	Value string
	Short bool
}

// Attachment is a card attached to a post. TitleLink makes the title a link to
// the ticket, Actions adds buttons to it. Both web and mobile clients render
// the same card format, so a post with cards looks identical on a phone and on
// a desktop.
//
// TitleLink, not "url": there is no whole-card link in the attachment schema —
// clients only know title_link/author_link, so a card is opened by its title.
type Attachment struct {
	Title     string
	TitleLink string
	Text      string
	Color     string
	Fields    []AttachmentField
	Buttons   []InteractiveButton
	Links     []LinkAction
}

// postProps renders a post's "attachments" prop. It is the shared builder for
// both new posts and interactive-action replies. Buttons of a card stay on that
// card — so a list of tickets can keep a separate action per ticket. Post-level
// buttons are a different case: they are built by buttonProps as a single card
// that holds only actions, and Mattermost renders such a card as the action row
// of the whole post.
func postProps(cards []Attachment) model.StringInterface {
	attachments := make([]model.StringInterface, 0, len(cards))

	for cardIdx, card := range cards {
		entry := model.StringInterface{}
		if card.Title != "" {
			entry["title"] = card.Title
		}
		if card.TitleLink != "" {
			entry["title_link"] = card.TitleLink
		}
		if card.Text != "" {
			entry["text"] = card.Text
		}
		if card.Color != "" {
			entry["color"] = card.Color
		}
		if len(card.Fields) > 0 {
			fields := make([]model.StringInterface, 0, len(card.Fields))
			for _, f := range card.Fields {
				fields = append(fields, model.StringInterface{
					"title": f.Title,
					"value": f.Value,
					"short": f.Short,
				})
			}
			entry["fields"] = fields
		}
		if actions := cardActions(card, cardIdx); len(actions) > 0 {
			entry["actions"] = actions
		}
		// Карточка, у которой кроме кнопок ничего нет, — это ряд кнопок
		// (см. buttonProps), отдельную пустую карточку в posts не отправляем.
		if len(entry) > 0 {
			attachments = append(attachments, entry)
		}
	}

	if len(attachments) == 0 {
		return nil
	}
	return model.StringInterface{"attachments": attachments}
}

// cardActions builds the action row of a single card: interactive buttons
// first, then plain links. Both kinds are rendered by Mattermost as the buttons
// of that card, so a card can offer its own actions without affecting others.
func cardActions(card Attachment, cardIdx int) []model.StringInterface {
	actions := make([]model.StringInterface, 0, len(card.Buttons)+len(card.Links))
	// Пустой style не отправляем: Mattermost подставит дефолтный сам.
	for i, btn := range card.Buttons {
		action := model.StringInterface{
			"id":   actionID(btn.ID, cardIdx, i),
			"name": btn.Text,
			"type": "button",
			"integration": model.StringInterface{
				"url":     btn.URL,
				"context": btn.Context,
			},
		}
		if btn.Style != "" {
			action["style"] = btn.Style
		}
		actions = append(actions, action)
	}
	for i, l := range card.Links {
		actions = append(actions, model.StringInterface{
			"id":   actionID(l.ID, cardIdx, len(card.Buttons)+i),
			"name": l.Text,
			"type": "link",
			"url":  l.URL,
		})
	}
	if len(actions) == 0 {
		return nil
	}
	return actions
}

// actionID returns the action_id for a button or a link of a card.
//
// Two rules make this mandatory rather than cosmetic:
//   - Mattermost routes the press by id: the client sends
//     POST /posts/{post_id}/actions/{action_id} (the route accepts
//     [A-Za-z0-9]+ only) and the server resolves it against the post's own
//     attachments. An empty id 404s on current servers and makes the control
//     disappear entirely on Mattermost 11.10+, whose Interactive Messages
//     renderer drops actions without an id.
//   - Post.GetAction returns the FIRST matching action, so ids must be unique
//     within the post — otherwise a click on the third card would be dispatched
//     with the first card's context.
//
// An explicit id wins; a positional one is the fallback and is unique by
// construction. Explicit ids are stripped down to [A-Za-z0-9] so a caller can
// pass a UUID or a readable name without breaking the route.
func actionID(explicit string, cardIdx, actionIdx int) string {
	var b strings.Builder
	for _, r := range explicit {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		return b.String()
	}
	return fmt.Sprintf("c%da%d", cardIdx, actionIdx)
}

// buttonProps renders a post with a row of interactive action buttons and no
// cards. Kept as a separate builder so the common "message + button row" case
// does not grow an empty card when Buttons is empty.
func buttonProps(buttons []InteractiveButton) model.StringInterface {
	if len(buttons) == 0 {
		return nil
	}
	return postProps([]Attachment{{Buttons: buttons}})
}
