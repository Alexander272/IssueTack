package services

import (
	"context"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func mattermostNotifierServiceFixtures() (*MockMattermostRepo, *MockUserService, *MockMMSender, *mattermostNotifier) {
	return mattermostNotifierFixturesWithPolicy(true, nil)
}

// mattermostNotifierFixturesWithPolicy собирает нотификатор с заданным ответом
// Enforce на coarse ticket:read, которым решается показ веб-ссылки в DM.
func mattermostNotifierFixturesWithPolicy(allowed bool, enforceErr error) (*MockMattermostRepo, *MockUserService, *MockMMSender, *mattermostNotifier) {
	mockMMRepo := new(MockMattermostRepo)
	mockUsers := new(MockUserService)
	mockSender := new(MockMMSender)
	mockPolicies := new(MockAccessPolicies)
	mockPolicies.On("Enforce", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(allowed, enforceErr)

	svc := NewMattermostNotifier(mockMMRepo, mockUsers, mockSender, mockPolicies, "http://localhost:9000")
	return mockMMRepo, mockUsers, mockSender, svc
}

func mmTicket() *models.Ticket {
	realmID := uuid.New()
	num := 12
	return &models.Ticket{
		ID:           uuid.New(),
		Title:        "Тестовая заявка",
		TicketNumber: &num,
		RealmID:      &realmID,
	}
}

func TestMattermostNotifier_NoRealm_NoSend(t *testing.T) {
	_, _, mockSender, svc := mattermostNotifierServiceFixtures()

	ticket := &models.Ticket{ID: uuid.New(), Title: "Без реалма"}
	dto := &models.CreateNotificationDTO{Type: string(models.NotificationTicketCreated), Title: "Новая задача", Body: "Без реалма"}

	delivered, err := svc.Notify(context.Background(), uuid.New(), dto, ticket)
	assert.NoError(t, err)
	assert.False(t, delivered)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMattermostNotifier_InactiveRealm_NoSend(t *testing.T) {
	mockMMRepo, _, mockSender, svc := mattermostNotifierServiceFixtures()

	ticket := mmTicket()
	realmID := *ticket.RealmID
	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, IsActive: false}, nil)

	delivered, err := svc.Notify(context.Background(), uuid.New(), &models.CreateNotificationDTO{}, ticket)
	assert.NoError(t, err)
	assert.False(t, delivered)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMattermostNotifier_NoMattermostUser_NoSend(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	ticket := mmTicket()
	realmID := *ticket.RealmID
	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: nil}, nil)

	delivered, err := svc.Notify(context.Background(), userID, &models.CreateNotificationDTO{}, ticket)
	assert.NoError(t, err)
	assert.False(t, delivered)
	mockSender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestMattermostNotifier_SendsCreatedDM(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-1"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketCreated),
		Title: "Новая задача",
		Body:  "Тестовая заявка",
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "**Новая задача №12: Тестовая заявка**") &&
			assert.Contains(t, msg, "http://localhost:9000/tasks/"+ticket.ID.String()) &&
			assert.Contains(t, msg, "[Открыть в плагине](/plug/issuetrack/ticket/"+ticket.ID.String()+")")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

func TestMattermostNotifier_SendsUpdatedDMWithChanges(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-2"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	changesData, err := json.Marshal([]*models.FieldChange{
		{Tag: models.ActionStatusChanged, OldVal: "open", NewVal: "in_progress"},
		{Tag: models.ActionPriorityChanged, OldVal: "none", NewVal: "high"},
	})
	assert.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	assert.NoError(t, err)
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketUpdated),
		Title: "Задача обновлена",
		Body:  "Тестовая заявка",
		Data:  data,
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "**Задача №12 обновлена: Тестовая заявка**") &&
			assert.Contains(t, msg, "• Статус: Новая → В работе") &&
			assert.Contains(t, msg, "• Приоритет: — → Высокий") &&
			// Ветка format с action != "" тоже содержит обе ссылки.
			assert.Contains(t, msg, "[Открыть в плагине](/plug/issuetrack/ticket/"+ticket.ID.String()+")")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

