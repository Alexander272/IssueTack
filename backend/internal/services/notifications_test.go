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

func notificationServiceFixtures() (*MockNotificationsRepo, *MockTicketsRepo, *ws_hub.Hub, *NotificationService) {
	mockRepo := new(MockNotificationsRepo)
	mockTicketRepo := new(MockTicketsRepo)
	hub := ws_hub.NewWebsocketHub()

	svc := &NotificationService{
		hub:        hub,
		repo:       mockRepo,
		ticketRepo: mockTicketRepo,
		txManager:  &mockTransactionManager{},
	}
	return mockRepo, mockTicketRepo, hub, svc
}

func TestNotificationService_TicketCreated_Success(t *testing.T) {
	mockRepo, _, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	dto := &models.TicketDTO{
		ID:         uuid.New(),
		Title:      "Test Ticket",
		ManagerID:  &managerID,
		CategoryID: categoryID,
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":true}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketCreated(context.Background(), dto)
	assert.NoError(t, err)
}

func TestNotificationService_TicketCreated_WithResponsible(t *testing.T) {
	mockRepo, _, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	respID := uuid.New()
	dto := &models.TicketDTO{
		ID:         uuid.New(),
		Title:      "Test Ticket",
		ManagerID:  &managerID,
		CategoryID: categoryID,
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("GetSettings", mock.Anything, respID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil).Twice()

	err := svc.TicketCreated(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_Success(t *testing.T) {
	mockRepo, mockTicketRepo, _, svc := notificationServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:      ticketID,
		Title:   "Test",
		Manager: &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}
	changes := []*models.FieldChange{
		{Tag: "title", OldVal: "Old", NewVal: "New"},
	}

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("GetSettings", mock.Anything, managerID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticketID, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_ActionAssigned_SelfAssign(t *testing.T) {
	mockRepo, mockTicketRepo, _, svc := notificationServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Category: &models.CategoryShort{ID: categoryID},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: actorID.String()},
	}

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	mockRepo.On("GetSettings", mock.Anything, respID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticketID, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_ActionAssigned_Other(t *testing.T) {
	mockRepo, mockTicketRepo, _, svc := notificationServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	newAssigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: newAssigneeID.String()},
	}

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("GetSettings", mock.Anything, newAssigneeID).Return(&models.NotificationSettings{
		Settings: []byte(`{"push":false}`),
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.TicketUpdated(context.Background(), ticketID, actorID, changes)
	assert.NoError(t, err)
}

func TestNotificationService_TicketUpdated_InvalidAssigneeUUID(t *testing.T) {
	mockRepo, mockTicketRepo, _, svc := notificationServiceFixtures()

	ticketID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Category: &models.CategoryShort{ID: uuid.New()},
	}
	changes := []*models.FieldChange{
		{Tag: models.ActionAssigned, OldVal: "", NewVal: "not-a-uuid"},
	}

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)

	err := svc.TicketUpdated(context.Background(), ticketID, uuid.New(), changes)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_TicketDeleted_Success(t *testing.T) {
	mockRepo, _, _, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	ticket := &models.Ticket{
		ID:    uuid.New(),
		Title: "Deleted Ticket",
		Manager: &models.UserShort{ID: managerID},
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
	mockRepo, _, _, svc := notificationServiceFixtures()

	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:    uuid.New(),
		Title: "Deleted Ticket",
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
