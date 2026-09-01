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

func subscriptionServiceFixtures() (*MockTicketSubscriptionsRepo, *MockTicketsRepo, *MockTicketAccessChecker, *MockAccessPolicies, *MockGroupsRepo, *TicketSubscriptionService) {
	mockRepo := new(MockTicketSubscriptionsRepo)
	mockTickets := new(MockTicketsRepo)
	mockAccess := new(MockTicketAccessChecker)
	mockPolicies := new(MockAccessPolicies)
	mockGroups := new(MockGroupsRepo)

	svc := NewTicketSubscriptionService(mockRepo, mockTickets, mockAccess, mockPolicies, mockGroups)
	return mockRepo, mockTickets, mockAccess, mockPolicies, mockGroups, svc
}

func TestSubscriptionService_Subscribe_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, mockPolicies, _, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: realmID.String(),
	}).Return(nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("Subscribe", mock.Anything, nil, ticketID, userID).Return(nil)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_Subscribe_Denied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, _, _, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSubscriptionService_Subscribe_NotSupervisorDenied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, mockPolicies, _, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: realmID.String(),
	}).Return(nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)

	err := svc.Subscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSubscriptionService_Unsubscribe_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, mockPolicies, _, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: "",
	}).Return(nil)
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("Unsubscribe", mock.Anything, nil, ticketID, userID).Return(nil)

	err := svc.Unsubscribe(context.Background(), &models.SubscribeDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSubscriptionService_IsSubscribed_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, mockPolicies, _, svc := subscriptionServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: "",
	}).Return(nil)
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("Exists", mock.Anything, ticketID, userID).Return(true, nil)

	ok, err := svc.IsSubscribed(context.Background(), &models.IsSubscribedDTO{TicketID: ticketID, ActorID: userID})
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestNotificationService_TicketCommented_NotSelf(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockTicketRepo := new(MockTicketsRepo)
	mockSubs := new(MockTicketSubscriptionsRepo)
	mockUserRealms := new(MockUserRealmsRepo)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		ticketRepo:    mockTicketRepo,
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

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("GetSettings", mock.Anything, assigneeID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":true}`)}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.TicketCommented(context.Background(), ticketID, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationService_TicketCommented_SelfIsAssignee(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockTicketRepo := new(MockTicketsRepo)
	mockSubs := new(MockTicketSubscriptionsRepo)
	mockUserRealms := new(MockUserRealmsRepo)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		ticketRepo:    mockTicketRepo,
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

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)

	err := svc.TicketCommented(context.Background(), ticketID, actorID)
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotificationService_AttachmentAdded_NotifiesAssignee(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockTicketRepo := new(MockTicketsRepo)
	mockSubs := new(MockTicketSubscriptionsRepo)
	mockUserRealms := new(MockUserRealmsRepo)
	hub := ws_hub.NewWebsocketHub()

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
		hub:           hub,
		repo:          mockRepo,
		ticketRepo:    mockTicketRepo,
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

	mockTicketRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("GetSettings", mock.Anything, assigneeID).Return(
		&models.NotificationSettings{Settings: []byte(`{"push":true}`)}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.AttachmentAdded(context.Background(), ticketID, actorID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
