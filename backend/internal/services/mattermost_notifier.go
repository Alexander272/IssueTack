package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

// mattermostNotifier — канал доставки уведомлений в личные сообщения Mattermost
// (DM от бота реалма). Условия доставки: у тикета есть realm, для реалма настроена
// активная Mattermost-интеграция, у пользователя заполнен mattermost_id. Формат
// сообщения настраивается отдельно для каждого типа события в format.
type mattermostNotifier struct {
	mmRepo  repository.Mattermost
	users   Users
	sender  mattermostSender
	baseURL string
}

// NewMattermostNotifier создаёт канал Mattermost.
func NewMattermostNotifier(mmRepo repository.Mattermost, users Users, sender mattermostSender, baseURL string) *mattermostNotifier {
	return &mattermostNotifier{
		mmRepo:  mmRepo,
		users:   users,
		sender:  sender,
		baseURL: baseURL,
	}
}

func (n *mattermostNotifier) Name() string { return "mattermost" }

// Notify отправляет DM пользователю, если интеграция активна и у пользователя есть
// mattermost_id. Пропуски (нет реалма/интеграции/адреса) — штатные условия, ошибками
// не считаются. Сбой самой отправки возвращается как ошибка.
func (n *mattermostNotifier) Notify(ctx context.Context, userID uuid.UUID, notif *models.CreateNotificationDTO, ticket *models.Ticket) error {
	if n.mmRepo == nil || n.users == nil || n.sender == nil {
		return nil
	}
	if ticket == nil || ticket.RealmID == nil {
		return nil
	}

	settings, err := n.mmRepo.GetByRealm(ctx, *ticket.RealmID)
	if err != nil {
		logger.Warn("failed to load mattermost settings for notification",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.ErrAttr(err),
		)
		return nil
	}
	if !settings.IsActive || settings.BotToken == "" {
		return nil
	}

	user, err := n.users.GetByID(ctx, userID)
	if err != nil {
		logger.Warn("failed to load user for mattermost notification",
			logger.StringAttr("user_id", userID.String()),
			logger.ErrAttr(err),
		)
		return nil
	}
	if user.MattermostID == nil || *user.MattermostID == "" {
		return nil
	}

	if err := n.sender.Send(settings.BotToken, settings.BotUserID, *user.MattermostID, n.format(ticket, notif)); err != nil {
		return fmt.Errorf("failed to send mattermost notification: %w", err)
	}
	return nil
}

// format собирает текст DM для конкретного события. Чтобы поправить сообщение
// отдельного события — правится только соответствующая ветка префикса/дополнений.
func (n *mattermostNotifier) format(ticket *models.Ticket, notif *models.CreateNotificationDTO) string {
	prefix := map[string]string{
		string(models.NotificationTicketCreated):    "Новая задача",
		string(models.NotificationTicketUpdated):    "Задача обновлена",
		string(models.NotificationTicketDeleted):    "Задача удалена",
		string(models.NotificationTicketComment):    "Новый комментарий",
		string(models.NotificationTicketAttachment): "Новое вложение",
		string(models.NotificationTicketOverdue):    "Задача просрочена",
	}[notif.Type]
	if prefix == "" {
		prefix = "Уведомление"
	}

	title := notif.Body
	if title == "" {
		title = ticket.Title
	}

	var number string
	if ticket.TicketNumber != nil {
		number = fmt.Sprintf(" №%d", *ticket.TicketNumber)
	}

	text := fmt.Sprintf("**%s%s: %s**", prefix, number, title)

	if notif.Type == string(models.NotificationTicketUpdated) {
		if details := n.changesSummary(notif.Data); details != "" {
			text += "\n\n" + details
		}
	}

	if n.baseURL != "" {
		text += fmt.Sprintf("\nОткрыть: %s/tasks/%s", n.baseURL, ticket.ID.String())
	}
	return text
}

// changesSummary превращает сводку изменений тикета (поле data.changes) в многострочный
// текст вида «• поле: старое → новое». Возвращает пустую строку, если изменений нет.
func (n *mattermostNotifier) changesSummary(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var payload struct {
		Changes string `json:"changes"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || payload.Changes == "" {
		return ""
	}
	var changes []*models.FieldChange
	if err := json.Unmarshal([]byte(payload.Changes), &changes); err != nil {
		return ""
	}
	lines := make([]string, 0, len(changes))
	for _, ch := range changes {
		lines = append(lines, fmt.Sprintf("• %s: %s → %s", ch.Tag, prettyVal(ch.OldVal), prettyVal(ch.NewVal)))
	}
	return strings.Join(lines, "\n")
}

func prettyVal(v string) string {
	if v == "" || v == "none" {
		return "—"
	}
	return v
}
