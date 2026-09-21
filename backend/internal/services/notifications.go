package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

// NotificationService — сервис уведомлений пользователей: сохранение в БД (persist) и доставка
// по каналам (channels, сейчас Mattermost). Сервис не обращается к репозиторию тикетов напрямую:
// агрегат тикета приходит уже загруженным от сервиса-владельца (TicketService) через параметры
// методов — так удаётся избежать цикла зависимостей (tickets → notifications → tickets) и
// сохранить инкапсуляцию бизнес-правил.
type NotificationService struct {
	repo          repository.Notifications
	subscriptions TicketSubscriptionOps
	userRealms    UserRealms
	groups        Groups
	txManager     TransactionManager
	channels      []Notifier
}

// NewNotificationService создаёт NotificationService. Уведомления сохраняются в БД
// (persist), а затем рассылаются по включённым каналам (channels) — сейчас
// Mattermost, в будущем push/email.
func NewNotificationService(deps *NotificationDeps) *NotificationService {
	return &NotificationService{
		repo:          deps.Repo,
		subscriptions: deps.Subscriptions,
		userRealms:    deps.UserRealms,
		groups:        deps.Groups,
		txManager:     deps.TxManager,
		channels:      deps.Channels,
	}
}

// NotificationDeps — зависимости, необходимые для создания NotificationService.
type NotificationDeps struct {
	Repo          repository.Notifications
	Subscriptions TicketSubscriptionOps
	UserRealms    UserRealms
	Groups        Groups
	TxManager     TransactionManager
	Channels      []Notifier
}

