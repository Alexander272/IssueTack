package models

import (
	"time"

	"github.com/google/uuid"
)

// TicketSubscription — подписка пользователя на получение уведомлений об изменениях по конкретной заявке.
type TicketSubscription struct {
	TicketID  uuid.UUID `json:"ticketId" db:"ticket_id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// SubscribeDTO — запрос на изменение подписки на заявку.
type SubscribeDTO struct {
	ActorID  uuid.UUID `json:"-"`
	TicketID uuid.UUID `json:"-"`
}

type IsSubscribedDTO struct {
	ActorID  uuid.UUID `json:"-"`
	TicketID uuid.UUID `json:"-"`
}

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
