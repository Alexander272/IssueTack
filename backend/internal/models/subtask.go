package models

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

type Subtask struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	TicketID    uuid.UUID    `json:"ticketId" db:"ticket_id"`
	Title       string       `json:"title" db:"title"`
	Description string       `json:"description" db:"description"`
	Status      TicketStatus `json:"status" db:"status"`
	Priority    Priority     `json:"priority" db:"priority"`
	Assignee    *UserShort   `json:"assignee,omitempty"`
	DueDate     *time.Time   `json:"dueDate" db:"due_date"`
	ClosedAt    *time.Time   `json:"closedAt" db:"closed_at"`
	SortOrder   int          `json:"sortOrder" db:"sort_order"`
	CreatedAt   time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time    `json:"updatedAt" db:"updated_at"`
}

type GetSubtaskDTO struct {
	ID uuid.UUID `json:"id"`
}

type GetSubtasksByTicketDTO struct {
	TicketID uuid.UUID `json:"ticketId"`
}

type SubtaskDTO struct {
	ID          uuid.UUID `json:"id"`
	Actor       *Actor
	TicketID    uuid.UUID    `json:"ticketId"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TicketStatus `json:"status"`
	Priority    Priority     `json:"priority"`
	AssigneeID  *uuid.UUID   `json:"assigneeId,omitempty"`
	DueDate     *time.Time   `json:"dueDate,omitempty"`
	SortOrder   int          `json:"sortOrder"`

	// Поля, переданные в запросе (заполняется при UnmarshalJSON). Позволяет делать partial update.
	Provided map[string]bool `json:"-"`
}

func (dto *SubtaskDTO) UnmarshalJSON(data []byte) error {
	type Alias SubtaskDTO
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(dto)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	dto.Provided = make(map[string]bool, len(raw))
	for key := range raw {
		dto.Provided[key] = true
	}
	return nil
}

func (dto *SubtaskDTO) HasField(key string) bool {
	return dto.Provided != nil && dto.Provided[key]
}

func (dto *SubtaskDTO) MarkProvided(key string) {
	if dto.Provided == nil {
		dto.Provided = make(map[string]bool)
	}
	dto.Provided[key] = true
}

type DelSubtaskDTO struct {
	ID    uuid.UUID `json:"id"`
	Actor *Actor
}

func (dto *SubtaskDTO) GetChanges(old *Subtask) []*FieldChange {
	var changes []*FieldChange

	toStr := func(v interface{}) string {
		if v == nil {
			return "none"
		}
		return fmt.Sprintf("%v", v)
	}

	if dto.HasField("title") && dto.Title != old.Title {
		changes = append(changes, &FieldChange{ActionTitleChanged, old.Title, dto.Title})
	}
	if dto.HasField("description") && dto.Description != old.Description {
		changes = append(changes, &FieldChange{ActionDescriptionChanged, old.Description, dto.Description})
	}
	if dto.HasField("status") && dto.Status != old.Status {
		changes = append(changes, &FieldChange{ActionStatusChanged, toStr(old.Status), toStr(dto.Status)})
	}
	if dto.HasField("priority") && dto.Priority != old.Priority {
		changes = append(changes, &FieldChange{ActionPriorityChanged, toStr(old.Priority), toStr(dto.Priority)})
	}
	if dto.HasField("assigneeId") {
		if dto.AssigneeID != nil && (old.Assignee == nil || *dto.AssigneeID != old.Assignee.ID) {
			oldVal := "none"
			if old.Assignee != nil {
				oldVal = old.Assignee.ID.String()
			}
			changes = append(changes, &FieldChange{ActionAssigned, oldVal, dto.AssigneeID.String()})
		} else if dto.AssigneeID == nil && old.Assignee != nil {
			changes = append(changes, &FieldChange{ActionAssigned, old.Assignee.ID.String(), "none"})
		}
	}
	if dto.HasField("dueDate") {
		if dto.DueDate != nil && (old.DueDate == nil || *dto.DueDate != *old.DueDate) {
			changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), toStr(dto.DueDate)})
		} else if dto.DueDate == nil && old.DueDate != nil {
			changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), "none"})
		}
	}

	return changes
}
