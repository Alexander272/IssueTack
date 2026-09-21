package models

import (
	"time"

	"github.com/google/uuid"
)

type RealmMattermost struct {
	RealmID       uuid.UUID `json:"realmId" db:"realm_id"`
	BotToken      string    `json:"-" db:"bot_token"`
	BotUserID     string    `json:"botUserId" db:"bot_user_id"`
	ChannelID     string    `json:"channelId" db:"channel_id"`
	WebhookSecret string    `json:"-" db:"webhook_secret"`
	IsActive      bool      `json:"isActive" db:"is_active"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

type RealmMattermostDTO struct {
	BotToken  string `json:"botToken" binding:"required"`
	ChannelID string `json:"channelId"`
}

// DialogOpenDTO — данные запроса на открытие диалога создания заявки
// Mattermost (нажатие кнопки «Создать заявку»).
type DialogOpenDTO struct {
	TriggerID    string            `json:"trigger_id"`
	UserID       string            `json:"user_id"`
	ChannelID    string            `json:"channel_id"`
	ButtonPostID string            `json:"post_id"`
	Context      map[string]string `json:"context"`
}

// InteractiveActionDTO — данные события нажатия интерактивной кнопки
// Mattermost (action в контексте, напр. «view_ticket»).
type InteractiveActionDTO struct {
	UserID    string            `json:"user_id"`
	ChannelID string            `json:"channel_id"`
	Context   map[string]string `json:"context"`
}
