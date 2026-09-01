package services

import (
	"context"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ticketServiceFixtures() (*MockTicketsRepo, *MockActivityLogService, *MockSubtaskService, *MockAttachmentService, *MockNotificationService, *MockGroupsRepo, *MockAccessPolicies, *TicketService) {
	mockRepo := new(MockTicketsRepo)
	mockLogs := new(MockActivityLogService)
	mockSubtasks := new(MockSubtaskService)
	mockAttachments := new(MockAttachmentService)
	mockNotifications := new(MockNotificationService)
	mockGroups := new(MockGroupsRepo)
	mockPolicies := new(MockAccessPolicies)

	svc := NewTicketService(&TicketDeps{
		Repo:          mockRepo,
		TxManager:     &mockTransactionManager{},
		Logs:          mockLogs,
		Subtasks:      mockSubtasks,
		Attachments:   mockAttachments,
		Notifications: mockNotifications,
		Groups:        mockGroups,
		Categories:    new(MockCategoriesRepo),
		Policies:      mockPolicies,
		Access:        NewTicketAccessService(mockRepo, mockGroups, mockPolicies),
	})
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)

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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)

	expected := []*models.Ticket{
		{ID: uuid.New(), Title: "Ticket 1"},
	}
	expectedFilter := &models.TicketFilter{
		Actor:                      &models.Actor{ID: actorID, Name: "test"},
		Limit:                      20,
		Offset:                     0,
		GroupIDs:                   []uuid.UUID{groupID},
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)

	expectedFilter := &models.TicketFilter{
		Actor:                      &models.Actor{ID: actorID, Name: "test"},
		Limit:                      20,
		Offset:                     0,
		IncludeUngroupedAssignedTo: &actorID,
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return([]*models.Ticket{}, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, 0, total)
}

func TestTicketService_Get_Assigned_Regular(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	managedID := uuid.New()
	memberID := uuid.New()
	mode := "assigned"
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Mode:  &mode,
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{memberID}, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{managedID}, nil)

	expected := []*models.Ticket{{ID: uuid.New(), Title: "Ticket 1"}}
	expectedFilter := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Mode:  &mode,
		Limit: 20, Offset: 0,
		MyWork: &models.MyWorkFilter{UserID: actorID, GroupIDs: []uuid.UUID{managedID, memberID}},
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return(expected, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Get_Assigned_Supervisor(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	memberID := uuid.New()
	mode := "assigned"
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Mode:  &mode,
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{memberID}, nil)

	expected := []*models.Ticket{{ID: uuid.New(), Title: "Ticket 1"}}
	expectedFilter := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Mode:  &mode,
		Limit: 20, Offset: 0,
		MyWork: &models.MyWorkFilter{UserID: actorID, GroupIDs: []uuid.UUID{memberID}},
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return(expected, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil))
}

