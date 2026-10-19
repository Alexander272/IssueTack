package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"userId" db:"user_id"`
	Type      string          `json:"type" db:"type"`
	Title     string          `json:"title" db:"title"`
	Body      string          `json:"body" db:"body"`
	Data      json.RawMessage `json:"data" db:"data"`
	IsRead    bool            `json:"isRead" db:"is_read"`
	CreatedAt time.Time       `json:"createdAt" db:"created_at"`
}

type CreateNotificationDTO struct {
	UserID uuid.UUID       `json:"userId"`
	Type   string          `json:"type"`
	Title  string          `json:"title"`
	Body   string          `json:"body"`
	Data   json.RawMessage `json:"data"`
}

// NotificationReminderDTO — параметры периодического напоминания по приоритету.
// Отдельная структура вместо длинного списка позиционных аргументов: у
// notifyPriorityReminder событие, тип уведомления, заголовок и интервал дедупа
// идут рядом и легко перепутать местами при вызове.
type NotificationReminderDTO struct {
	Ticket      *Ticket
	Event       string
	NotifType   string
	Title       string
	MinInterval time.Duration
}

// NotificationCommentDTO — параметры уведомления о новом комментарии для
// NotificationService.TicketCommented: текст комментария и подпись автора —
// два одинаковых string подряд, поэтому позиционно их передавать нельзя.
type NotificationCommentDTO struct {
	Ticket      *Ticket
	ActorID     uuid.UUID
	CommentText string
	AuthorLabel string
}

type NotificationSettings struct {
	UserID   uuid.UUID       `json:"userId" db:"user_id"`
	Settings json.RawMessage `json:"settings" db:"settings"`
}

// Категории событий, по которым могут настраиваться уведомления.
// Используются как ключи в JSON настройках (см. EventFieldName) и как события обращений к репозиторию.
const (
	EventNewTask        = "new"
	EventStatus         = "status"
	EventComment        = "comment"
	EventOverdue        = "overdue"
	EventReminderHigh   = "reminder_high"
	EventReminderUrgent = "reminder_urgent"
)

// EventFieldName возвращает имя поля JSON-настройки события для категории.
func EventFieldName(event string) string {
	switch event {
	case EventNewTask:
		return "newTask"
	case EventStatus:
		return "status"
	case EventComment:
		return "comment"
	case EventOverdue:
		return "overdue"
	case EventReminderHigh:
		return "reminderHigh"
	case EventReminderUrgent:
		return "reminderUrgent"
	default:
		return ""
	}
}

// CategoryNotificationSetting — настройки уведомлений по категории (матрица «категория × событие»).
type CategoryNotificationSetting struct {
	ID             uuid.UUID `json:"id"`
	NewTask        bool      `json:"newTask"`
	Status         bool      `json:"status"`
	Comment        bool      `json:"comment"`
	Overdue        bool      `json:"overdue"`
	ReminderHigh   bool      `json:"reminderHigh"`
	ReminderUrgent bool      `json:"reminderUrgent"`
}

// GroupNotificationSetting — настройки уведомлений по группе задач.
type GroupNotificationSetting struct {
	ID      uuid.UUID `json:"id"`
	NewTask bool      `json:"newTask"`
	Overdue bool      `json:"overdue"`
}

// NotificationSettingsPayload — персональные настройки уведомлений пользователя в типизированном виде.
// Хранится в колонке settings таблицы user_notification_settings как JSON.
//   - Enabled — мастер-переключатель: если выключен, добровольные подписки (по категориям и группам)
//     полностью отключены. Личные уведомления (назначение, менеджерство и т.п.) приходят всегда.
//   - Categories — матрица «категория × событие» (новые задачи, статус, комментарий, просрочка).
//   - Groups — настройки по группам (новые задачи, просрочка).
//   - DeadlineReminders — пороги напоминаний «скоро срок» исполнителю (в минутах до дедлайна).
//     Персональная настройка, не зависит от мастер-переключателя enabled (у исполнителей его нет).
type NotificationSettingsPayload struct {
	Enabled           bool                          `json:"enabled"`
	Categories        []CategoryNotificationSetting `json:"categories"`
	Groups            []GroupNotificationSetting    `json:"groups"`
	DeadlineReminders []int                         `json:"deadlineReminders"`
}

// DefaultNotificationSettings возвращает настройки уведомлений по умолчанию (все подписки включены,
// матрица пуста — пользователь ни на что не подписывается, пока не отметит категории/события).
func DefaultNotificationSettings() *NotificationSettingsPayload {
	return &NotificationSettingsPayload{
		Enabled:           true,
		Categories:        []CategoryNotificationSetting{},
		Groups:            []GroupNotificationSetting{},
		DeadlineReminders: DefaultDeadlineReminders(),
	}
}

// DefaultDeadlineReminders возвращает пороги напоминаний «скоро срок» по умолчанию:
// за 2 дня (2880 мин), 1 день (1440 мин) и 6 часов (360 мин) до дедлайна.
func DefaultDeadlineReminders() []int {
	return []int{2880, 1440, 360}
}

// DeadlineRemindersDTO — тело запроса/ответа персональных порогов напоминаний «скоро срок».
type DeadlineRemindersDTO struct {
	Reminders []int `json:"reminders"`
}
