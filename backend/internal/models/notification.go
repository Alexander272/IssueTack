package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"userId" db:"user_id"`
	Type      string          `json:"type" db:"type"`
	Title     string          `json:"title" db:"title"`
	Body      string          `json:"body" db:"body"`
	Data      json.RawMessage `json:"data" db:"data"`
	IsRead    bool            `json:"isRead" db:"is_read"`
	CreatedAt time.Time       `json:"createdAt" db:"created_at"`
}

type CreateNotificationDTO struct {
	UserID uuid.UUID       `json:"userId"`
	Type   string          `json:"type"`
	Title  string          `json:"title"`
	Body   string          `json:"body"`
	Data   json.RawMessage `json:"data"`
}

type NotificationSettings struct {
	UserID   uuid.UUID       `json:"userId" db:"user_id"`
	Settings json.RawMessage `json:"settings" db:"settings"`
}
