package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func notificationServiceFixtures() (*MockNotificationsRepo, *ws_hub.Hub, *NotificationService) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	mockGroups := new(MockGroupsRepo)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetSubscribersByEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockGroups.On("GetByID", mock.Anything, mock.Anything).Return(&models.Group{}, nil).Maybe()
	mockRepo.On("GetCategoryEventSubscribers", mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockRepo.On("GetGroupEventSubscribers", mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockRepo.On("GetOverdueTicketIDs", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		groups:        mockGroups,
		txManager:     &mockTransactionManager{},
	}
	return mockRepo, hub, svc
}

func TestNotificationService_TicketCreated_Success(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{
			ID: categoryID,
		},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":true}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketCreated(context.Background(), ticket)
	assert.NoError(t, err)
}

func TestNotificationService_TicketCreated_WithResponsible(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{
			ID: categoryID,
		},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("GetSettings", mock.Anything, respID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil).Twice()

	err := svc.TicketCreated(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_Success(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	actorID := uuid.New()
	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test",
		Manager: &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{
			ID: uuid.New(),
		},
	}
	changes := []*models.FieldChange{
		{Tag: "title", OldVal: "Old", NewVal: "New"},
	}

	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_ActionAssigned_SelfAssign(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	actorID := uuid.New()
	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:    uuid.New(),
		Title: "Test",
		Category: &models.CategoryShort{
			ID: categoryID,
		},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: actorID.String()},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	mockRepo.On("GetSettings", mock.Anything, respID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_ActionAssigned_Other(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	actorID := uuid.New()
	newAssigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:    uuid.New(),
		Title: "Test",
		Category: &models.CategoryShort{
			ID: uuid.New(),
		},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: newAssigneeID.String()},
	}

	mockRepo.On("GetSettings", mock.Anything, newAssigneeID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_InvalidAssigneeUUID(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	ticket := &models.Ticket{
		ID:    uuid.New(),
		Title: "Test",
		Category: &models.CategoryShort{
			ID: uuid.New(),
		},
	}
	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: "not-a-uuid"},
	}

	err := svc.TicketUpdated(context.Background(), ticket, uuid.New(), changes)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_TicketDeleted_Success(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Deleted Ticket",
		Manager:  &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{ID: categoryID},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketDeleted(context.Background(), ticket)
	assert.NoError(t, err)
}

func TestNotificationService_TicketDeleted_NoManager(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Deleted Ticket",
		Category: &models.CategoryShort{ID: categoryID},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	mockRepo.On("GetSettings", mock.Anything, respID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketDeleted(context.Background(), ticket)
	assert.NoError(t, err)
}

func TestNotificationService_NotifyOverdue_NotifiesAssigneeAndManager(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Overdue Ticket",
		Assignee: &models.UserShort{ID: assigneeID},
		Manager:  &models.UserShort{ID: managerID},
		Category: nil,
		Group:    nil,
		RealmID:  nil,
	}

	mockRepo.On("HasNotification", mock.Anything, mock.Anything, ticket.ID, string(models.NotificationTicketOverdue)).Return(false, nil).Twice()
	mockRepo.On("GetSettings", mock.Anything, assigneeID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":false}`)}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":false}`)}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()

	err := svc.NotifyOverdue(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationService_NotifyOverdue_NoDuplicate(t *testing.T) {
	mockRepo, _, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Overdue Ticket",
		Assignee: &models.UserShort{ID: assigneeID},
	}

	mockRepo.On("HasNotification", mock.Anything, assigneeID, ticket.ID, string(models.NotificationTicketOverdue)).Return(true, nil)

	err := svc.NotifyOverdue(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}
