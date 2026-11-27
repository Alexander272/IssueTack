package models

// NotificationType — типы уведомлений, генерируемые сервисом уведомлений.
type NotificationType string

const (
	NotificationTicketCreated          NotificationType = "ticket.created"
	NotificationTicketUpdated          NotificationType = "ticket.updated"
	NotificationTicketDeleted          NotificationType = "ticket.deleted"
	NotificationTicketComment          NotificationType = "ticket.comment_added"
	NotificationTicketAttachment       NotificationType = "ticket.attachment_added"
	NotificationTicketOverdue          NotificationType = "ticket.overdue"
	NotificationDeadlineSoon           NotificationType = "ticket.deadline_soon"
	NotificationTicketReminderHigh     NotificationType = "ticket.reminder_high"
	NotificationTicketReminderUrgent   NotificationType = "ticket.reminder_urgent"
	NotificationTicketDueDateRequested NotificationType = "ticket.due_date_requested"
	NotificationTicketDueDateApproved  NotificationType = "ticket.due_date_approved"
	NotificationTicketDueDateRejected  NotificationType = "ticket.due_date_rejected"
	NotificationDuplicateLinked        NotificationType = "ticket.duplicate_linked"
	NotificationDuplicateUnlinked      NotificationType = "ticket.duplicate_unlinked"
	// NotificationEquipmentReturnDue — напоминание о необходимости вернуть
	// выданное оборудование: приходит тому, кто выдал, когда наступил срок
	// возврата, а единица всё ещё у держателя.
	NotificationEquipmentReturnDue NotificationType = "equipment.return_due"
)
