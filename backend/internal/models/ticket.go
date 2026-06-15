package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`

	// Статусы и приоритеты
	Status   TicketStatus `json:"status" db:"status"`
	Priority Priority     `json:"priority" db:"priority"`

	Site     *SiteShort     `json:"site"`     // Площадка выполнения
	Category *CategoryShort `json:"category"` // Категория

	// Кто участвует
	Creator  UserShort   `json:"creator"`  // Кто фактически создал (может быть Manager)
	Owner    *UserShort  `json:"owner"`    // Для кого выполняется (Клиент/Заявитель)
	Group    *GroupShort `json:"group"`    // На какую группу назначен (напр. IT-отдел)
	Assignee *UserShort  `json:"assignee"` // Конкретный агент-исполнитель
	Manager  *UserShort  `json:"manager"`  // Начальник, контролирующий задачу

	// Сроки и планирование
	DueDate   *time.Time `json:"dueDate" db:"due_date"`
	ClosedAt  *time.Time `json:"closedAt" db:"closed_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`

	Subtasks    []*Subtask    `json:"subtasks,omitempty"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

type GetTicketByIdDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Actor `json:"actor"`
}

type TicketFilter struct {
	SiteID     *uuid.UUID    `json:"siteId" db:"site_id"`
	Status     *TicketStatus `json:"status" db:"status"`
	OwnerID    *uuid.UUID    `json:"ownerId" db:"owner_id"`
	AssigneeID *uuid.UUID    `json:"assigneeId" db:"assignee_id"`
	GroupID    *uuid.UUID    `json:"groupId" db:"group_id"`
	Limit      int           `json:"limit" db:"limit"`
	Offset     int           `json:"offset" db:"offset"`
}

type TicketDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Actor `json:"actor"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`

	// Статусы и приоритеты
	Status   TicketStatus `json:"status" db:"status"`
	Priority Priority     `json:"priority" db:"priority"`

	SiteID     uuid.UUID `json:"siteId"`     // Площадка выполнения
	CategoryID uuid.UUID `json:"categoryId"` // Категория

	// Кто участвует
	CreatorID  uuid.UUID  `json:"creatorId" db:"creator_id"`   // Кто фактически создал (может быть Manager)
	OwnerID    *uuid.UUID `json:"ownerId" db:"owner_id"`       // Для кого выполняется (Клиент/Заявитель)
	GroupID    *uuid.UUID `json:"groupId" db:"group_id"`       // На какую группу назначен (напр. IT-отдел)
	AssigneeID *uuid.UUID `json:"assigneeId" db:"assignee_id"` // Конкретный агент-исполнитель
	ManagerID  *uuid.UUID `json:"managerId" db:"manager_id"`   // Начальник, контролирующий задачу

	// Сроки и планирование
	DueDate  *time.Time `json:"dueDate" db:"due_date"`
	ClosedAt *time.Time `json:"closedAt" db:"closed_at"`
}

type DeleteTicketDTO struct {
	ID    uuid.UUID `json:"id" db:"id"`
	Actor `json:"actor"`
}

type FieldChange struct {
	Tag    ActivityType
	OldVal string
	NewVal string
}

func (dto *TicketDTO) GetChanges(old *Ticket) []*FieldChange {
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

	if dto.DueDate != nil && (old.DueDate == nil || *dto.DueDate != *old.DueDate) {
		changes = append(changes, &FieldChange{ActionDueDateChanged, toStr(old.DueDate), toStr(dto.DueDate)})
	}
	if dto.ClosedAt != nil && (old.ClosedAt == nil || *dto.ClosedAt != *old.ClosedAt) {
		changes = append(changes, &FieldChange{ActionClosed, toStr(old.ClosedAt), toStr(dto.ClosedAt)})
	}

	if dto.SiteID != old.Site.ID {
		changes = append(changes, &FieldChange{ActionSiteChanged, old.Site.ID.String(), dto.SiteID.String()})
	}
	if dto.CategoryID != old.Category.ID {
		changes = append(changes, &FieldChange{ActionCategoryChanged, old.Category.ID.String(), dto.CategoryID.String()})
	}

	if dto.GroupID != nil && (old.Group == nil || *dto.GroupID != old.Group.ID) {
		oldVal := "none"
		if old.Group != nil {
			oldVal = old.Group.ID.String()
			changes = append(changes, &FieldChange{ActionGroupChanged, oldVal, dto.GroupID.String()})
		} else {
			changes = append(changes, &FieldChange{ActionGroupAssigned, oldVal, dto.GroupID.String()})
		}
	}

	if dto.AssigneeID != nil && (old.Assignee == nil || *dto.AssigneeID != old.Assignee.ID) {
		oldVal := "none"
		if old.Assignee != nil {
			oldVal = old.Assignee.ID.String()
			changes = append(changes, &FieldChange{ActionAssignChanged, oldVal, dto.AssigneeID.String()})
		} else {
			changes = append(changes, &FieldChange{ActionAssigned, oldVal, dto.AssigneeID.String()})
		}
	}
	if dto.OwnerID != nil && (old.Owner == nil || *dto.OwnerID != old.Owner.ID) {
		oldVal := "none"
		if old.Owner != nil {
			oldVal = old.Owner.ID.String()
		}
		changes = append(changes, &FieldChange{ActionOwnerChanged, oldVal, dto.OwnerID.String()})
	}

	return changes
}