func TestTicketService_Get_Created_Regular(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	mode := "created"
	req := &models.TicketFilter{
		Actor: &models.Actor{ID: actorID, Name: "test"},
		Mode:  &mode,
		Limit: 20, Offset: 0,
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)

	expected := []*models.Ticket{{ID: uuid.New(), Title: "Ticket 1"}}
	expectedFilter := &models.TicketFilter{
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Mode:      &mode,
		Limit:     20,
		Offset:    0,
		CreatorID: &actorID,
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return(expected, 0, nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
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
	mockAttachments.On("GetByEntity", mock.Anything, mock.AnythingOfType("*models.EntityAccessDTO")).Return([]*models.Attachment{}, nil)

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)

	got, err := svc.GetByID(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, ticket, got)
	assert.NotNil(t, got.Access)
	assert.True(t, got.Access.CanRead)
	assert.True(t, got.Access.CanWrite)
	assert.False(t, got.Access.CanDelete)
	assert.True(t, got.Access.CanWork)
}

func TestTicketService_Create_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	managerID := uuid.New()
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
		ManagerID:         &managerID,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketCreated", mock.Anything, dto).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, &assigneeID, dto.AssigneeID)
	assert.Equal(t, &managerID, dto.ManagerID)
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

func TestTicketService_Create_Executor_OwnerRequired(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	dto := &models.TicketDTO{
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Title:     "New Ticket",
		RealmID:   &realmID,
		CreatorID: actorID,
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, &realmID).Return([]uuid.UUID{uuid.New()}, nil)

	err := svc.Create(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrOwnerRequired)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Create_Executor_OwnGroup(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	categoryID := uuid.New()
	ownerID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:         &id,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		Title:      "New Ticket",
		RealmID:    &realmID,
		CategoryID: categoryID,
		OwnerID:    &ownerID,
		CreatorID:  actorID,
	}

	mockCategories := new(MockCategoriesRepo)
	svc.categories = mockCategories

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockGroups.On("GetMemberGroups", mock.Anything, actorID, &realmID).Return([]uuid.UUID{groupID}, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID}).Return(&models.Group{
		ID: groupID,
	}, nil)
	mockCategories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: realmID}).Return(&models.Category{
		ID:       categoryID,
		GroupID:  groupID,
		Priority: models.PriorityHigh,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketCreated", mock.Anything, mock.Anything).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, groupID, *dto.GroupID)
	assert.Equal(t, actorID, *dto.AssigneeID)
	assert.Equal(t, models.PriorityHigh, dto.Priority)
	assert.Nil(t, dto.DueDate)
}

func TestTicketService_Update_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated Ticket",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Creator: models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Update_FieldEdit_WriteNoRole_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated Ticket",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Creator: models.UserShort{ID: uuid.New()},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_ManagerIdDenied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	managerID := uuid.New()
	dto := &models.TicketDTO{
		ID:        &ticketID,
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		ManagerID: &managerID,
		Provided:  map[string]bool{"managerId": true},
	}

	oldTicket := &models.Ticket{ID: ticketID, Title: "Original Ticket"}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Update_StatusOnly_Assignee(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusInProgress,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Priority: models.PriorityHigh,
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLogs.AssertExpectations(t)
	mockNotifications.AssertExpectations(t)
}

func TestTicketService_Update_Assignee_SetResolved_Success(t *testing.T) {
	mockRepo, mockLogs, mockSubtasks, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusResolved,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ResolvedAt)
	assert.Nil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Resolve_UnresolvedSubtasks_Denied(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusResolved,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(1, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrSubtasksNotResolved)
	assert.Nil(t, dto.ResolvedAt)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Resolve_CancelledSubtasks_Success(t *testing.T) {
	mockRepo, mockLogs, mockSubtasks, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusResolved,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Resolve_SubtaskCheckError(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusResolved,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, assert.AnError)

	err := svc.Update(context.Background(), dto)
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Assignee_SetClosed_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusClosed,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Creator_SetClosed_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusClosed,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusResolved,
		Creator: models.UserShort{ID: actorID},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Close_NotResolved_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusClosed,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: actorID},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrCloseRequiresResolved)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Manager_SetCancelled_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusCancelled,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_Accept_FromResolved_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusClosed,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusResolved,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_ReturnToWork_FromResolved_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusInProgress,
		Provided: map[string]bool{"status": true},
	}

	resolvedAt := time.Now()
	oldTicket := &models.Ticket{
		ID:         ticketID,
		Title:      "Original Ticket",
		Status:     models.StatusResolved,
		Creator:    models.UserShort{ID: uuid.New()},
		Owner:      &models.UserShort{ID: actorID},
		ResolvedAt: &resolvedAt,
		Group:      &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.Nil(t, dto.ResolvedAt)
	assert.Nil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_Cancel_FromActive_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusCancelled,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_Cancel_FromResolved_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusCancelled,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusResolved,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Owner_StatusChange_FromNonResolved_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusInProgress,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Owner_FieldChange_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated Ticket",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_PolicyWrite_NotManager_SetClosed_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Status:   models.StatusClosed,
		Provided: map[string]bool{"status": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_AutoCloseResolved_Disabled(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	n, err := svc.AutoCloseResolved(context.Background(), 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
	mockRepo.AssertNotCalled(t, "CloseResolved")
}

func TestTicketService_AutoCloseResolved_Success(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	mockRepo.On("CloseResolved", mock.Anything, mock.Anything).Return(int64(3), nil)

	n, err := svc.AutoCloseResolved(context.Background(), 24*time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), n)
	mockRepo.AssertExpectations(t)
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
	mockRepo, _, _, _, _, _, mockPolicies, _ := ticketServiceFixtures()
	accessSvc := NewTicketAccessService(mockRepo, nil, mockPolicies)

	actorID := uuid.New()
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)

	err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{TicketID: uuid.New(), UserID: actorID, Action: string(access.Read)})
	assert.NoError(t, err)
}

