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

func ticketServiceFixtures() (*MockTicketsRepo, *MockActivityLogService, *MockSubtaskService, *MockAttachmentService, *MockNotificationService, *MockGroupsRepo, *MockAccessPolices, *TicketService) {
	mockRepo := new(MockTicketsRepo)
	mockLogs := new(MockActivityLogService)
	mockSubtasks := new(MockSubtaskService)
	mockAttachments := new(MockAttachmentService)
	mockNotifications := new(MockNotificationService)
	mockGroups := new(MockGroupsRepo)
	mockPolicies := new(MockAccessPolices)

	svc := NewTicketService(mockRepo, &mockTransactionManager{}, mockLogs, mockSubtasks, mockAttachments, mockNotifications, mockGroups, mockPolicies)
	return mockRepo, mockLogs, mockSubtasks, mockAttachments, mockNotifications, mockGroups, mockPolicies, svc
}

func TestTicketService_Get_Elevated(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()
	svc.policies = mockPolicies

	actorID := uuid.New()
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)

	expected := []*models.Ticket{
		{ID: uuid.New(), Title: "Ticket 1"},
	}
	mockRepo.On("Get", mock.Anything, req).Return(expected, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockPolicies.AssertExpectations(t)
}

func TestTicketService_Get_GroupFilter(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID).Return([]uuid.UUID{groupID}, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID).Return([]uuid.UUID{}, nil)

	expected := []*models.Ticket{
		{ID: uuid.New(), Title: "Ticket 1"},
	}
	expectedFilter := &models.TicketFilter{
		Actor:                     &models.Actor{ID: actorID, Name: "test"},
		Limit:                     20,
		Offset:                    0,
		GroupIDs:                  []uuid.UUID{groupID},
		IncludeUngroupedAssignedTo: &actorID,
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return(expected, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	assert.Equal(t, []uuid.UUID{groupID}, req.GroupIDs)
}

func TestTicketService_Get_NoGroups_ReturnsEmpty(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()
	svc.groups = mockGroups
	svc.policies = mockPolicies

	actorID := uuid.New()
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID).Return([]uuid.UUID{}, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID).Return([]uuid.UUID{}, nil)

	expectedFilter := &models.TicketFilter{
		Actor:                     &models.Actor{ID: actorID, Name: "test"},
		Limit:                     20,
		Offset:                    0,
		IncludeUngroupedAssignedTo: &actorID,
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return([]*models.Ticket{}, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, 0, total)
}

func TestTicketService_GetByID_Success(t *testing.T) {
	mockRepo, _, mockSubtasks, mockAttachments, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	req := &models.GetTicketByIdDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket"}
	mockRepo.On("GetByID", mock.Anything, req).Return(ticket, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)
	mockSubtasks.On("GetByTicketID", mock.Anything, ticketID, actorID).Return([]*models.Subtask{}, nil)
	mockAttachments.On("GetByEntity", mock.Anything, string(access.ResourceTicket), ticketID, actorID).Return([]*models.Attachment{}, nil)

	got, err := svc.GetByID(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, ticket, got)
}

func TestTicketService_Create_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:        &id,
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Title:     "New Ticket",
		GroupID:   &groupID,
		CreatorID: actorID,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID}).Return(&models.Group{
		ID:                groupID,
		DefaultAssigneeID: &assigneeID,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketCreated", mock.Anything, dto).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Create_MissingGroup(t *testing.T) {
	_, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()
	svc.policies = mockPolicies

	actorID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:    &id,
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Title: "New Ticket",
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)

	err := svc.Create(context.Background(), dto)
	assert.Error(t, err)
}

func TestTicketService_Update_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:    &ticketID,
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Title: "Updated Ticket",
	}

	oldTicket := &models.Ticket{
		ID:    ticketID,
		Title: "Original Ticket",
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Delete_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.DeleteTicketDTO{
		ID:    ticketID,
		Actor: &models.Actor{ID: actorID, Name: "test"},
	}

	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket"}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketDeleted", mock.Anything, ticket).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_CheckAccess_PolicyGranted(t *testing.T) {
	_, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)

	err := svc.CheckAccess(context.Background(), uuid.New(), actorID, string(access.Read))
	assert.NoError(t, err)
}

func TestTicketService_CheckAccess_GroupMember(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:    ticketID,
		Group: &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(true, nil)

	err := svc.CheckAccess(context.Background(), ticketID, actorID, string(access.Read))
	assert.NoError(t, err)
}

func TestTicketService_CheckAccess_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:    ticketID,
		Group: &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID).Return([]uuid.UUID{}, nil)

	err := svc.CheckAccess(context.Background(), ticketID, actorID, string(access.Read))
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}
