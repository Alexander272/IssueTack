package mattermost

import (
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
type InteractiveButton struct {
	Text    string
	Style   string
	URL     string
	Context map[string]string
}

// LinkAction describes a plain link attached to a card: unlike InteractiveButton
// it posts nothing back, it just opens the URL (a page of the web app).
type LinkAction struct {
	Text string
	URL  string
}

// AttachmentField is a short "label: value" line of a card.
type AttachmentField struct {
	Title string
	Value string
	Short bool
}

// Attachment is a card attached to a post. URL makes the whole card clickable,
// Actions adds buttons to it. Both web and mobile clients render the same card
// format, so a post with cards looks identical on a phone and on a desktop.
type Attachment struct {
	Title   string
	Text    string
	Color   string
	URL     string
	Fields  []AttachmentField
	Buttons []InteractiveButton
	Links   []LinkAction
}

// postProps renders a post's "attachments" prop. It is the shared builder for
// both new posts and interactive-action replies. Buttons of a card stay on that
// card — so a list of tickets can keep a separate action per ticket. Post-level
// buttons are a different case: they are built by buttonProps as a single card
// that holds only actions, and Mattermost renders such a card as the action row
// of the whole post.
func postProps(cards []Attachment) model.StringInterface {
	attachments := make([]model.StringInterface, 0, len(cards))

	for _, card := range cards {
		entry := model.StringInterface{}
		if card.Title != "" {
			entry["title"] = card.Title
		}
		if card.Text != "" {
			entry["text"] = card.Text
		}
		if card.Color != "" {
			entry["color"] = card.Color
		}
		if card.URL != "" {
			entry["url"] = card.URL
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
		if actions := cardActions(card); len(actions) > 0 {
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
func cardActions(card Attachment) []model.StringInterface {
	actions := make([]model.StringInterface, 0, len(card.Buttons)+len(card.Links))
	// Пустой style не отправляем: Mattermost подставит дефолтный сам.
	for _, btn := range card.Buttons {
		action := model.StringInterface{
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
	for _, l := range card.Links {
		actions = append(actions, model.StringInterface{
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

// buttonProps renders a post with a row of interactive action buttons and no
// cards. Kept as a separate builder so the common "message + button row" case
// does not grow an empty card when Buttons is empty.
func buttonProps(buttons []InteractiveButton) model.StringInterface {
	if len(buttons) == 0 {
		return nil
	}
	return postProps([]Attachment{{Buttons: buttons}})
}
