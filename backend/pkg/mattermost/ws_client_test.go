package mattermost

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcceptPost закрепляет правило «это вопрос боту»: бот отвечает только на
// обычные сообщения пользователя. Регресс — цикл «бот — сам себе» (~4000 постов)
// и справка тому, кто лишь открыл диалог, — начинался с пропущенных сюда постов:
// собственного поста бота (bot_user_id разошёлся с токеном) и системного
// сообщения Mattermost (оно приходит от имени пользователя и не пусто).
func TestAcceptPost(t *testing.T) {
	const botID = "bot-id"

	tests := []struct {
		name string
		post *model.Post
		want bool
	}{
		{
			name: "обычное сообщение",
			post: &model.Post{UserId: "u1", Message: "привет"},
			want: true,
		},
		{
			name: "сообщение с вложениями без текста",
			post: &model.Post{UserId: "u1", FileIds: []string{"f1"}},
			want: true,
		},
		{
			name: "собственный пост бота",
			post: &model.Post{UserId: botID, Message: "справка"},
			want: false,
		},
		{
			name: "пост без автора",
			post: &model.Post{Message: "привет"},
			want: false,
		},
		{
			name: "системное сообщение",
			post: &model.Post{UserId: "u1", Type: model.PostTypeAddToChannel, Message: "добавил бота в канал"},
			want: false,
		},
		{
			name: "пост другого бота",
			post: &model.Post{UserId: "u1", Message: "привет", Props: model.StringInterface{model.PostPropsFromBot: "true"}},
			want: false,
		},
		{
			name: "пустое сообщение",
			post: &model.Post{UserId: "u1"},
			want: false,
		},
		{
			name: "отсутствующий пост",
			post: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AcceptPost(tt.post, botID))
		})
	}
}

// TestParseDirectAddedEvent закрепляет формат события direct_added: канал лежит
// не в data (его передаёт вызывающий из broadcast), а участники — в data. Именно
// по creator_id/teammate_id сервис отличает «пользователь открыл диалог» от
// «бот прислал уведомление».
func TestParseDirectAddedEvent(t *testing.T) {
	data := []byte(`{"creator_id":"user1","teammate_id":"bot1"}`)

	event, err := parseDirectAddedEvent("ch1", data)
	require.NoError(t, err)

	assert.Equal(t, DirectAddedEvent{ChannelID: "ch1", CreatorID: "user1", TeammateID: "bot1"}, event)
}

func TestParseDirectAddedEvent_InvalidJSON(t *testing.T) {
	_, err := parseDirectAddedEvent("ch1", []byte(`{`))
	require.Error(t, err)
}
