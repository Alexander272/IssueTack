package mattermost

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

// Dialog capability: open interactive dialogs from action buttons.
type Dialog struct {
	client  *Client
	baseURL string
}

func NewDialog(client *Client, baseURL string) *Dialog {
	return &Dialog{client: client, baseURL: baseURL}
}

// DialogElement describes one dialog field.
type DialogElement struct {
	DisplayName string
	Name        string
	Type        string
	MaxLength   int
	Options     []*model.PostActionOptions
	Placeholder string
}

// OpenRequest carries everything needed to open a dialog for a button press.
type OpenRequest struct {
	TriggerID    string
	RealmID      string // used to build the submission callback URL
	Title        string
	Introduction string
	SubmitLabel  string
	Elements     []DialogElement
	State        string
}

func toModelElements(els []DialogElement) []model.DialogElement {
	out := make([]model.DialogElement, 0, len(els))
	for _, e := range els {
		d := model.DialogElement{
			DisplayName: e.DisplayName,
			Name:        e.Name,
			Type:        e.Type,
			MaxLength:   e.MaxLength,
			Placeholder: e.Placeholder,
		}
		if e.Options != nil {
			d.Options = e.Options
		}
		out = append(out, d)
	}
	return out
}

// Open opens an interactive dialog directed at the submission endpoint.
func (s *Dialog) Open(botToken string, req OpenRequest) error {
	reqURL := fmt.Sprintf("%s/api/v1/mattermost/dialog/%s", s.baseURL, req.RealmID)
	modelReq := &model.OpenDialogRequest{
		TriggerId: req.TriggerID,
		URL:       reqURL,
		Dialog: model.Dialog{
			CallbackId:       req.RealmID,
			Title:            req.Title,
			IntroductionText: req.Introduction,
			Elements:         toModelElements(req.Elements),
			SubmitLabel:      req.SubmitLabel,
			NotifyOnCancel:   true,
			State:            req.State,
		},
	}
	return s.client.OpenDialog(botToken, modelReq)
}
