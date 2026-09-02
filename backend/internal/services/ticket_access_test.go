package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func accessServiceFixtures() (*MockTicketsRepo, *MockGroupsRepo, *MockAccessPolicies, *TicketAccessService) {
	mockRepo := new(MockTicketsRepo)
	mockGroups := new(MockGroupsRepo)
	mockPolicies := new(MockAccessPolicies)
	svc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)
	return mockRepo, mockGroups, mockPolicies, svc
}

func TestTicketAccessService_IsRealmSupervisor_CategoryWrite(t *testing.T) {
	_, _, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, ok)
	mockPolicies.AssertNotCalled(t, "Enforce", mock.Anything, mock.Anything, string(access.ResourceSite), mock.Anything)
}

func TestTicketAccessService_IsRealmSupervisor_SiteWrite(t *testing.T) {
	_, _, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceSite), string(access.Write)).Return(true, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestTicketAccessService_IsRealmSupervisor_NoRights(t *testing.T) {
	_, _, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceSite), string(access.Write)).Return(false, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.False(t, ok)
	mockPolicies.AssertExpectations(t)
}

func TestTicketAccessService_CanManage_Supervisor(t *testing.T) {
	_, mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.True(t, allowed)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketAccessService_CanManage_GroupManager(t *testing.T) {
	_, mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{groupID}, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestTicketAccessService_CanManage_OtherGroupManager(t *testing.T) {
	_, mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{uuid.New()}, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestTicketAccessService_CanManage_GrouplessTicket(t *testing.T) {
	_, mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: nil,
		Group:   nil,
	}
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.False(t, allowed)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketAccessService_CanCreateTicket(t *testing.T) {
	_, _, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceTicket), string(access.Write)).Return(true, nil)

	allowed, err := svc.CanCreateTicket(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, allowed)
	mockPolicies.AssertExpectations(t)
}

func TestTicketAccessService_CanCreateTicket_Denied(t *testing.T) {
	_, _, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceTicket), string(access.Write)).Return(false, nil)

	allowed, err := svc.CanCreateTicket(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.False(t, allowed)
	mockPolicies.AssertExpectations(t)
}
