package models

import (
	"time"

	"github.com/google/uuid"
)

type Subtask struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	TicketID    uuid.UUID   `json:"ticketId" db:"ticket_id"`
	Title       string      `json:"title" db:"title"`
	Description string      `json:"description" db:"description"`
	Status      TicketStatus `json:"status" db:"status"`
	Priority    Priority    `json:"priority" db:"priority"`
	Assignee    *UserShort  `json:"assignee,omitempty"`
	DueDate     *time.Time  `json:"dueDate" db:"due_date"`
	ClosedAt    *time.Time  `json:"closedAt" db:"closed_at"`
	SortOrder   int         `json:"sortOrder" db:"sort_order"`
	CreatedAt   time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time   `json:"updatedAt" db:"updated_at"`
}

type GetSubtaskDTO struct {
	ID uuid.UUID `json:"id"`
}

type GetSubtasksByTicketDTO struct {
	TicketID uuid.UUID `json:"ticketId"`
}

type SubtaskDTO struct {
	ID          uuid.UUID   `json:"id"`
	TicketID    uuid.UUID   `json:"ticketId"`
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Status      TicketStatus `json:"status"`
	Priority    Priority    `json:"priority"`
	AssigneeID  *uuid.UUID  `json:"assigneeId,omitempty"`
	DueDate     *time.Time  `json:"dueDate,omitempty"`
	SortOrder   int         `json:"sortOrder"`
}

type DelSubtaskDTO struct {
	ID    uuid.UUID `json:"id"`
	Actor Actor
}
