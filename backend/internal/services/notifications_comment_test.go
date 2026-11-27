package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNotificationService_TicketCommented_NotSelf(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)
	mockNotifier := new(MockNotifier)

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockNotifier.On("Name").Return("mock").Maybe()

	svc := &NotificationService{
		repo:          mockRepo,
		subscriptions: mockSubs,
		userRealms:    mockUserRealms,
		txManager:     &mockTransactionManager{},
		channels:      []Notifier{mockNotifier},
	}

	ticketID := uuid.New()
	actorID := uuid.New()
	assigneeID := uuid.New()
	ticket := &models.Ticket{
		ID:       ticketID,
		Title:    "Test",
		Assignee: &models.UserShort{ID: assigneeID},
	}

	mockNotifier.On("Notify", mock.Anything, assigneeID, mock.Anything, mock.Anything).Return(true, nil).Once()
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil).Once()

	err := svc.TicketCommented(context.Background(), &models.NotificationCommentDTO{
		Ticket:      ticket,
		ActorID:     actorID,
		CommentText: "уточните сроки",
		AuthorLabel: "Петров Иван",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockNotifier.AssertExpectations(t)
}

func TestNotificationService_TicketCommented_SelfIsAssignee(t *testing.T) {
	mockRepo := new(MockNotificationsRepo)
	mockSubs := new(MockTicketSubscriptionOps)
	mockUserRealms := new(MockUserRealmsService)

	mockUserRealms.On("GetRealmSupervisors", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()
	mockSubs.On("GetByTicket", mock.Anything, mock.Anything).Return([]uuid.UUID{}, nil).Maybe()

	svc := &NotificationService{
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

	err := svc.TicketCommented(context.Background(), &models.NotificationCommentDTO{
		Ticket:      ticket,
		ActorID:     actorID,
		CommentText: "уточните сроки",
		AuthorLabel: "Петров Иван",
	})
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}