func TestMattermostNotifier_SendsOverdueDMWithDeadline(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-overdue"
	dueDate := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	ticket := mmTicket()
	ticket.DueDate = &dueDate
	realmID := *ticket.RealmID
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketOverdue),
		Title: "Задача просрочена",
		Body:  "Тестовая заявка",
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "**Задача №12 просрочена: Тестовая заявка**") &&
			assert.Contains(t, msg, "Дедлайн: 02.09.2026 12:00") &&
			assert.Contains(t, msg, "http://localhost:9000/tasks/"+ticket.ID.String())
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

func TestMattermostNotifier_SendsDeadlineSoonDMWithDeadline(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-soon"
	dueDate := time.Date(2026, 9, 5, 18, 0, 0, 0, time.UTC)
	ticket := mmTicket()
	ticket.DueDate = &dueDate
	realmID := *ticket.RealmID
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationDeadlineSoon),
		Title: "Скоро срок",
		Body:  "Тестовая заявка",
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "**Скоро срок №12: Тестовая заявка**") &&
			assert.Contains(t, msg, "Дедлайн: 05.09.2026 18:00") &&
			assert.Contains(t, msg, "http://localhost:9000/tasks/"+ticket.ID.String())
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

func TestMattermostNotifier_SendError_Returned(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-3"
	ticket := mmTicket()
	realmID := *ticket.RealmID

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Return(assert.AnError).Once()

	delivered, err := svc.Notify(context.Background(), userID, &models.CreateNotificationDTO{}, ticket)
	assert.Error(t, err)
	assert.False(t, delivered)
	mockSender.AssertExpectations(t)
}

func TestMattermostNotifier_Name(t *testing.T) {
	_, _, _, svc := mattermostNotifierServiceFixtures()
	assert.Equal(t, "mattermost", svc.Name())
}

// Проверка гейта веб-ссылки: маршрут /tasks/:id закрыт Casbin-мидлваром
// ticket:read, поэтому ссылка на веб включается только получателю с этим правом.
// Plugin-ссылка не гейтится — её маршрут закрыт SourceGuard/PluginTokenGuard,
// а доступ к заявке решается на уровне сервиса по атрибутам.
func TestMattermostNotifier_Links_WebLinkGatedByTicketRead(t *testing.T) {
	tests := []struct {
		name        string
		allowed     bool
		enforceErr  error
		wantWebLink bool
	}{
		{"есть ticket:read — веб-ссылка есть", true, nil, true},
		{"нет ticket:read — только ссылка плагина", false, nil, false},
		{"ошибка проверки — только ссылка плагина, DM доставлен", false, assert.AnError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierFixturesWithPolicy(tt.allowed, tt.enforceErr)

			userID := uuid.New()
			mmID := "mm-user-links"
			ticket := mmTicket()
			realmID := *ticket.RealmID
			dto := &models.CreateNotificationDTO{
				Type:  string(models.NotificationTicketCreated),
				Title: "Новая задача",
				Body:  "Тестовая заявка",
			}

			mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
			mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)

			var sent string
			mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
				sent = args.String(3)
			}).Return(nil).Once()

			delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
			assert.NoError(t, err)
			assert.True(t, delivered)

			webLink := "Открыть: http://localhost:9000/tasks/" + ticket.ID.String()
			pluginLink := "[Открыть в плагине](/plug/issuetrack/ticket/" + ticket.ID.String() + ")"
			if tt.wantWebLink {
				assert.Contains(t, sent, webLink)
			} else {
				assert.NotContains(t, sent, "Открыть: ")
				assert.NotContains(t, sent, "http://localhost:9000/tasks/")
			}
			assert.Contains(t, sent, pluginLink)
			mockSender.AssertExpectations(t)
		})
	}
}

