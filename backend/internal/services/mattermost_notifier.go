package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
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
	mmRepo   repository.Mattermost
	users    Users
	sender   mattermostSender
	policies AccessPolicies
	baseURL  string
}

// NewMattermostNotifier создаёт канал Mattermost. policies нужен для проверки
// coarse-права ticket:read: веб-ссылка на заявку ведёт на маршрут, закрытый
// Casbin-мидлваром, и без проверки получателю уходит ссылка, которая откроет 403.
func NewMattermostNotifier(mmRepo repository.Mattermost, users Users, sender mattermostSender, policies AccessPolicies, baseURL string) *mattermostNotifier {
	return &mattermostNotifier{
		mmRepo:   mmRepo,
		users:    users,
		sender:   sender,
		policies: policies,
		baseURL:  baseURL,
	}
}

func (n *mattermostNotifier) Name() string { return "mattermost" }

// Notify отправляет DM пользователю, если интеграция активна и у пользователя есть
// mattermost_id. Пропуски (нет реалма/интеграции/адреса) — штатные условия, ошибками
// не считаются, но сообщаются как delivered=false (получатель не сохраняется в БД и
// будет обслужен при следующей попытке). Сбой самой отправки возвращается как ошибка.
func (n *mattermostNotifier) Notify(ctx context.Context, userID uuid.UUID, notif *models.CreateNotificationDTO, ticket *models.Ticket) (bool, error) {
	if n.mmRepo == nil || n.users == nil || n.sender == nil {
		return false, nil
	}
	if ticket == nil || ticket.RealmID == nil {
		return false, nil
	}

	settings, err := n.mmRepo.GetByRealm(ctx, *ticket.RealmID)
	if err != nil {
		logger.Warn("failed to load mattermost settings for notification",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.ErrAttr(err),
		)
		return false, nil
	}
	if !settings.IsActive || settings.BotToken == "" {
		logger.Info("mattermost notification skipped: integration is not active",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.StringAttr("user_id", userID.String()),
		)
		return false, nil
	}

	user, err := n.users.GetByID(ctx, userID)
	if err != nil {
		logger.Warn("failed to load user for mattermost notification",
			logger.StringAttr("user_id", userID.String()),
			logger.ErrAttr(err),
		)
		return false, nil
	}
	if user.MattermostID == nil || *user.MattermostID == "" {
		logger.Info("mattermost notification skipped: user has no mattermost_id",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.StringAttr("user_id", userID.String()),
			logger.StringAttr("username", user.Username),
		)
		return false, nil
	}

	if err := n.sender.Send(settings.BotToken, settings.BotUserID, *user.MattermostID, n.format(ctx, user, ticket, notif)); err != nil {
		return false, fmt.Errorf("failed to send mattermost notification: %w", err)
	}
	logger.Info("mattermost notification sent",
		logger.StringAttr("ticket_id", ticket.ID.String()),
		logger.StringAttr("user_id", userID.String()),
		logger.StringAttr("username", user.Username),
	)
	return true, nil
}

