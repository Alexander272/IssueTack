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
	mockMMRepo := new(MockMattermostRepo)
	mockUsers := new(MockUserService)
	mockSender := new(MockMMSender)

	svc := NewMattermostNotifier(mockMMRepo, mockUsers, mockSender, "http://localhost:9000")
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
			assert.Contains(t, msg, "http://localhost:9000/tasks/"+ticket.ID.String())
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
	data, err := json.Marshal(map[string]interface{}{
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
			assert.Contains(t, msg, "• status_changed: open → in_progress") &&
			assert.Contains(t, msg, "• priority_changed: — → high")
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