// Notifications — интерфейс уведомлений о событиях тикетов.
type Notifications interface {
	// TicketCreated оповещает заинтересованных пользователей о создании тикета.
	// Сам создатель (actorID) уведомление не получает.
	TicketCreated(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error
	// TicketUpdated оповещает заинтересованных пользователей об обновлении тикета.
	TicketUpdated(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID, changes []*models.FieldChange) error
	// TicketDeleted оповещает заинтересованных пользователей об удалении тикета.
	TicketDeleted(ctx context.Context, ticket *models.Ticket) error
	// TicketCommented оповещает исполнителя и подписанных о новом комментарии.
	TicketCommented(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error
	// AttachmentAdded оповещает исполнителя и подписанных о новом вложении.
	AttachmentAdded(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error
	// NotifyOverdue оповещает о просроченном тикете: исполнителя, менеджера, админов реалма,
	// а также подписчиков категории/группы на событие «Просрочка».
	NotifyOverdue(ctx context.Context, ticket *models.Ticket) error
	// GetOverdueTicketIDs возвращает ID активных тикетов с просроченным сроком.
	GetOverdueTicketIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error)
	// GetSettings возвращает персональные настройки уведомлений пользователя.
	GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error)
	// SaveSettings сохраняет персональные настройки уведомлений пользователя.
	SaveSettings(ctx context.Context, userID uuid.UUID, settings json.RawMessage) error
	// GetSettingsPayload возвращает типизированные настройки уведомлений пользователя.
	GetSettingsPayload(ctx context.Context, userID uuid.UUID) (*models.NotificationSettingsPayload, error)
	// SaveSettingsPayload сохраняет типизированные настройки уведомлений пользователя.
	SaveSettingsPayload(ctx context.Context, userID uuid.UUID, settings *models.NotificationSettingsPayload) error
	// SendUnread отправляет клиенту непрочитанные уведомления.
	SendUnread(ctx context.Context, client *ws_hub.Client) error
}

// NotifyOverdue оповещает о просроченном тикете: исполнителя, менеджера, админов реалма и
// подписчиков категории/группы на событие «Просрочка». Дедупликация — по уже существующему
// уведомлению ticket.overdue для тикета (чтобы cron не спамил на каждом прогоне).
func (s *NotificationService) NotifyOverdue(ctx context.Context, ticket *models.Ticket) error {

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil {
		recipients[ticket.Assignee.ID] = struct{}{}
	}
	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}
	// Подписанные на заявку, у которых включено событие «Просрочка» для категории тикета.
	overdueSubs, err := s.subscribersByEvent(ctx, ticket, models.EventOverdue)
	if err != nil {
		return err
	}
	for _, id := range overdueSubs {
		recipients[id] = struct{}{}
	}
	if ticket.Category != nil {
		catSubs, err := s.repo.GetCategoryEventSubscribers(ctx, ticket.Category.ID, models.EventFieldName(models.EventOverdue))
		if err != nil {
			return fmt.Errorf("failed to get overdue category subscribers: %w", err)
		}
		for _, id := range catSubs {
			recipients[id] = struct{}{}
		}
	}
	if ticket.Group != nil {
		grpSubs, err := s.repo.GetGroupEventSubscribers(ctx, ticket.Group.ID, models.EventFieldName(models.EventOverdue))
		if err != nil {
			return fmt.Errorf("failed to get overdue group subscribers: %w", err)
		}
		for _, id := range grpSubs {
			recipients[id] = struct{}{}
		}
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal overdue notification data: %w", err)
	}

	var pending []uuid.UUID
	for userID := range recipients {
		exists, err := s.repo.HasNotification(ctx, userID, ticket.ID, string(models.NotificationTicketOverdue))
		if err != nil {
			return fmt.Errorf("failed to check overdue notification: %w", err)
		}
		if !exists {
			pending = append(pending, userID)
		}
	}

	if len(pending) == 0 {
		return nil
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketOverdue),
		Title: "Задача просрочена",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, pending, dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// TicketCreated оповещает менеджера, ответственных категории и исполнителя о создании тикета,
// а также авто-подписывает на заявку надзителей реалма и менеджера группы (с включёнными
// уведомлениями), чтобы они получали дальнейшие события через подписку. Создатель тикета
// уведомление о собственном действии не получает.
func (s *NotificationService) TicketCreated(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	if ticket.Assignee != nil {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	if ticket.Category != nil {
		responsible, err := s.repo.GetResponsibleByCategory(ctx, ticket.Category.ID)
		if err != nil {
			return fmt.Errorf("failed to get responsible by category: %w", err)
		}
		for _, id := range responsible {
			recipients[id] = struct{}{}
		}
	}

	// Авто-подписка на заявку: надзители реалма и менеджер группы с включёнными уведомлениями.
	auto, err := s.autoSubscribeOnCreate(ctx, ticket)
	if err != nil {
		return err
	}
	for _, id := range auto {
		recipients[id] = struct{}{}
	}

	// Пользователи, подписавшиеся на новые задачи в категории тикета (добровольная подписка).
	if ticket.Category != nil {
		subscribers, err := s.repo.GetCategoryEventSubscribers(ctx, ticket.Category.ID, models.EventFieldName(models.EventNewTask))
		if err != nil {
			return fmt.Errorf("failed to get new-task subscribers by category: %w", err)
		}
		for _, id := range subscribers {
			recipients[id] = struct{}{}
		}
	}

	// Участники группы тикета, подписавшиеся на новые задачи этой группы.
	if ticket.Group != nil {
		groupSubscribers, err := s.repo.GetGroupEventSubscribers(ctx, ticket.Group.ID, models.EventFieldName(models.EventNewTask))
		if err != nil {
			return fmt.Errorf("failed to get new-task group subscribers: %w", err)
		}
		for _, id := range groupSubscribers {
			recipients[id] = struct{}{}
		}
	}

	// Создатель тикета не уведомляется о собственном действии.
	delete(recipients, actorID)

	if len(recipients) == 0 {
		return nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketCreated),
		Title: "Новая задача",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, recipientsToSlice(recipients), dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// TicketUpdated оповещает менеджера и новых исполнителей об обновлении тикета.
func (s *NotificationService) TicketUpdated(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID, changes []*models.FieldChange) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	// Подписанные на заявку, у которых включено событие «Изменение статуса» для категории.
	statusSubs, err := s.subscribersByEvent(ctx, ticket, models.EventStatus)
	if err != nil {
		return err
	}
	for _, id := range statusSubs {
		recipients[id] = struct{}{}
	}

	for _, change := range changes {
		switch change.Tag {
		case models.ActionAssigned:
			newAssigneeID, err := uuid.Parse(change.NewVal)
			if err != nil {
				continue
			}
			if newAssigneeID == actorID {
				responsible, err := s.repo.GetResponsibleByCategory(ctx, ticket.Category.ID)
				if err != nil {
					return fmt.Errorf("failed to get responsible by category: %w", err)
				}
				for _, id := range responsible {
					recipients[id] = struct{}{}
				}
			} else {
				recipients[newAssigneeID] = struct{}{}
			}

		case models.ActionAssignChanged:
			if change.NewVal == "" || change.NewVal == "none" {
				continue
			}
			newAssigneeID, err := uuid.Parse(change.NewVal)
			if err != nil {
				continue
			}
			recipients[newAssigneeID] = struct{}{}
		}
	}

	// При смене статуса (в т.ч. отмена/возврат в работу) исполнитель уведомляется всегда,
	// кроме случая, когда исполнитель — сам автор изменения.
	if hasStatusChange(changes) && ticket.Assignee != nil && ticket.Assignee.ID != actorID {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	// Если изменён статус задачи — уведомляем подписчиков категории на событие «Изменение статуса».
	if ticket.Category != nil && hasStatusChange(changes) {
		statusSubscribers, err := s.repo.GetCategoryEventSubscribers(ctx, ticket.Category.ID, models.EventFieldName(models.EventStatus))
		if err != nil {
			return fmt.Errorf("failed to get status-change subscribers by category: %w", err)
		}
		for _, id := range statusSubscribers {
			recipients[id] = struct{}{}
		}
	}

	// Автор изменения не уведомляется о собственном действии.
	delete(recipients, actorID)

	if len(recipients) == 0 {
		return nil
	}

	changesData, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}
	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketUpdated),
		Title: "Задача обновлена",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, recipientsToSlice(recipients), dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// TicketDeleted оповещает менеджера, ответственных категории и подписанных об удалении тикета.
func (s *NotificationService) TicketDeleted(ctx context.Context, ticket *models.Ticket) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	responsible, err := s.repo.GetResponsibleByCategory(ctx, ticket.Category.ID)
	if err != nil {
		return fmt.Errorf("failed to get responsible by category: %w", err)
	}
	for _, id := range responsible {
		recipients[id] = struct{}{}
	}

	subscribers, err := s.subscriptions.GetByTicket(ctx, ticket.ID)
	if err != nil {
		return fmt.Errorf("failed to get ticket subscribers: %w", err)
	}
	for _, id := range subscribers {
		recipients[id] = struct{}{}
	}

	if len(recipients) == 0 {
		return nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketDeleted),
		Title: "Задача удалена",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, recipientsToSlice(recipients), dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// TicketCommented оповещает исполнителя и подписанных о новом комментарии по заявке.
// Самому автору комментария уведомление не отправляется.
func (s *NotificationService) TicketCommented(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil && ticket.Assignee.ID != actorID {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	// Подписанные на заявку, у которых включено событие «Комментарий» для категории тикета.
	commentSubs, err := s.subscribersByEvent(ctx, ticket, models.EventComment)
	if err != nil {
		return err
	}
	for _, id := range commentSubs {
		if id != actorID {
			recipients[id] = struct{}{}
		}
	}

	// Подписчики категории на событие «Комментарий».
	if ticket.Category != nil {
		commentSubscribers, err := s.repo.GetCategoryEventSubscribers(ctx, ticket.Category.ID, models.EventFieldName(models.EventComment))
		if err != nil {
			return fmt.Errorf("failed to get comment subscribers by category: %w", err)
		}
		for _, id := range commentSubscribers {
			recipients[id] = struct{}{}
		}
	}

	delete(recipients, actorID)

	if len(recipients) == 0 {
		return nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketComment),
		Title: "Новый комментарий",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, recipientsToSlice(recipients), dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// AttachmentAdded оповещает исполнителя и подписанных о новом вложении по заявке.
// Самому автору вложения уведомление не отправляется.
func (s *NotificationService) AttachmentAdded(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) error {
	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil && ticket.Assignee.ID != actorID {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	// Подписанные на заявку, у которых включено событие «Комментарий» для категории тикета.
	commentSubs, err := s.subscribersByEvent(ctx, ticket, models.EventComment)
	if err != nil {
		return err
	}
	for _, id := range commentSubs {
		if id != actorID {
			recipients[id] = struct{}{}
		}
	}

	delete(recipients, actorID)

	if len(recipients) == 0 {
		return nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketAttachment),
		Title: "Новое вложение",
		Body:  ticket.Title,
		Data:  data,
	}
	persisted := s.persistForUsers(ctx, recipientsToSlice(recipients), dto)
	s.deliver(ctx, ticket, persisted, dto)

	return nil
}

// GetOverdueTicketIDs возвращает ID активных тикетов с просроченным сроком.
func (s *NotificationService) GetOverdueTicketIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	return s.repo.GetOverdueTicketIDs(ctx, now)
}

// GetSettings возвращает персональные настройки уведомлений пользователя.
func (s *NotificationService) GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error) {
	return s.repo.GetSettings(ctx, userID)
}

// SaveSettings сохраняет персональные настройки уведомлений пользователя.
func (s *NotificationService) SaveSettings(ctx context.Context, userID uuid.UUID, settings json.RawMessage) error {
	return s.repo.SaveSettings(ctx, nil, userID, settings)
}

// GetSettingsPayload возвращает типизированные настройки уведомлений пользователя.
// Пустые/отсутствующие настройки трактуются как значения по умолчанию.
func (s *NotificationService) GetSettingsPayload(ctx context.Context, userID uuid.UUID) (*models.NotificationSettingsPayload, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	payload := models.DefaultNotificationSettings()
	if len(settings.Settings) > 0 {
		json.Unmarshal(settings.Settings, payload) // nolint:errcheck — при битых данных остаются дефолты
	}
	if payload.Categories == nil {
		payload.Categories = []models.CategoryNotificationSetting{}
	}
	if payload.Groups == nil {
		payload.Groups = []models.GroupNotificationSetting{}
	}
	return payload, nil
}

// SaveSettingsPayload сохраняет типизированные настройки уведомлений пользователя.
func (s *NotificationService) SaveSettingsPayload(ctx context.Context, userID uuid.UUID, settings *models.NotificationSettingsPayload) error {
	if settings.Categories == nil {
		settings.Categories = []models.CategoryNotificationSetting{}
	}
	if settings.Groups == nil {
		settings.Groups = []models.GroupNotificationSetting{}
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal notification settings: %w", err)
	}
	return s.repo.SaveSettings(ctx, nil, userID, raw)
}

// hasStatusChange возвращает true, если среди изменений тикета есть смена статуса.
func hasStatusChange(changes []*models.FieldChange) bool {
	for _, ch := range changes {
		if ch.Tag == models.ActionStatusChanged || ch.Tag == models.ActionClosed {
			return true
		}
	}
	return false
}

// subscribersByEvent возвращает подписанных на заявку пользователей, у которых для категории
// тикета включено событие event (status/comment/overdue). Если у тикета нет категории —
// возвращает всех подписанных.
func (s *NotificationService) subscribersByEvent(ctx context.Context, ticket *models.Ticket, event string) ([]uuid.UUID, error) {
	if ticket.Category == nil {
		return s.subscriptions.GetByTicket(ctx, ticket.ID)
	}
	return s.subscriptions.GetSubscribersByEvent(ctx, ticket.ID, ticket.Category.ID, models.EventFieldName(event))
}

// notifEnabled возвращает true, если у пользователя включён мастер-переключатель уведомлений
// (settings.enabled). По умолчанию (нет настроек) — включён.
func (s *NotificationService) notifEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return false, err
	}
	var payload models.NotificationSettingsPayload
	if len(settings.Settings) > 0 {
		json.Unmarshal(settings.Settings, &payload) // nolint:errcheck — при битых данных остаётся дефолт
	}
	return payload.Enabled, nil
}

