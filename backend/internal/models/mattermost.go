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

type MattermostUserLink struct {
	UserID   uuid.UUID `json:"userId" db:"user_id"`
	MmUserID string    `json:"mmUserId" db:"mm_user_id"`
}

type MattermostUserLinkDTO struct {
	MmUserID string `json:"mmUserId" binding:"required"`
}
