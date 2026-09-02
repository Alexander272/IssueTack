package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func subscriptionServiceFixtures() (*MockTicketSubscriptionsRepo, *MockTicketsService, *MockTicketAccessChecker, *TicketSubscriptionService) {
	mockRepo := new(MockTicketSubscriptionsRepo)
	mockTickets := new(MockTicketsService)
	mockAccess := new(MockTicketAccessChecker)

	svc := NewTicketSubscriptionService(mockRepo, mockTickets, mockAccess)
	return mockRepo, mockTickets, mockAccess, svc
}

func TestSubscriptionService_Subscribe_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()
	ticket := &models.Ticket{ID: ticketID, RealmID: &realmID}

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(ticket, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, ticket, userID, string(access.Read), realmID.String()).Return(nil)
	mockAccess.On("CanManage", mock.Anything, userID, ticket).Return(true, nil)
	mockRepo.On("Subscribe", mock.Anything, nil, ticketID, userID).Return(nil)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_Subscribe_Denied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSubscriptionService_Subscribe_NotManagerDenied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()
	ticket := &models.Ticket{ID: ticketID, RealmID: &realmID}

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(ticket, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, ticket, userID, string(access.Read), realmID.String()).Return(nil)
	mockAccess.On("CanManage", mock.Anything, userID, ticket).Return(false, nil)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSubscriptionService_Unsubscribe_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	ticket := &models.Ticket{ID: ticketID}

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(ticket, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, ticket, userID, string(access.Read), "").Return(nil)
	mockAccess.On("CanManage", mock.Anything, userID, ticket).Return(true, nil)
	mockRepo.On("Unsubscribe", mock.Anything, nil, ticketID, userID).Return(nil)

	err := svc.Unsubscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_IsSubscribed_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	ticket := &models.Ticket{ID: ticketID}

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(ticket, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, ticket, userID, string(access.Read), "").Return(nil)
	mockAccess.On("CanManage", mock.Anything, userID, ticket).Return(true, nil)
	mockRepo.On("Exists", mock.Anything, ticketID, userID).Return(true, nil)

	ok, err := svc.IsSubscribed(context.Background(), &models.IsSubscribedDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestNotificationService_TicketCommented_NotSelf(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		txManager:     &mockTransactionManager{},
	}

	ticketID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Assignee: &models.UserShort{ID: assigneeID},
	}

	mockRepo.On("GetSettings", mock.Anything, assigneeID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":true}`)}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.TicketCommented(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationService_TicketCommented_SelfIsAssignee(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		txManager:     &mockTransactionManager{},
	}

	ticketID := uuid.New()
	actorID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Assignee: &models.UserShort{ID: actorID},
	}

	err := svc.TicketCommented(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_AttachmentAdded_NotifiesAssignee(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		txManager:     &mockTransactionManager{},
	}

	ticketID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Assignee: &models.UserShort{ID: assigneeID},
	}

	mockRepo.On("GetSettings", mock.Anything, assigneeID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":true}`)}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.AttachmentAdded(context.Background(), ticket, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
