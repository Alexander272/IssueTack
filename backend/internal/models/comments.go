package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID        `json:"id" db:"id"`
	Text       string           `json:"text" db:"text"`
	UserID     uuid.UUID        `json:"userId" db:"user_id"`
	TicketID   uuid.UUID        `json:"ticketId" db:"ticket_id"`
	IsInternal bool             `json:"isInternal" db:"is_internal"`
	Type       string           `json:"type" db:"type"`
	CreatedAt  time.Time        `json:"createdAt" db:"created_at"`
	User       *UserShort       `json:"user"`
}

type CreateCommentDTO struct {
	Text       string    `json:"text" binding:"required"`
	TicketID   uuid.UUID `json:"-" binding:"-"`
	IsInternal bool      `json:"isInternal"`
	Type       string    `json:"type"`
	UserID     uuid.UUID `json:"-"`
	Realm      string    `json:"-"`
}

type DeleteCommentDTO struct {
	ID      uuid.UUID `json:"-"`
	ActorID uuid.UUID `json:"-"`
}