// Ветка события с действием («Задача … обновлена») использует тот же блок
// ссылок, что и обычные уведомления, — правило гейта проверяется и на ней.
func TestMattermostNotifier_Links_WebLinkGatedOnUpdatedEvent(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierFixturesWithPolicy(false, nil)

	userID := uuid.New()
	mmID := "mm-user-links-updated"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketUpdated),
		Title: "Задача обновлена",
		Body:  "Тестовая заявка",
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)

	var sent string
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
		sent = args.String(3)
	}).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	assert.Contains(t, sent, "**Задача №12 обновлена: Тестовая заявка**")
	assert.NotContains(t, sent, "Открыть: ")
	assert.Contains(t, sent, "[Открыть в плагине](/plug/issuetrack/ticket/"+ticket.ID.String()+")")
	mockSender.AssertExpectations(t)
}

// Домен проверки — realm тикета, а не realm из заголовка запроса: тикет грузится
// по id без realm-фильтра, поэтому сверять Enforce надо именно с ticket.RealmID.
func TestMattermostNotifier_Links_EnforceUsesTicketRealm(t *testing.T) {
	mockMMRepo, mockUsers, mockSender := new(MockMattermostRepo), new(MockUserService), new(MockMMSender)
	mockPolicies := new(MockAccessPolicies)
	svc := NewMattermostNotifier(mockMMRepo, mockUsers, mockSender, mockPolicies, "http://localhost:9000")

	userID := uuid.New()
	mmID := "mm-user-realm"
	ticket := mmTicket()
	realmID := *ticket.RealmID

	mockPolicies.On("Enforce", userID.String(), realmID.String(), "ticket", "read").Return(true, nil).Once()
	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)

	var sent string
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
		sent = args.String(3)
	}).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketCreated), Body: "Тестовая заявка",
	}, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	assert.Contains(t, sent, "Открыть: http://localhost:9000/tasks/"+ticket.ID.String())
	mockPolicies.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

// Без собранного AccessPolicies (nil) веб-ссылка не печатается, но DM всё
// равно доставляется — гейт не должен ломать доставку уведомления.
func TestMattermostNotifier_Links_NilPolicies_StillDelivers(t *testing.T) {
	mockMMRepo := new(MockMattermostRepo)
	mockUsers := new(MockUserService)
	mockSender := new(MockMMSender)
	svc := NewMattermostNotifier(mockMMRepo, mockUsers, mockSender, nil, "http://localhost:9000")

	userID := uuid.New()
	mmID := "mm-user-nil"
	ticket := mmTicket()
	realmID := *ticket.RealmID

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)

	var sent string
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
		sent = args.String(3)
	}).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketCreated), Body: "Тестовая заявка",
	}, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	assert.NotContains(t, sent, "Открыть: ")
	assert.Contains(t, sent, "[Открыть в плагине](/plug/issuetrack/ticket/"+ticket.ID.String()+")")
	mockSender.AssertExpectations(t)
}

// Пользователю из Mattermost веб-ссылка не показывается даже при наличии
// coarse-права: учётной записи в Keycloak у него нет, в веб он не входит.
func TestMattermostNotifier_Links_MattermostSource_NoWebLink(t *testing.T) {
	mockMMRepo, mockUsers, mockSender := new(MockMattermostRepo), new(MockUserService), new(MockMMSender)
	mockPolicies := new(MockAccessPolicies)
	svc := NewMattermostNotifier(mockMMRepo, mockUsers, mockSender, mockPolicies, "http://localhost:9000")

	userID := uuid.New()
	mmID := "mm-user-src"
	ticket := mmTicket()
	realmID := *ticket.RealmID

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{
		ID: userID, MattermostID: &mmID, Source: models.UserSourceMattermost,
	}, nil)

	var sent string
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
		sent = args.String(3)
	}).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketCreated), Body: "Тестовая заявка",
	}, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	assert.NotContains(t, sent, "Открыть: ")
	assert.Contains(t, sent, "[Открыть в плагине](/plug/issuetrack/ticket/"+ticket.ID.String()+")")
	mockPolicies.AssertNotCalled(t, "Enforce", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockSender.AssertExpectations(t)
}

