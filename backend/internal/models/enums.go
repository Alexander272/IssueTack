package models

type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusPending    TicketStatus = "pending"
	StatusOnHold     TicketStatus = "on_hold"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
	StatusCancelled  TicketStatus = "cancelled"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// type Role string

// const (
// 	RoleAdmin   Role = "admin"
// 	RoleManager Role = "manager"
// 	RoleAgent   Role = "agent"
// 	RoleClient  Role = "client"
// )

type ActivityType string

const (
	ActionCreated            ActivityType = "created"
	ActionClosed             ActivityType = "closed"
	ActionTitleChanged       ActivityType = "title_changed" // Изменен заголовок
	ActionDescriptionChanged ActivityType = "description_changed"
	ActionStatusChanged      ActivityType = "status_changed"
	ActionPriorityChanged    ActivityType = "priority_changed"
	ActionAssigned           ActivityType = "assigned"         // Назначен исполнитель
	ActionAssignChanged      ActivityType = "assign_changed"   // Изменен исполнитель
	ActionOwnerChanged       ActivityType = "owner_changed"    // Изменен владелец
	ActionGroupChanged       ActivityType = "group_changed"    // Изменена группа
	ActionGroupAssigned      ActivityType = "group_assigned"   // Назначена группа
	ActionDueDateChanged     ActivityType = "due_date_changed" // Изменен срок
	ActionSiteChanged        ActivityType = "site_changed"     // Изменена площадка
	ActionCategoryChanged    ActivityType = "category_changed" // Изменена категория
	ActionCommentAdded       ActivityType = "comment_added"
)
