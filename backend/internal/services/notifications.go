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

// NotificationService — сервис уведомлений пользователей (сохранение в БД и push через WebSocket-хаб).
type NotificationService struct {
	hub           *ws_hub.Hub
	repo          repository.Notifications
	ticketRepo    repository.Tickets
	subscriptions repository.TicketSubscriptions
	txManager     TransactionManager
}

// NewNotificationService создаёт NotificationService.
func NewNotificationService(hub *ws_hub.Hub, repo repository.Notifications, ticketRepo repository.Tickets, subscriptions repository.TicketSubscriptions, txManager TransactionManager) *NotificationService {
	return &NotificationService{
		hub:           hub,
		repo:          repo,
		ticketRepo:    ticketRepo,
		subscriptions: subscriptions,
		txManager:     txManager,
	}
}

// Notifications — интерфейс уведомлений о событиях тикетов.
type Notifications interface {
	// TicketCreated оповещает заинтересованных пользователей о создании тикета.
	TicketCreated(ctx context.Context, dto *models.TicketDTO) error
	// TicketUpdated оповещает заинтересованных пользователей об обновлении тикета.
	TicketUpdated(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, changes []*models.FieldChange) error
	// TicketDeleted оповещает заинтересованных пользователей об удалении тикета.
	TicketDeleted(ctx context.Context, ticket *models.Ticket) error
	// TicketCommented оповещает исполнителя и подписанных о новом комментарии.
	TicketCommented(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID) error
	// AttachmentAdded оповещает исполнителя и подписанных о новом вложении.
	AttachmentAdded(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID) error
	// NotifyOverdue оповещает о просроченном тикете: исполнителя, менеджера, админов реалма,
	// а также подписчиков категории/группы на событие «Просрочка».
	NotifyOverdue(ctx context.Context, ticketID uuid.UUID) error
	// GetOverdueTicketIDs возвращает ID активных тикетов с просроченным сроком.
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
func (s *NotificationService) NotifyOverdue(ctx context.Context, ticketID uuid.UUID) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to get ticket for overdue notification: %w", err)
	}

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil {
		recipients[ticket.Assignee.ID] = struct{}{}
	}
	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}
	if ticket.RealmID != nil {
		if err := s.addRealmAdmins(ctx, *ticket.RealmID, recipients); err != nil {
			return err
		}
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

	for userID := range recipients {
		exists, err := s.repo.HasNotification(ctx, userID, ticketID, string(models.NotificationTicketOverdue))
		if err != nil {
			return fmt.Errorf("failed to check overdue notification: %w", err)
		}
		if exists {
			continue
		}
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   string(models.NotificationTicketOverdue),
			Title:  "Задача просрочена",
			Body:   ticket.Title,
			Data:   data,
		}
		if err := s.send(ctx, userID, dto); err != nil {
			logger.Warn("failed to send overdue notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// TicketCreated оповещает менеджера и ответственных категории о создании нового тикета.
func (s *NotificationService) TicketCreated(ctx context.Context, dto *models.TicketDTO) error {
	recipients := make(map[uuid.UUID]struct{})

	if dto.ManagerID != nil {
		recipients[*dto.ManagerID] = struct{}{}
	}

	responsible, err := s.repo.GetResponsibleByCategory(ctx, dto.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get responsible by category: %w", err)
	}
	for _, id := range responsible {
		recipients[id] = struct{}{}
	}

	if dto.RealmID != nil {
		if err := s.addRealmAdmins(ctx, *dto.RealmID, recipients); err != nil {
			return err
		}
	}

	// Пользователи, подписавшиеся на новые задачи в категории тикета (добровольная подписка).
	subscribers, err := s.repo.GetCategoryEventSubscribers(ctx, dto.CategoryID, models.EventFieldName(models.EventNewTask))
	if err != nil {
		return fmt.Errorf("failed to get new-task subscribers by category: %w", err)
	}
	for _, id := range subscribers {
		recipients[id] = struct{}{}
	}

	// Участники группы тикета, подписавшиеся на новые задачи этой группы.
	if dto.GroupID != nil {
		groupSubscribers, err := s.repo.GetGroupEventSubscribers(ctx, *dto.GroupID, models.EventFieldName(models.EventNewTask))
		if err != nil {
			return fmt.Errorf("failed to get new-task group subscribers: %w", err)
		}
		for _, id := range groupSubscribers {
			recipients[id] = struct{}{}
		}
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": dto.ID.String(),
		"title":     dto.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	for userID := range recipients {
		n := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.created",
			Title:  "Новая задача",
			Body:   dto.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, n); err != nil {
			logger.Warn("failed to send notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// TicketUpdated оповещает менеджера и новых исполнителей об обновлении тикета.
func (s *NotificationService) TicketUpdated(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, changes []*models.FieldChange) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to get ticket for notification: %w", err)
	}

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Manager != nil {
		recipients[ticket.Manager.ID] = struct{}{}
	}

	if ticket.RealmID != nil {
		if err := s.addRealmAdmins(ctx, *ticket.RealmID, recipients); err != nil {
			return err
		}
	}

	subscribers, err := s.subscriptions.GetByTicket(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket subscribers: %w", err)
	}
	for _, id := range subscribers {
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

	changesData, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}
	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   changesData,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.updated",
			Title:  "Задача обновлена",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			logger.Warn("failed to send notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// TicketDeleted оповещает менеджера и ответственных категории об удалении тикета.
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

	if ticket.RealmID != nil {
		if err := s.addRealmAdmins(ctx, *ticket.RealmID, recipients); err != nil {
			return err
		}
	}

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   "ticket.deleted",
			Title:  "Задача удалена",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			logger.Warn("failed to send notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// TicketCommented оповещает исполнителя и подписанных о новом комментарии по заявке.
// Самому автору комментария уведомление не отправляется.
func (s *NotificationService) TicketCommented(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to get ticket for comment notification: %w", err)
	}

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil && ticket.Assignee.ID != actorID {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	subscribers, err := s.subscriptions.GetByTicket(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket subscribers: %w", err)
	}
	for _, id := range subscribers {
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

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   string(models.NotificationTicketComment),
			Title:  "Новый комментарий",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			logger.Warn("failed to send notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

	return nil
}

// AttachmentAdded оповещает исполнителя и подписанных о новом вложении по заявке.
// Самому автору вложения уведомление не отправляется.
func (s *NotificationService) AttachmentAdded(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID) error {
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to get ticket for attachment notification: %w", err)
	}

	recipients := make(map[uuid.UUID]struct{})

	if ticket.Assignee != nil && ticket.Assignee.ID != actorID {
		recipients[ticket.Assignee.ID] = struct{}{}
	}

	subscribers, err := s.subscriptions.GetByTicket(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket subscribers: %w", err)
	}
	for _, id := range subscribers {
		if id != actorID {
			recipients[id] = struct{}{}
		}
	}

	delete(recipients, actorID)

	data, err := json.Marshal(map[string]interface{}{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	for userID := range recipients {
		dto := &models.CreateNotificationDTO{
			UserID: userID,
			Type:   string(models.NotificationTicketAttachment),
			Title:  "Новое вложение",
			Body:   ticket.Title,
			Data:   data,
		}

		if err := s.send(ctx, userID, dto); err != nil {
			logger.Warn("failed to send notification", logger.StringAttr("user_id", userID.String()), logger.ErrAttr(err))
		}
	}

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

// addRealmAdmins добавляет в получателей админов и root-пользователей реалма.
func (s *NotificationService) addRealmAdmins(ctx context.Context, realmID uuid.UUID, recipients map[uuid.UUID]struct{}) error {
	admins, err := s.repo.GetRealmAdmins(ctx, realmID)
	if err != nil {
		return err
	}
	for _, id := range admins {
		recipients[id] = struct{}{}
	}
	return nil
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

// send сохраняет уведомление в БД и, если в настройках пользователя включён push, отправляет его
// через WebSocket-хаб. Запись и отправка выполняются в одной транзакции, чтобы уведомление не
// осталось сохранённым, но не доставленным. Повреждённые настройки трактуются как включённый push
// (значение по умолчанию), чтобы уведомление не потерялось из-за битых данных.
func (s *NotificationService) send(ctx context.Context, userID uuid.UUID, dto *models.CreateNotificationDTO) error {
	settings, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get notification settings: %w", err)
	}

	var prefs map[string]bool
	if err := json.Unmarshal(settings.Settings, &prefs); err != nil {
		prefs = map[string]bool{}
	}
	// push не задан явно — считаем включённым (значение по умолчанию), чтобы уведомление
	// доставлялось через WebSocket, если пользователь не отключил push.
	push := true
	if v, ok := prefs["push"]; ok {
		push = v
	}

	return s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Create(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create notification: %w", err)
		}

		if push {
			eventData, err := json.Marshal(map[string]interface{}{
				"type":  dto.Type,
				"title": dto.Title,
				"body":  dto.Body,
				"data":  dto.Data,
			})
			if err != nil {
				logger.Warn("failed to marshal push event data", logger.ErrAttr(err))
			} else {
				s.hub.SendToUser(userID, eventData)
			}
		}

		return nil
	})
}
