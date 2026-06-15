package models

import (
	"fmt"
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

func (dto *SubtaskDTO) GetChanges(old *Subtask) []*FieldChange {
	var changes []*FieldChange

	toStr := func(v interface{}) string {
		if v == nil {
			return "none"
		}
		return fmt.Sprintf("%v", v)
	}

	if dto.Title != old.Title {
		changes = append(changes, &FieldChange{ActionTitleChanged, old.Title, dto.Title})
	}
	if dto.Description != old.Description {
		changes = append(changes, &FieldChange{ActionDescriptionChanged, old.Description, dto.Description})
	}
	if dto.Status != old.Status {
		changes = append(changes, &FieldChange{ActionStatusChanged, toStr(old.Status), toStr(dto.Status)})
	}
	if dto.Priority != old.Priority {
		changes = append(changes, &FieldChange{ActionPriorityChanged, toStr(old.Priority), toStr(dto.Priority)})
	}
	if dto.AssigneeID != nil && (old.Assignee == nil || *dto.AssigneeID != old.Assignee.ID) {
		oldVal := "none"
		if old.Assignee != nil {
			oldVal = old.Assignee.ID.String()
		}
		changes = append(changes, &FieldChange{ActionAssigned, oldVal, dto.AssigneeID.String()})
	}
	if dto.DueDate != nil && (old.DueDate == nil || *dto.DueDate != *old.DueDate) {
		changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), toStr(dto.DueDate)})
	}

	return changes
}
