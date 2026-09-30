package mattermost

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

// Post capability: create/update/delete messages in channels, optionally with
// interactive action buttons and cards.
type Post struct {
	client *Client
}

func NewPost(client *Client) *Post {
	return &Post{client: client}
}

type CreatePostDTO struct {
	ChannelID   string
	Message     string
	Buttons     []InteractiveButton
	Attachments []Attachment
}

// Create sends a message to the channel. Buttons are rendered as an action row
// above the cards, Attachments as cards under the text. Returns the created post id.
func (s *Post) Create(botToken string, dto CreatePostDTO) (string, error) {
	post := &model.Post{ChannelId: dto.ChannelID, Message: dto.Message}
	// Кнопки поста — ведущая карточка, которую Mattermost рисует рядом над
	// остальными. props собираем одним вызовом postProps: иначе карточки
	// затирают кнопки (и наоборот).
	cards := dto.Attachments
	if len(dto.Buttons) > 0 {
		cards = append([]Attachment{{Buttons: dto.Buttons}}, cards...)
	}
	if props := postProps(cards); props != nil {
		post.Props = props
	}

	created, err := s.client.CreatePost(botToken, post)
	if err != nil {
		return "", fmt.Errorf("create post: %w", err)
	}
	return created.Id, nil
}

// Reply builds an interactive-action reply post (e.g. updating the original
// message with an action button). Returns nil when there is nothing to send.
func (s *Post) Reply(message string, buttons ...InteractiveButton) *model.Post {
	post := &model.Post{Message: message}
	if props := buttonProps(buttons); props != nil {
		post.Props = props
	}
	return post
}

// UpdateCards replaces the text and the cards of an existing post. Ответ на
// диалог не умеет обновлять посты (SubmitDialogResponse знает только
// errors/cancel), поэтому список после возврата заявки в работу правим так.
func (s *Post) UpdateCards(botToken, postID, message string, cards []Attachment) error {
	patch := &model.PostPatch{Message: &message}
	if props := postProps(cards); props != nil {
		patch.Props = &props
	} else {
		empty := model.StringInterface{}
		patch.Props = &empty
	}
	return s.client.UpdatePost(botToken, postID, patch)
}

// UpdateMessage replaces the message text of an existing post (keeps other props).
func (s *Post) UpdateMessage(botToken, postID, message string) error {
	msg := message
	return s.client.UpdatePost(botToken, postID, &model.PostPatch{Message: &msg})
}

// Delete removes an existing post.
func (s *Post) Delete(botToken, postID string) error {
	return s.client.DeletePost(botToken, postID)
}