// При создании заявки DM дополняется строкой «Заказчик: <ФИО>» для того,
// кому задача предназначена.
func TestMattermostNotifier_SendsCreatedDM_WithOwner(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-owner"
	ticket := mmTicket()
	ticket.Owner = &models.UserShort{LastName: "Петров", FirstName: "Иван", Username: "ivan"}
	realmID := *ticket.RealmID
	dto := &models.CreateNotificationDTO{
		Type:  string(models.NotificationTicketCreated),
		Title: "Новая задача",
		Body:  "Тестовая заявка",
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "**Новая задача №12: Тестовая заявка**") &&
			assert.Contains(t, msg, "Заказчик: Петров Иван")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

// Смена статуса + ожидаемый получатель MakeDM: только статус, только русские значения
func TestMattermostNotifier_SendsUpdatedDM_StatusRussianValues(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-ru"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	changesData, err := json.Marshal([]*models.FieldChange{
		{Tag: models.ActionStatusChanged, OldVal: "in_progress", NewVal: "resolved"},
		{Tag: models.ActionPriorityChanged, OldVal: "medium", NewVal: "urgent"},
	})
	assert.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	assert.NoError(t, err)
	dto := &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketUpdated),
		Body: "Тестовая заявка",
		Data: data,
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "• Статус: В работе → Решена") &&
			assert.Contains(t, msg, "• Приоритет: Средний → Срочный") &&
			assert.NotContains(t, msg, "in_progress") &&
			assert.NotContains(t, msg, "status_changed")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

// Передача заявки самому исполнителю: «Вам назначена задача.» без id исполнителей.
func TestMattermostNotifier_SendsUpdatedDM_AssignedToRecipient(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-assigned"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	changesData, err := json.Marshal([]*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "none", NewVal: userID.String()},
	})
	assert.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	assert.NoError(t, err)
	dto := &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketUpdated),
		Body: "Тестовая заявка",
		Data: data,
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "Вам назначена задача.") &&
			assert.NotContains(t, msg, userID.String()) &&
			assert.NotContains(t, msg, "assigned")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

// Тот же DM остальным получателям (менеджер/подписчики): нейтральная строка без id.
func TestMattermostNotifier_SendsUpdatedDM_AssignedOtherRecipient(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	assigneeID := uuid.New()
	mmID := "mm-user-not-assigned"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	changesData, err := json.Marshal([]*models.FieldChange{
		{Tag: models.ActionAssignChanged, OldVal: uuid.New().String(), NewVal: assigneeID.String()},
	})
	assert.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	assert.NoError(t, err)
	dto := &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketUpdated),
		Body: "Тестовая заявка",
		Data: data,
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	mockSender.On("Send", "bt", "bb", mmID, mock.MatchedBy(func(msg string) bool {
		return assert.Contains(t, msg, "Исполнитель: изменён") &&
			assert.NotContains(t, msg, assigneeID.String()) &&
			assert.NotContains(t, msg, userID.String()) &&
			assert.NotContains(t, msg, "Вам назначена")
	})).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	mockSender.AssertExpectations(t)
}

