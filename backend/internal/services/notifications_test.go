package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func notificationServiceFixtures() (*MockNotificationsRepo, *MockNotifier, *NotificationService) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	mockGroups := new(MockGroupsRepo)
	mockNotifier := new(MockNotifier)

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetSubscribersByEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockGroups.On("GetByID", mock.Anything, mock.Anything).Return(&models.Group{}, nil).Maybe()
	mockRepo.On("GetCategoryEventSubscribers", mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockRepo.On("GetGroupEventSubscribers", mock.Anything, mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockRepo.On("GetOverdueTicketIDs", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockNotifier.On("Name").Return("mock").Maybe()

	svc := &NotificationService{
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		groups:        mockGroups,
		txManager:     &mockTransactionManager{},
		channels:      []Notifier{mockNotifier},
	}
	return mockRepo, mockNotifier, svc
}

func expectDeliverAndPersist(mockRepo *MockNotificationsRepo, mockNotifier *MockNotifier, userIDs ...uuid.UUID) {
	for _, userID := range userIDs {
		mockNotifier.On("Notify", mock.Anything, userID, mock.Anything, mock.Anything).Return(true, nil).Once()
		mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil).Once()
	}
}

func TestNotificationService_TicketCreated_Success(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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
	expectDeliverAndPersist(mockRepo, mockNotifier, managerID)

	err := svc.TicketCreated(context.Background(), ticket, uuid.New())
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCreated_CreatorExcluded(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	creatorID := uuid.New()
	categoryID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: creatorID},
		Category: &models.CategoryShort{
			ID: categoryID,
		},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{}, nil)

	err := svc.TicketCreated(context.Background(), ticket, creatorID)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_TicketCreated_WithResponsible(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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
	expectDeliverAndPersist(mockRepo, mockNotifier, managerID, respID)

	err := svc.TicketCreated(context.Background(), ticket, uuid.New())
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCreated_NotifiesWithType(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: managerID},
	}

	mockRepo.On("Create", mock.Anything, nil, mock.MatchedBy(func(dto *models.CreateNotificationDTO) bool {
		return dto.Type == string(models.NotificationTicketCreated) && dto.Title == "Новая задача"
	})).Return(nil).Once()
	mockNotifier.On("Notify", mock.Anything, managerID, mock.Anything, mock.Anything).Return(true, nil).Once()

	err := svc.TicketCreated(context.Background(), ticket, uuid.New())
	assert.NoError(t, err)
}

func TestNotificationService_TicketCreated_NotDelivered_NotPersisted(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: managerID},
	}

	mockNotifier.On("Notify", mock.Anything, managerID, mock.Anything, mock.Anything).Return(false, nil).Once()

	err := svc.TicketCreated(context.Background(), ticket, uuid.New())
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCreated_ChannelError_NotPersisted(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test Ticket",
		Manager: &models.UserShort{ID: managerID},
	}

	mockNotifier.On("Notify", mock.Anything, managerID, mock.Anything, mock.Anything).Return(false, assert.AnError).Once()

	err := svc.TicketCreated(context.Background(), ticket, uuid.New())
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_Success(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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

	expectDeliverAndPersist(mockRepo, mockNotifier, managerID)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_ActorExcluded(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Test",
		Manager: &models.UserShort{ID: actorID},
		Category: &models.CategoryShort{
			ID: uuid.New(),
		},
	}
	changes := []*models.FieldChange{
		{Tag: "title", OldVal: "Old", NewVal: "New"},
	}

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_TicketUpdated_ActionAssigned_SelfAssign(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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
	expectDeliverAndPersist(mockRepo, mockNotifier, respID)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_ActionAssigned_Other(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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

	expectDeliverAndPersist(mockRepo, mockNotifier, newAssigneeID)

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
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

func TestNotificationService_TicketUpdated_StatusChange_AssigneeNotified(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Test",
		Assignee: &models.UserShort{ID: assigneeID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionStatusChanged, OldVal: "open", NewVal: "in_progress"},
	}

	expectDeliverAndPersist(mockRepo, mockNotifier, assigneeID)

	err := svc.TicketUpdated(context.Background(), ticket, uuid.New(), changes)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketUpdated_StatusChange_AssigneeIsActor(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Test",
		Assignee: &models.UserShort{ID: actorID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	changes := []*models.FieldChange{
		{Tag: models.ActionStatusChanged, OldVal: "open", NewVal: "in_progress"},
	}

	err := svc.TicketUpdated(context.Background(), ticket, actorID, changes)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_TicketDeleted_Success(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	managerID := uuid.New()
	categoryID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Deleted Ticket",
		Manager:  &models.UserShort{ID: managerID},
		Category: &models.CategoryShort{ID: categoryID},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{}, nil)
	expectDeliverAndPersist(mockRepo, mockNotifier, managerID)

	err := svc.TicketDeleted(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketDeleted_NoManager(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	categoryID := uuid.New()
	respID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Deleted Ticket",
		Category: &models.CategoryShort{ID: categoryID},
	}

	mockRepo.On("GetResponsibleByCategory", mock.Anything, categoryID).Return([]uuid.UUID{respID}, nil)
	expectDeliverAndPersist(mockRepo, mockNotifier, respID)

	err := svc.TicketDeleted(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCommented_NotifiesAssignee(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Comment Ticket",
		Assignee: &models.UserShort{ID: assigneeID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	expectDeliverAndPersist(mockRepo, mockNotifier, assigneeID)

	err := svc.TicketCommented(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCommented_ActorExcluded(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Comment Ticket",
		Assignee: &models.UserShort{ID: actorID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	err := svc.TicketCommented(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	mockNotifier.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_AttachmentAdded_NotifiesAssignee(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Attachment Ticket",
		Assignee: &models.UserShort{ID: assigneeID},
		Category: &models.CategoryShort{ID: uuid.New()},
	}

	expectDeliverAndPersist(mockRepo, mockNotifier, assigneeID)

	err := svc.AttachmentAdded(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_NotifyOverdue_NotifiesAssigneeAndManager(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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
	expectDeliverAndPersist(mockRepo, mockNotifier, assigneeID, managerID)

	err := svc.NotifyOverdue(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_NotifyOverdue_NoDuplicate(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

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
	mockNotifier.AssertNotCalled(t, "Notify", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_NotifyOverdue_PartialDeliver_PersistsOnlyDelivered(t *testing.T) {
	mockRepo, mockNotifier, svc := notificationServiceFixtures()

	assigneeID := uuid.New()
	managerID := uuid.New()
	ticket := &models.Ticket{
		ID:       uuid.New(),
		Title:    "Overdue Ticket",
		Assignee: &models.UserShort{ID: assigneeID},
		Manager:  &models.UserShort{ID: managerID},
	}

	mockRepo.On("HasNotification", mock.Anything, mock.Anything, ticket.ID, string(models.NotificationTicketOverdue)).Return(false, nil).Twice()
	mockNotifier.On("Notify", mock.Anything, assigneeID, mock.Anything, mock.Anything).Return(true, nil).Once()
	mockNotifier.On("Notify", mock.Anything, managerID, mock.Anything, mock.Anything).Return(false, nil).Once()
	mockRepo.On("Create", mock.Anything, nil, mock.MatchedBy(func(dto *models.CreateNotificationDTO) bool {
		return dto.UserID == assigneeID
	})).Return(nil).Once()

	err := svc.NotifyOverdue(context.Background(), ticket)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}
