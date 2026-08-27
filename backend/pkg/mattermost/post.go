package mattermost

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

// Post capability: create/update/delete messages in channels, optionally with
// an interactive action button.
type Post struct {
	client *Client
}

func NewPost(client *Client) *Post {
	return &Post{client: client}
}

type CreatePostDTO struct {
	ChannelID string
	Message   string
	Button    *InteractiveButton
}

// Create sends a message to the channel. If Button is set, an interactive
// action button is attached. Returns the created post id.
func (s *Post) Create(botToken string, dto CreatePostDTO) (string, error) {
	post := &model.Post{ChannelId: dto.ChannelID, Message: dto.Message}
	if dto.Button != nil {
		post.Props = buttonAttachments(*dto.Button)
	}

	created, err := s.client.CreatePost(botToken, post)
	if err != nil {
		return "", fmt.Errorf("create post: %w", err)
	}
	return created.Id, nil
}

// Reply builds an interactive-action reply post (e.g. updating the original
// message with an action button). Returns nil when there is nothing to send.
func (s *Post) Reply(message string, button *InteractiveButton) *model.Post {
	post := &model.Post{Message: message}
	if button != nil {
		post.Props = buttonAttachments(*button)
	}
	return post
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