func TestTicketService_CheckAccess_GroupMember(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, _ := ticketServiceFixtures()
	accessSvc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:    ticketID,
		Group: &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(true, nil)

	err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read)})
	assert.NoError(t, err)
}

func TestTicketService_CheckAccess_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, _ := ticketServiceFixtures()
	accessSvc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:    ticketID,
		Group: &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)

	err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read)})
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Take_NoAssignee_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TakeTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		dtoUpdate := args.Get(2).(*models.TicketDTO)
		assert.Equal(t, models.StatusInProgress, dtoUpdate.Status)
		assert.Equal(t, actorID, *dtoUpdate.AssigneeID)
		assert.True(t, dtoUpdate.HasField("status"))
		assert.True(t, dtoUpdate.HasField("assigneeId"))
	})
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Take(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Take_OtherAssignee_Open_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	otherID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TakeTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: actorID},
		Assignee: &models.UserShort{ID: otherID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, mock.Anything).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Take(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Take_OtherAssignee_NotOpen_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	otherID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TakeTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: actorID},
		Assignee: &models.UserShort{ID: otherID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Take(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Take_AlreadyAssignee_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TakeTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: actorID},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Take(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Take_InactiveStatus_Frozen(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TakeTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:     ticketID,
		Title:  "Ticket",
		Status: models.StatusResolved,
	}, nil)

	err := svc.Take(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrTicketFrozen)
}

func TestTicketService_Transfer_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, _, svc := ticketServiceFixtures()

	actorID := uuid.New()
	newAssigneeID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TransferTicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID, Name: "Group"},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, newAssigneeID).Return(true, nil)
	mockRepo.On("Update", mock.Anything, nil, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		dtoUpdate := args.Get(2).(*models.TicketDTO)
		assert.Equal(t, &newAssigneeID, dtoUpdate.AssigneeID)
		assert.True(t, dtoUpdate.HasField("assigneeId"))
	})
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Transfer(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Transfer_NotAssignee_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	actorID := uuid.New()
	otherID := uuid.New()
	newAssigneeID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TransferTicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID, Name: "Group"},
		Assignee: &models.UserShort{ID: otherID},
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Transfer(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Transfer_NotGroupMember_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, _, svc := ticketServiceFixtures()

	actorID := uuid.New()
	newAssigneeID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TransferTicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Ticket",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID, Name: "Group"},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, newAssigneeID).Return(false, nil)

	err := svc.Transfer(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Transfer_Inactive_Frozen(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	actorID := uuid.New()
	newAssigneeID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TransferTicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:     ticketID,
		Title:  "Ticket",
		Status: models.StatusResolved,
	}, nil)

	err := svc.Transfer(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrTicketFrozen)
}

func TestTicketService_Update_Owner_EditInOpen_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:     ticketID,
		Title:  "Original",
		Status: models.StatusOpen,
		Owner:  &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, ticketID, actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Update_Owner_EditNotOpen_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original",
		Status:  models.StatusInProgress,
		Owner:   &models.UserShort{ID: actorID},
		Creator: models.UserShort{ID: uuid.New()},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_Update_Assignee_EditFields_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		Title:    "Updated",
		Provided: map[string]bool{"title": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original",
		Status:   models.StatusInProgress,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestTicketService_GetAccessFlags_CanEditFields(t *testing.T) {
	_, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	creatorID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", creatorID.String(), mock.Anything, string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", creatorID.String(), mock.Anything, string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockPolicies.On("Enforce", creatorID.String(), mock.Anything, string(access.ResourceTicket), string(access.Read)).Return(true, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, creatorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, mock.Anything).Return(0, nil)

	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: creatorID},
		Group:   &models.GroupShort{ID: groupID, Name: "Group"},
	}

	flags, err := svc.GetAccessFlags(context.Background(), ticket, creatorID, "")
	assert.NoError(t, err)
	assert.True(t, flags.CanEditFields)
}
