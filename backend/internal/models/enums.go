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

func (s TicketStatus) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusPending, StatusOnHold, StatusResolved, StatusClosed, StatusCancelled:
		return true
	}
	return false
}

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// IsValid сообщает, что значение приоритета является допустимым (не мусор).
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

type ActivityType string

const (
	ActionCreated            ActivityType = "created"
	ActionClosed             ActivityType = "closed"
	ActionTitleChanged       ActivityType = "title_changed" // Изменен заголовок
	ActionDescriptionChanged ActivityType = "description_changed"
	ActionStatusChanged      ActivityType = "status_changed"
	ActionPriorityChanged    ActivityType = "priority_changed"
	ActionAssigned           ActivityType = "assigned"            // Назначен исполнитель
	ActionAssignChanged      ActivityType = "assign_changed"      // Изменен исполнитель
	ActionOwnerChanged       ActivityType = "owner_changed"       // Изменен владелец
	ActionOwnerMutedChanged  ActivityType = "owner_muted_changed" // Изменено скрытие/уведомление заказчика
	ActionGroupChanged       ActivityType = "group_changed"       // Изменена группа
	ActionGroupAssigned      ActivityType = "group_assigned"      // Назначена группа
	ActionManagerChanged     ActivityType = "manager_changed"     // Изменен начальник (manager)
	ActionDueDateChanged     ActivityType = "due_date_changed"    // Изменен срок
	ActionDueDateRequested   ActivityType = "due_date_requested"  // Запросили перенос срока
	ActionDueDateApproved    ActivityType = "due_date_approved"   // Срок перенесён (запрос утверждён)
	ActionDueDateRejected    ActivityType = "due_date_rejected"   // Срок не перенесён (запрос отклонён)
	ActionSiteChanged        ActivityType = "site_changed"        // Изменена площадка
	ActionCategoryChanged    ActivityType = "category_changed"    // Изменена категория
	ActionSubcategoryChanged ActivityType = "subcategory_changed" // Изменена подкатегория
	ActionEquipmentChanged   ActivityType = "equipment_changed"   // Изменено оборудование
	ActionNotifiersChanged   ActivityType = "notifiers_changed"   // Изменён список наблюдателей
	ActionCommentAdded       ActivityType = "comment_added"
	ActionDuplicateLinked    ActivityType = "dup_linked"    // Заявка помечена дублем другой
	ActionDuplicateUnlinked  ActivityType = "dup_unlinked"  // Дубль отвязан, статус восстановлен
	ActionDuplicateRepointed ActivityType = "dup_repointed" // Дубли перенаправлены на новую основную
)

func (a ActivityType) IsValid() bool {
	switch a {
	case ActionCreated, ActionClosed, ActionTitleChanged, ActionDescriptionChanged, ActionStatusChanged,
		ActionPriorityChanged, ActionAssigned, ActionAssignChanged, ActionOwnerChanged, ActionOwnerMutedChanged, ActionGroupChanged,
		ActionGroupAssigned, ActionManagerChanged, ActionDueDateChanged, ActionDueDateRequested, ActionDueDateApproved,
		ActionDueDateRejected, ActionSiteChanged, ActionCategoryChanged, ActionSubcategoryChanged, ActionEquipmentChanged, ActionNotifiersChanged,
		ActionCommentAdded, ActionDuplicateLinked, ActionDuplicateUnlinked, ActionDuplicateRepointed:
		return true
	}
	return false
}
