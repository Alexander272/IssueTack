package models

import "time"

type TicketStatus string
type Priority string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusClosed     TicketStatus = "closed"
)

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type Ticket struct {
	ID          string `json:"id" db:"id"`
	Title       string `json:"title" db:"title"`
	Description string `json:"description" db:"description"`

	// Статусы и приоритеты
	Status   TicketStatus `json:"status" db:"status"`
	Priority Priority     `json:"priority" db:"priority"`

	// Кто участвует
	CreatorID  string  `json:"creatorId" db:"creator_id"`   // Клиент
	GroupID    *string `json:"groupId" db:"group_id"`       // На какую группу назначен (напр. IT-отдел)
	AssigneeID *string `json:"assigneeId" db:"assignee_id"` // Конкретный агент-исполнитель
	ManagerID  *string `json:"managerId" db:"manager_id"`   // Начальник, контролирующий задачу

	// Сроки (SLA)
	DueDate *time.Time `json:"dueDate" db:"due_date"` // Тот самый срок выполнения

	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	ClosedAt  *time.Time `json:"closedAt" db:"closed_at"`
}