// format собирает текст DM для конкретного события. Чтобы поправить сообщение
// отдельного события — правится только соответствующая ветка префикса/дополнений.
// format собирает текст DM для конкретного события. События «Задача …» («просрочена»,
// «обновлена», «удалена») и «Скоро срок» выводят номер сразу после слова «Задача»/
// «Скоро срок» («Задача №12 обновлена: …»), прочие типы — номер после префикса
// («Новая задача №12: …»). Для overdue и deadline_soon при заданном дедлайне
// добавляется строка «Дедлайн: …». Блок ссылок общий для всех типов и добавляется
// один раз (links), чтобы правило показа веб-ссылки жило в одном месте.
func (n *mattermostNotifier) format(ctx context.Context, user *models.UserData, ticket *models.Ticket, notif *models.CreateNotificationDTO) string {
	var prefix, action string
	switch notif.Type {
	case string(models.NotificationTicketUpdated):
		prefix, action = "Задача", "обновлена"
	case string(models.NotificationTicketDeleted):
		prefix, action = "Задача", "удалена"
	case string(models.NotificationTicketOverdue):
		prefix, action = "Задача", "просрочена"
	case string(models.NotificationDeadlineSoon):
		prefix = "Скоро срок"
	case string(models.NotificationTicketCreated):
		prefix = "Новая задача"
	case string(models.NotificationTicketComment):
		prefix = "Новый комментарий"
	case string(models.NotificationTicketAttachment):
		prefix = "Новое вложение"
	default:
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

	links := n.links(ctx, user, ticket)

	if action != "" {
		// «Задача №12 обновлена: title» — действие идёт после номера.
		text := fmt.Sprintf("**%s%s %s: %s**", prefix, number, action, title)
		if notif.Type == string(models.NotificationTicketOverdue) && ticket.DueDate != nil {
			text += fmt.Sprintf("\nДедлайн: %s", ticket.DueDate.Format("02.01.2006 15:04"))
		}
		if notif.Type == string(models.NotificationTicketUpdated) {
			if details := n.changesSummary(user.ID, notif.Data); details != "" {
				text += "\n\n" + details
			}
		}
		return text + links
	}

	text := fmt.Sprintf("**%s%s: %s**", prefix, number, title)

	if notif.Type == string(models.NotificationTicketCreated) && ticket.Owner != nil {
		if label := userShortLabel(ticket.Owner); label != "" {
			text += "\nЗаказчик: " + label
		}
	}

	if notif.Type == string(models.NotificationDeadlineSoon) && ticket.DueDate != nil {
		text += fmt.Sprintf("\nДедлайн: %s", ticket.DueDate.Format("02.01.2006 15:04"))
	}

	if notif.Type == string(models.NotificationTicketUpdated) {
		if details := n.changesSummary(user.ID, notif.Data); details != "" {
			text += "\n\n" + details
		}
	}

	if notif.Type == string(models.NotificationTicketComment) {
		if block := commentBlock(notif.Data); block != "" {
			text += "\n\n" + block
		}
	}

	return text + links
}

// userShortLabel собирает «Фамилия Имя» заказчика для DM; без ФИО — username.
func userShortLabel(u *models.UserShort) string {
	name := strings.TrimSpace(u.LastName + " " + u.FirstName)
	if name == "" {
		return u.Username
	}
	return name
}

// commentBlock извлекает из data.comment текст и автора комментария и собирает
// блок «Комментарий пользователя <ФИО>:\n<текст>». Без автора — только текст,
// пустой data/текст — пустая строка (блок не печатается).
func commentBlock(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var payload struct {
		Comment struct {
			Text   string `json:"text"`
			Author string `json:"author"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || payload.Comment.Text == "" {
		return ""
	}
	if payload.Comment.Author == "" {
		return payload.Comment.Text
	}
	return fmt.Sprintf("Комментарий пользователя %s:\n%s", payload.Comment.Author, payload.Comment.Text)
}

// links собирает блок ссылок в конце DM. Ссылка на заявку внутри Mattermost
// (pluginLink) добавляется всем: её маршрут защищён SourceGuard/PluginTokenGuard,
// Casbin-мидлвара на нём нет, а доступ к заявке решается на уровне сервиса по
// атрибутам. Веб-ссылка ведёт на /tasks/:id, который закрыт coarse-правом
// ticket:read, поэтому она персональная: показывается только веб-пользователю
// (Source != mattermost — у пользователей из Mattermost нет учётки Keycloak) и
// только при наличии права в домене тикета. Нет права, не веб-пользователь или
// ошибка проверки — веб-ссылка не печатается (fail-closed на ссылке, доставка
// DM при этом не страдает).
func (n *mattermostNotifier) links(ctx context.Context, user *models.UserData, ticket *models.Ticket) string {
	links := n.pluginLink(ticket.ID)
	if user == nil || user.Source.IsFromMattermost() {
		return links
	}
	if n.baseURL == "" || n.policies == nil {
		return links
	}
	if ticket.RealmID == nil {
		return links
	}

	allowed, err := n.policies.Enforce(user.ID.String(), ticket.RealmID.String(), string(access.ResourceTicket), string(access.Read))
	if err != nil {
		logger.Warn("failed to check ticket read policy for mattermost notification",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.StringAttr("user_id", user.ID.String()),
			logger.ErrAttr(err),
		)
		return links
	}
	if !allowed {
		logger.Info("mattermost notification skipped web link: no ticket read permission",
			logger.StringAttr("ticket_id", ticket.ID.String()),
			logger.StringAttr("user_id", user.ID.String()),
		)
		return links
	}
	return fmt.Sprintf("\nОткрыть: %s/tasks/%s", n.baseURL, ticket.ID.String()) + links
}

// pluginLink возвращает вторую ссылку в DM — на заявку внутри Mattermost.
// Ссылка site-relative: Mattermost помечает такой href как data-link и открывает
// его через собственный роутер, поэтому заявка открывается внутри плагина, без
// перехода в веб-приложение. Абсолютный URL здесь не подходит — site URL сервера
// Mattermost бэкенду неизвестен, а бот-токен не может прочитать /api/v4/config.
//
// Ссылка обязана быть markdown-ссылкой: autolink в Mattermost срабатывает только
// на URL со схемой (http://, mailto:), поэтому site-relative путь в виде обычного
// текста остаётся некликабельным.
func (n *mattermostNotifier) pluginLink(ticketID uuid.UUID) string {
	return fmt.Sprintf("\n[Открыть в плагине](%s)", pluginDeepLink(ticketID))
}

// changeTagLabels — русские подписи полей в сводке изменений тикета. Значения
// статусов/приоритетов/сроков переводятся отдельно, поля с id выводятся вообще
// без значений (см. changeLine), чтобы в DM не светились сырые uuid.
var changeTagLabels = map[models.ActivityType]string{
	models.ActionCreated:            "Создана",
	models.ActionClosed:             "Закрыта",
	models.ActionTitleChanged:       "Заголовок",
	models.ActionDescriptionChanged: "Описание",
	models.ActionStatusChanged:      "Статус",
	models.ActionPriorityChanged:    "Приоритет",
	models.ActionOwnerChanged:       "Заказчик",
	models.ActionGroupChanged:       "Группа",
	models.ActionGroupAssigned:      "Группа",
	models.ActionManagerChanged:     "Начальник",
	models.ActionDueDateChanged:     "Срок",
	models.ActionSiteChanged:        "Площадка",
	models.ActionCategoryChanged:    "Категория",
}

// changesSummary превращает сводку изменений тикета (поле data.changes) в многострочный
// текст вида «• поле: старое → новое». Возвращает пустую строку, если изменений нет.
func (n *mattermostNotifier) changesSummary(userID uuid.UUID, data json.RawMessage) string {
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
		if line, ok := n.changeLine(userID, ch); ok {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// changeLine превращает одно изменение тикета в строку сводки. При передаче заявки
// новому исполнителю пишем «Вам назначена задача.» без id, остальным получателям —
// нейтрально. Поля с id (заказчик/начальник/группа/категория/площадка) выводятся без
// значений. Неизвестный тег строку не даёт, чтобы в DM не просочился английский тег.
func (n *mattermostNotifier) changeLine(userID uuid.UUID, ch *models.FieldChange) (string, bool) {
	switch ch.Tag {
	case models.ActionAssigned, models.ActionAssignChanged:
		if ch.NewVal == "" || ch.NewVal == "none" {
			return "", false
		}
		newAssigneeID, err := uuid.Parse(ch.NewVal)
		if err != nil {
			return "", false
		}
		if newAssigneeID == userID {
			return "Вам назначена задача.", true
		}
		verb := "назначен"
		if ch.Tag == models.ActionAssignChanged {
			verb = "изменён"
		}
		return "Исполнитель: " + verb, true

	case models.ActionOwnerChanged:
		return "Заказчик: изменён", true
	case models.ActionManagerChanged:
		return "Начальник: изменён", true
	case models.ActionGroupChanged:
		return "Группа: изменена", true
	case models.ActionGroupAssigned:
		return "Группа: назначена", true
	case models.ActionSiteChanged:
		return "Площадка: изменена", true
	case models.ActionCategoryChanged:
		return "Категория: изменена", true
	}

	label, ok := changeTagLabels[ch.Tag]
	if !ok {
		return "", false
	}

	switch ch.Tag {
	case models.ActionStatusChanged:
		return fmt.Sprintf("• %s: %s → %s", label, ruStatus(ch.OldVal), ruStatus(ch.NewVal)), true
	case models.ActionPriorityChanged:
		return fmt.Sprintf("• %s: %s → %s", label, ruPriority(ch.OldVal), ruPriority(ch.NewVal)), true
	case models.ActionDueDateChanged, models.ActionClosed:
		return fmt.Sprintf("• %s: %s → %s", label, ruTime(ch.OldVal), ruTime(ch.NewVal)), true
	default:
		return fmt.Sprintf("• %s: %s → %s", label, prettyVal(ch.OldVal), prettyVal(ch.NewVal)), true
	}
}

// ruStatus переводит значение статуса в русскую подпись (той же, что в карточках бота).
func ruStatus(v string) string {
	if v == "" || v == "none" {
		return "—"
	}
	if label, ok := mmStatusLabels[models.TicketStatus(v)]; ok {
		return label
	}
	return v
}

// ruPriority переводит значение приоритета в русскую подпись (той же, что в карточках бота).
func ruPriority(v string) string {
	if v == "" || v == "none" {
		return "—"
	}
	if label, ok := mmPriorityLabels[models.Priority(v)]; ok {
		return label
	}
	return v
}

// ruTime форматирует «сырое» время изменения (строковое представление *time.Time)
// в читаемый «02.01.2006 15:04». Неразобранное значение возвращается как есть.
func ruTime(v string) string {
	if v == "" || v == "none" {
		return "—"
	}
	if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", v); err == nil {
		return t.Format("02.01.2006 15:04")
	}
	return v
}

func prettyVal(v string) string {
	if v == "" || v == "none" {
		return "—"
	}
	return v
}
