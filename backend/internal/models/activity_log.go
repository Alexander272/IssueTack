package models

import (
	"time"

	"github.com/google/uuid"
)

type ActivityLog struct {
	ID       uuid.UUID    `json:"id" db:"id"`
	TicketID uuid.UUID    `json:"ticketId" db:"ticket_id"`
	UserID   uuid.UUID    `json:"userId" db:"user_id"` // Кто совершил действие (напр. Менеджер)
	UserName string       `json:"userName" db:"user_name"`
	Type     ActivityType `json:"type" db:"type"` // Тип события

	// Поля для хранения изменений "было -> стало"
	OldValue *string `json:"oldValue" db:"old_value"`
	NewValue *string `json:"newValue" db:"new_value"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type GetLogsDTO struct {
	TicketID string `json:"ticketId" db:"ticket_id"`
}

type ActivityLogDTO struct {
	ID       uuid.UUID    `json:"id" db:"id"`
	TicketID uuid.UUID    `json:"ticketId" db:"ticket_id"`
	UserID   uuid.UUID    `json:"userId" db:"user_id"`
	Type     ActivityType `json:"type" db:"type"`
	OldValue *string      `json:"oldValue" db:"old_value"`
	NewValue *string      `json:"newValue" db:"new_value"`
}