// autoSubscribeOnCreate подписывает на создаваемую заявку надзителей реалма и менеджера группы
// (у кого включены уведомления) и возвращает их ID, чтобы они получили уведомление о создании.
func (s *NotificationService) autoSubscribeOnCreate(ctx context.Context, ticket *models.Ticket) ([]uuid.UUID, error) {
	candidates := make(map[uuid.UUID]struct{})

	if ticket.RealmID != nil {
		supervisors, err := s.userRealms.GetRealmSupervisors(ctx, *ticket.RealmID)
		if err != nil {
			return nil, err
		}
		for _, id := range supervisors {
			candidates[id] = struct{}{}
		}
	}

	if ticket.Group != nil {
		group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: ticket.Group.ID})
		if err != nil {
			return nil, err
		}
		if group.ManagerID != nil {
			candidates[*group.ManagerID] = struct{}{}
		}
	}

	var subscribed []uuid.UUID
	for id := range candidates {
		enabled, err := s.notifEnabled(ctx, id)
		if err != nil {
			continue
		}
		if !enabled {
			continue
		}
		if err := s.subscriptions.SubscribeInternal(ctx, ticket.ID, id); err != nil {
			return nil, err
		}
		subscribed = append(subscribed, id)
	}
	return subscribed, nil
}

