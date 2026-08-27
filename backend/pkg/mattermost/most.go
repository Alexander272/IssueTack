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

// InteractiveButton describes an action button attached to a post.
type InteractiveButton struct {
	Text    string
	Style   string
	URL     string
	Context map[string]string
}

// toAttachments renders a post's "attachments" prop containing a single
// interactive action button. It is the shared builder for both new posts
// and interactive-action replies.
func buttonAttachments(button InteractiveButton) model.StringInterface {
	return model.StringInterface{
		"attachments": []model.StringInterface{{
			"actions": []model.StringInterface{{
				"name":  button.Text,
				"type":  "button",
				"style": button.Style,
				"integration": model.StringInterface{
					"url":     button.URL,
					"context": button.Context,
				},
			}},
		}},
	}
}
