package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Text       string    `json:"text" db:"text"`
	UserID     uuid.UUID `json:"userId" db:"user_id"`
	TicketID   uuid.UUID `json:"ticketId" db:"ticket_id"`
	IsInternal bool      `json:"isInternal" db:"is_internal"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
}