// SendUnread отправляет клиенту непрочитанные уведомления и помечает их прочитанными.
func (s *NotificationService) SendUnread(ctx context.Context, client *ws_hub.Client) error {
	notifications, err := s.repo.GetUnread(ctx, client.UserID)
	if err != nil {
		return fmt.Errorf("failed to get unread notifications: %w", err)
	}

	for _, n := range notifications {
		if err := client.SendJSON("notification", n); err != nil {
			logger.Warn("failed to send unread notification", logger.StringAttr("user_id", client.UserID.String()), logger.ErrAttr(err))
		}
	}

	if len(notifications) > 0 {
		if err := s.repo.MarkAllRead(ctx, nil, client.UserID); err != nil {
			logger.Warn("failed to mark notifications as read", logger.StringAttr("user_id", client.UserID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// persist сохраняет уведомление в БД. Запись выполняется в отдельной транзакции на каждого
// получателя, чтобы сбой одного не откатывал создание уведомлений остальным.
func (s *NotificationService) persist(ctx context.Context, dto *models.CreateNotificationDTO) error {
	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Create(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create notification: %w", err)
		}
		return nil
	})
}

// persistForUsers сохраняет уведомление каждому получателю и возвращает только тех, кому
// строка реально создана в БД — по ним затем выполняется доставка по каналам (чтобы, например,
// cron не спамил Mattermost каждым прогоном, если запись не удалась). Ошибки записи логируются.
func (s *NotificationService) persistForUsers(ctx context.Context, userIDs []uuid.UUID, dto *models.CreateNotificationDTO) []uuid.UUID {
	var persisted []uuid.UUID
	for _, userID := range userIDs {
		dto.UserID = userID
		if err := s.persist(ctx, dto); err != nil {
			logger.Warn("failed to save notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
			continue
		}
		persisted = append(persisted, userID)
	}
	return persisted
}

// deliver рассылает уже сохранённые уведомления по всем включённым каналам (best-effort).
// Канал сам решает, может ли он доставить этому пользователю; ошибки лишь логируются.
func (s *NotificationService) deliver(ctx context.Context, ticket *models.Ticket, userIDs []uuid.UUID, dto *models.CreateNotificationDTO) {
	for _, ch := range s.channels {
		for _, userID := range userIDs {
			if err := ch.Notify(ctx, userID, dto, ticket); err != nil {
				logger.Warn("failed to deliver notification",
					logger.StringAttr("channel", ch.Name()),
					logger.StringAttr("user_id", userID.String()),
					logger.ErrAttr(err),
				)
			}
		}
	}
}

// recipientsToSlice превращает множество получателей в слайс.
func recipientsToSlice(recipients map[uuid.UUID]struct{}) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(recipients))
	for id := range recipients {
		ids = append(ids, id)
	}
	return ids
}