// Прочие id-поля (заказчик/начальник/группа/категория/площадка) и срок выводятся
// без сырых uuid: подпись по-русски, срок — в формате «02.01.2006 15:04».
func TestMattermostNotifier_SendsUpdatedDM_NoRawIDs(t *testing.T) {
	mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

	userID := uuid.New()
	mmID := "mm-user-noids"
	ticket := mmTicket()
	realmID := *ticket.RealmID
	changesData, err := json.Marshal([]*models.FieldChange{
		{Tag: models.ActionOwnerChanged, OldVal: uuid.New().String(), NewVal: uuid.New().String()},
		{Tag: models.ActionManagerChanged, OldVal: "none", NewVal: uuid.New().String()},
		{Tag: models.ActionGroupChanged, OldVal: uuid.New().String(), NewVal: "none"},
		{Tag: models.ActionSiteChanged, OldVal: uuid.New().String(), NewVal: uuid.New().String()},
		{Tag: models.ActionCategoryChanged, OldVal: "none", NewVal: uuid.New().String()},
		{Tag: models.ActionDueDateChanged, OldVal: "2026-09-02 12:00:00 +0000 UTC", NewVal: "none"},
	})
	assert.NoError(t, err)
	data, err := json.Marshal(map[string]any{
		"ticket_id": ticket.ID.String(),
		"title":     ticket.Title,
		"changes":   string(changesData),
	})
	assert.NoError(t, err)
	dto := &models.CreateNotificationDTO{
		Type: string(models.NotificationTicketUpdated),
		Body: "Тестовая заявка",
		Data: data,
	}

	mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
	mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)
	var sent string
	mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
		sent = args.String(3)
	}).Return(nil).Once()

	delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
	assert.NoError(t, err)
	assert.True(t, delivered)
	assert.Contains(t, sent, "Заказчик: изменён")
	assert.Contains(t, sent, "Начальник: изменён")
	assert.Contains(t, sent, "Группа: изменена")
	assert.Contains(t, sent, "Площадка: изменена")
	assert.Contains(t, sent, "Категория: изменена")
	assert.Contains(t, sent, "• Срок: 02.09.2026 12:00 → —")
	assert.NotContains(t, sent, "uuid")
	mockSender.AssertExpectations(t)
}

// DM канала «Новый комментарий» (исполнителю/подписчикам): после шапки идёт
// «Комментарий пользователя <ФИО>:» и текст комментария. Без автора — только текст.
func TestMattermostNotifier_SendsCommentedDM_WithAuthorAndText(t *testing.T) {
	tests := []struct {
		name   string
		author string
		want   []string
	}{
		{
			name:   "автор известен — «Комментарий пользователя …:» + текст",
			author: "Петров Иван",
			want:   []string{"Комментарий пользователя Петров Иван:", "уточните сроки"},
		},
		{
			name: "автор не определён — только текст",
			want: []string{"уточните сроки"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMMRepo, mockUsers, mockSender, svc := mattermostNotifierServiceFixtures()

			userID := uuid.New()
			mmID := "mm-user-comment"
			ticket := mmTicket()
			realmID := *ticket.RealmID
			data, err := json.Marshal(map[string]any{
				"ticket_id": ticket.ID.String(),
				"title":     ticket.Title,
				"comment": map[string]string{
					"text":   "уточните сроки",
					"author": tt.author,
				},
			})
			assert.NoError(t, err)
			dto := &models.CreateNotificationDTO{
				Type: string(models.NotificationTicketComment),
				Body: "Тестовая заявка",
				Data: data,
			}

			mockMMRepo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, BotToken: "bt", BotUserID: "bb", IsActive: true}, nil)
			mockUsers.On("GetByID", mock.Anything, userID).Return(&models.UserData{ID: userID, MattermostID: &mmID}, nil)

			var sent string
			mockSender.On("Send", "bt", "bb", mmID, mock.Anything).Run(func(args mock.Arguments) {
				sent = args.String(3)
			}).Return(nil).Once()

			delivered, err := svc.Notify(context.Background(), userID, dto, ticket)
			assert.NoError(t, err)
			assert.True(t, delivered)
			assert.Contains(t, sent, "**Новый комментарий №12: Тестовая заявка**")
			for _, want := range tt.want {
				assert.Contains(t, sent, want)
			}
			if tt.author == "" {
				assert.NotContains(t, sent, "Комментарий пользователя")
			}
			mockSender.AssertExpectations(t)
		})
	}
}
