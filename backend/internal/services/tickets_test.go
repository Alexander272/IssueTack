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
		Access:        NewTicketAccessService(mockRepo, mockGroups, mockPolicies),
	})
	return mockRepo, mockLogs, mockSubtasks, mockAttachments, mockNotifications, mockGroups, mockPolicies, svc
}

func TestTicketService_Get_Elevated(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, _, mockPolicies, svc := ticketServiceFixtures()

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
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockPolicies.AssertExpectations(t)
}

func TestTicketService_Get_GroupFilter(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

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
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	assert.Equal(t, []uuid.UUID{groupID}, req.GroupIDs)
}

func TestTicketService_Get_NoGroups_ReturnsEmpty(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()
	svc.groups = mockGroups

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
	mockRepo, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

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
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Get_Assigned_Supervisor(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

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
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil))
}

func TestTicketService_Get_Created_Regular(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

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
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

	got, total, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	assert.Equal(t, 0, total)
}

func TestTicketService_Get_CreatedOrOwned_Regular(t *testing.T) {
	mockRepo, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	mode := "created_or_owned"
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
		Actor:            &models.Actor{ID: actorID, Name: "test"},
		Mode:             &mode,
		Limit:            20,
		Offset:           0,
		CreatedOrOwnedBy: &actorID,
	}
	mockRepo.On("Get", mock.Anything, expectedFilter).Return(expected, 0, nil)
	mockSubtasks.On("GetByTicketIDs", mock.Anything, mock.Anything).Return((map[uuid.UUID][]*models.Subtask)(nil), nil)

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

	// Создатель заявки читает собственную заявку: coarse-гейт Casbin даёт роль
	// user, а решение принимает атрибутная модель (создатель).
	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket", Creator: models.UserShort{ID: actorID, Username: "test"}}
	mockRepo.On("GetByID", mock.Anything, req).Return(ticket, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)
	mockSubtasks.On("GetByTicketID", mock.Anything, ticketID, actorID).Return([]*models.Subtask{}, nil)
	mockAttachments.On("GetByEntity", mock.Anything, mock.AnythingOfType("*models.EntityAccessDTO")).Return([]*models.Attachment{}, nil)

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)

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
	realmID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	managerID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:        &id,
		RealmID:   &realmID,
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Title:     "New Ticket",
		GroupID:   &groupID,
		CreatorID: actorID,
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID, RealmID: &realmID}).Return(&models.Group{
		ID:                groupID,
		DefaultAssigneeID: &assigneeID,
		ManagerID:         &managerID,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: id}).Return(&models.Ticket{ID: id}, nil)
	mockNotifications.On("TicketCreated", mock.Anything, mock.AnythingOfType("*models.Ticket"), mock.AnythingOfType("uuid.UUID")).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, &assigneeID, dto.AssigneeID)
	assert.Equal(t, &managerID, dto.ManagerID)
}

func TestTicketService_Create_WithSubtasks_Success(t *testing.T) {
	mockRepo, mockLogs, mockSubtasks, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	managerID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:        &id,
		RealmID:   &realmID,
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Title:     "New Ticket With Subtasks",
		GroupID:   &groupID,
		CreatorID: actorID,
		Subtasks: []*models.SubtaskDTO{
			{Title: "Subtask 1", Description: "first"},
			{Title: "Subtask 2"},
		},
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID, RealmID: &realmID}).Return(&models.Group{
		ID:                groupID,
		DefaultAssigneeID: &assigneeID,
		ManagerID:         &managerID,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockSubtasks.On("CreateManyOnCreate", mock.Anything, mock.Anything, mock.AnythingOfType("[]*models.SubtaskDTO")).Return(nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: id}).Return(&models.Ticket{ID: id}, nil)
	mockNotifications.On("TicketCreated", mock.Anything, mock.AnythingOfType("*models.Ticket"), mock.AnythingOfType("uuid.UUID")).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, id, dto.Subtasks[0].TicketID)
	assert.Equal(t, id, dto.Subtasks[1].TicketID)
	assert.Equal(t, dto.Actor, dto.Subtasks[0].Actor)
	assert.Equal(t, dto.Actor, dto.Subtasks[1].Actor)
	mockSubtasks.AssertCalled(t, "CreateManyOnCreate", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Create_DueDate_NonManagerDenied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	id := uuid.New()
	dueDate := time.Now().Add(48 * time.Hour)
	dto := &models.TicketDTO{
		ID:         &id,
		RealmID:    &realmID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		Title:      "New Ticket",
		GroupID:    &groupID,
		CreatorID:  actorID,
		AssigneeID: &assigneeID,
		DueDate:    &dueDate,
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID, RealmID: &realmID}).Return(&models.Group{ID: groupID}, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)

	err := svc.Create(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Create_DueDate_ManagerAllowed(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	id := uuid.New()
	dueDate := time.Now().Add(48 * time.Hour)
	dto := &models.TicketDTO{
		ID:         &id,
		RealmID:    &realmID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		Title:      "New Ticket",
		GroupID:    &groupID,
		CreatorID:  actorID,
		AssigneeID: &assigneeID,
		DueDate:    &dueDate,
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID, RealmID: &realmID}).Return(&models.Group{ID: groupID}, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: id}).Return(&models.Ticket{ID: id}, nil)
	mockNotifications.On("TicketCreated", mock.Anything, mock.AnythingOfType("*models.Ticket"), mock.AnythingOfType("uuid.UUID")).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, &assigneeID, dto.AssigneeID)
}

func TestTicketService_Create_MissingGroup(t *testing.T) {
	_, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

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
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: groupID, RealmID: &realmID}).Return(&models.Group{
		ID: groupID,
	}, nil)
	mockCategories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: realmID}).Return(&models.Category{
		ID:       categoryID,
		GroupID:  groupID,
		Priority: models.PriorityHigh,
	}, nil)
	mockRepo.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: id}).Return(&models.Ticket{ID: id}, nil)
	mockNotifications.On("TicketCreated", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, groupID, *dto.GroupID)
	assert.Equal(t, actorID, *dto.AssigneeID)
	assert.Equal(t, models.PriorityHigh, dto.Priority)
	assert.Nil(t, dto.DueDate)
}

// Группа — часть реалма тикета: попытка привязать её к группе чужого реалма
// (по id из тела) давала бы её участникам read-доступ к заявке, менеджеру —
// write/delete, а ответственному по умолчанию — назначение исполнителем.
func TestTicketService_Create_ForeignRealmGroup_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	foreignGroupID := uuid.New()
	id := uuid.New()
	dto := &models.TicketDTO{
		ID:        &id,
		RealmID:   &realmID,
		Actor:     &models.Actor{ID: actorID, Name: "test"},
		Title:     "New Ticket",
		GroupID:   &foreignGroupID,
		CreatorID: actorID,
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	// Группа чужого реалма под предиктом realm найдена не будет.
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: foreignGroupID, RealmID: &realmID}).
		Return(nil, models.ErrNotFound)

	err := svc.Create(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrNotFound)
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
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
	// Создатель правит собственный заголовок: права «поверх» заявки не нужны,
	// поэтому supervisor-роли у него нет (IsRealmSupervisor → category/site write).
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Update_ManagerIdEcho_Allowed(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	managerID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:          &ticketID,
		Actor:       &models.Actor{ID: actorID, Name: "test"},
		Description: "Updated description",
		ManagerID:   &managerID,
		Provided:    map[string]bool{"description": true, "managerId": true},
	}

	oldTicket := &models.Ticket{
		ID:          ticketID,
		Title:       "Original Ticket",
		Description: "Original description",
		Status:      models.StatusOpen,
		Creator:     models.UserShort{ID: actorID},
		Group:       &models.GroupShort{ID: groupID, Name: "Test Group"},
		Manager:     &models.UserShort{ID: managerID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)
	mockGroups.AssertNotCalled(t, "GetManagedGroups")

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLogs.AssertExpectations(t)
}

func TestTicketService_Update_DueDate_ManagerAllowed(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dueDate := time.Now().Add(48 * time.Hour)
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		DueDate:  &dueDate,
		Provided: map[string]bool{"dueDate": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestTicketService_Update_DueDate_ExecutorDenied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	ticketID := uuid.New()
	dueDate := time.Now().Add(48 * time.Hour)
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		DueDate:  &dueDate,
		Provided: map[string]bool{"dueDate": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: actorID},
		Group:    &models.GroupShort{ID: groupID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Update_Status_WriteNoRole_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
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
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketService_Update_Status_ManagerAllowed(t *testing.T) {
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

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Status:   models.StatusOpen,
		Creator:  models.UserShort{ID: uuid.New()},
		Assignee: &models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, ticketID).Return(0, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.Nil(t, dto.ResolvedAt)
	assert.Nil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_Cancel_FromOpen_Success(t *testing.T) {
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
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	assert.NotNil(t, dto.ClosedAt)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_Update_Owner_Cancel_FromInProgress_Denied(t *testing.T) {
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
		Status:  models.StatusInProgress,
		Creator: models.UserShort{ID: uuid.New()},
		Owner:   &models.UserShort{ID: actorID},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Group_NonAdmin_Denied(t *testing.T) {
	mockRepo, _, _, _, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	newGroupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		GroupID:  &newGroupID,
		Provided: map[string]bool{"groupId": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		Title:   "Original Ticket",
		Creator: models.UserShort{ID: actorID},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Group_Admin_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	newGroupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		GroupID:  &newGroupID,
		Provided: map[string]bool{"groupId": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		RealmID: &realmID,
		Title:   "Original Ticket",
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	// «Администратором» заявки теперь выступает начальник области, а не обладатель
	// ticket:write: право переносить заявку в другую группу больше не выдаётся
	// рядовым пользователям вместе с правом работать с заявками.
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	// Новая группа обязана принадлежать реалму тикета — иначе её участники
	// получили бы доступ к заявке (см. TestTicketService_Update_Group_ForeignRealm).
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: newGroupID, RealmID: &realmID}).
		Return(&models.Group{ID: newGroupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLogs.AssertExpectations(t)
}

// Смена группы на группу чужого реалма запрещена: проверка идёт по realm тикета,
// а право смены принадлежит администратору реалма. Чужой id отвечает ErrNotFound.
func TestTicketService_Update_Group_ForeignRealm_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	foreignGroupID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:       &ticketID,
		Actor:    &models.Actor{ID: actorID, Name: "test"},
		GroupID:  &foreignGroupID,
		Provided: map[string]bool{"groupId": true},
	}

	oldTicket := &models.Ticket{
		ID:      ticketID,
		RealmID: &realmID,
		Title:   "Original Ticket",
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}

	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockGroups.On("GetByID", mock.Anything, &models.GetGroupDTO{ID: foreignGroupID, RealmID: &realmID}).
		Return(nil, models.ErrNotFound)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrNotFound)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Assignee_NonAdmin_NonManager_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	newAssigneeID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
		Provided:   map[string]bool{"assigneeId": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Creator:  models.UserShort{ID: actorID},
		Group:    &models.GroupShort{ID: groupID, Name: "Test Group"},
		Assignee: &models.UserShort{ID: uuid.New()},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)

	err := svc.Update(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestTicketService_Update_Assignee_GroupManager_Success(t *testing.T) {
	mockRepo, mockLogs, _, _, mockNotifications, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	groupID := uuid.New()
	newAssigneeID := uuid.New()
	ticketID := uuid.New()
	dto := &models.TicketDTO{
		ID:         &ticketID,
		Actor:      &models.Actor{ID: actorID, Name: "test"},
		AssigneeID: &newAssigneeID,
		Provided:   map[string]bool{"assigneeId": true},
	}

	oldTicket := &models.Ticket{
		ID:       ticketID,
		Title:    "Original Ticket",
		Creator:  models.UserShort{ID: uuid.New()},
		Group:    &models.GroupShort{ID: groupID, Name: "Test Group"},
		Assignee: &models.UserShort{ID: uuid.New()},
	}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockLogs.AssertExpectations(t)
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
	mockRepo, mockLogs, mockSubtasks, mockAttachments, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.DeleteTicketDTO{
		ID:    ticketID,
		Actor: &models.Actor{ID: actorID, Name: "test"},
	}

	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket"}

	// Удаление по атрибутной модели доступно менеджеру группы; здесь им является
	// начальник области (обход через IsRealmSupervisor). Одного realm-wide
	// ticket:delete, без связи с тикетом, теперь недостаточно.
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockAttachments.On("DeleteByEntity", mock.Anything, nil, "ticket", ticketID).Return(nil)
	mockSubtasks.On("GetByTicketID", mock.Anything, ticketID, actorID).Return([]*models.Subtask{}, nil)
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketDeleted", mock.Anything, ticket).Return(nil)
	// Файлы вложений снимаются с диска только после коммита транзакции.
	mockAttachments.On("RemoveEntityDir", "ticket", ticketID).Return()

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
	mockAttachments.AssertExpectations(t)
}

// TestTicketService_Delete_RemovesAttachmentDirsAfterCommit — директории вложений
// тикета и его подзадач очищаются после коммита, а не внутри транзакции: файловая
// система в транзакцию не входит, и удаление до коммита при откате потеряло бы файлы
// безвозвратно.
func TestTicketService_Delete_RemovesAttachmentDirsAfterCommit(t *testing.T) {
	mockRepo, mockLogs, mockSubtasks, mockAttachments, mockNotifications, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	subID := uuid.New()
	dto := &models.DeleteTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket"}
	sub := &models.Subtask{ID: subID}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockAttachments.On("DeleteByEntity", mock.Anything, nil, "ticket", ticketID).Return(nil)
	mockAttachments.On("DeleteByEntity", mock.Anything, nil, "subtask", subID).Return(nil)
	mockSubtasks.On("GetByTicketID", mock.Anything, ticketID, actorID).Return([]*models.Subtask{sub}, nil)
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketDeleted", mock.Anything, ticket).Return(nil)
	mockAttachments.On("RemoveEntityDir", "ticket", ticketID).Return()
	mockAttachments.On("RemoveEntityDir", "subtask", subID).Return()

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
	mockAttachments.AssertExpectations(t)
}

// TestTicketService_Delete_RollbackKeepsFiles — регрессия на потерю файлов: если
// транзакция откатилась (здесь — ошибка удаления строки тикета уже после очистки
// записей вложений), директории с диска сниматься не должны. Раньше DeleteByEntity
// удалял их внутри транзакции, и файлы пропадали безвозвратно при живых записях БД.
func TestTicketService_Delete_RollbackKeepsFiles(t *testing.T) {
	mockRepo, mockLogs, mockSubtasks, mockAttachments, _, _, mockPolicies, svc := ticketServiceFixtures()

	actorID := uuid.New()
	ticketID := uuid.New()
	dto := &models.DeleteTicketDTO{ID: ticketID, Actor: &models.Actor{ID: actorID, Name: "test"}}

	ticket := &models.Ticket{ID: ticketID, Title: "Test Ticket"}

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Delete)).Return(true, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
	mockAttachments.On("DeleteByEntity", mock.Anything, nil, "ticket", ticketID).Return(nil)
	mockSubtasks.On("GetByTicketID", mock.Anything, ticketID, actorID).Return([]*models.Subtask{}, nil)
	// Откат: удаление строки тикета падает после очистки записей вложений.
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(assert.AnError)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.Error(t, err)

	// Файлы обязаны остаться на месте.
	mockAttachments.AssertNotCalled(t, "RemoveEntityDir", mock.Anything, mock.Anything)
}

// TestTicketService_CheckAccess_PolicyGranted: realm-wide ticket:read сам по себе
// больше не открывает заявку — решение принимает атрибутная модель. Обход есть
// только у начальника области (supervisor), поэтому доступ получает именно он.
func TestTicketService_CheckAccess_PolicyGranted(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, _ := ticketServiceFixtures()
	accessSvc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:    ticketID,
		Group: &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	// Ни в группе, ни её менеджером акт не связан, но у него права начальника области.
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read)})
	assert.NoError(t, err)
}

// TestTicketService_CheckAccess_PolicyGrantedWithoutAttributes_Denied фиксирует
// исходную проблему: заявитель с realm-wide ticket:read не должен видеть чужую
// заявку, в которой он не создатель, не исполнитель, не участник и не менеджер.
func TestTicketService_CheckAccess_PolicyGrantedWithoutAttributes_Denied(t *testing.T) {
	mockRepo, _, _, _, _, mockGroups, mockPolicies, _ := ticketServiceFixtures()
	accessSvc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)

	actorID := uuid.New()
	ticketID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceTicket), string(access.Read)).Return(true, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(&models.Ticket{
		ID:      ticketID,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Test Group"},
	}, nil)
	mockGroups.On("IsMember", mock.Anything, groupID, actorID).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)

	err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read)})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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

// TestTicketService_CheckAccess_AttributeMatrix закрепляет атрибутную модель
// целиком: кто что видит и что правит при наличии/отсутствии группы тикета и при
// наличии/отсутствии realm-wide прав. Ключевая проверка — заявитель с общим
// ticket:read/write не выходит за пределы собственных заявок, тогда как
// участник группы сохраняет доступ даже без coarse-прав, а начальник области
// (category:write) видит всё — в том числе без realm-wide ticket:read/write:
// coarse-права на тикет в модели доступа не участвуют.
func TestTicketService_CheckAccess_AttributeMatrix(t *testing.T) {
	type attrs struct {
		creator, assignee, member, manager, supervisor, coarse bool
		noGroup                                                bool
	}
	// coarse в атрибутах не участвует — CheckAccessOnTicket больше не смотрит в
	// Casbin за ticket:read/write. Поле осталось, чтобы кейсы прямо называли
	// ситуацию «realm-wide права есть» и тем самым закрепляли, что они ничего
	// не меняют.
	cases := []struct {
		name   string
		action string
		attrs  attrs
		allow  bool
	}{
		// Тикет с группой.
		{"grouped/creator/read", string(access.Read), attrs{creator: true}, true},
		{"grouped/creator/write", string(access.Write), attrs{creator: true}, true},
		{"grouped/creator/delete", string(access.Delete), attrs{creator: true}, false},
		{"grouped/assignee/read", string(access.Read), attrs{assignee: true}, true},
		{"grouped/assignee/write", string(access.Write), attrs{assignee: true}, false},
		{"grouped/member/read", string(access.Read), attrs{member: true}, true},
		{"grouped/member/write", string(access.Write), attrs{member: true}, false},
		{"grouped/manager/read", string(access.Read), attrs{manager: true}, true},
		{"grouped/manager/write", string(access.Write), attrs{manager: true}, true},
		{"grouped/manager/delete", string(access.Delete), attrs{manager: true}, true},
		{"grouped/stranger/read", string(access.Read), attrs{}, false},
		{"grouped/stranger/write", string(access.Write), attrs{}, false},
		{"grouped/stranger/delete", string(access.Delete), attrs{}, false},
		// Заявитель с realm-wide правами не выходит за пределы своей заявки.
		{"grouped/requester_coarse/own_read", string(access.Read), attrs{creator: true, coarse: true}, true},
		{"grouped/requester_coarse/own_write", string(access.Write), attrs{creator: true, coarse: true}, true},
		{"grouped/requester_coarse/foreign_read", string(access.Read), attrs{coarse: true}, false},
		{"grouped/requester_coarse/foreign_write", string(access.Write), attrs{coarse: true}, false},
		{"grouped/requester_coarse/foreign_delete", string(access.Delete), attrs{coarse: true}, false},
		// Начальник области видит всё независимо от coarse-прав.
		{"grouped/supervisor/foreign_read", string(access.Read), attrs{supervisor: true, coarse: true}, true},
		{"grouped/supervisor/foreign_write", string(access.Write), attrs{supervisor: true, coarse: true}, true},
		{"grouped/supervisor/foreign_delete", string(access.Delete), attrs{supervisor: true, coarse: true}, true},
		{"grouped/supervisor_no_coarse/foreign_read", string(access.Read), attrs{supervisor: true}, true},
		{"grouped/supervisor_no_coarse/foreign_write", string(access.Write), attrs{supervisor: true}, true},
		{"grouped/supervisor_no_coarse/foreign_delete", string(access.Delete), attrs{supervisor: true}, true},
		// Участник/менеджер без realm-wide прав не теряют доступ.
		{"grouped/member_no_coarse/read", string(access.Read), attrs{member: true}, true},
		{"grouped/manager_no_coarse/write", string(access.Write), attrs{manager: true}, true},
		// Тикет без группы: решение только по автору и исполнителю.
		{"ungrouped/creator/read", string(access.Read), attrs{creator: true, noGroup: true}, true},
		{"ungrouped/creator/write", string(access.Write), attrs{creator: true, noGroup: true}, true},
		{"ungrouped/creator/delete", string(access.Delete), attrs{creator: true, noGroup: true}, false},
		{"ungrouped/assignee/read", string(access.Read), attrs{assignee: true, noGroup: true}, true},
		{"ungrouped/assignee/write", string(access.Write), attrs{assignee: true, noGroup: true}, true},
		{"ungrouped/stranger/read", string(access.Read), attrs{noGroup: true}, false},
		{"ungrouped/stranger/write", string(access.Write), attrs{noGroup: true}, false},
		{"ungrouped/member_no_group/read", string(access.Read), attrs{member: true, manager: true, noGroup: true}, false},
		{"ungrouped/supervisor/foreign_read", string(access.Read), attrs{supervisor: true, coarse: true, noGroup: true}, true},
		{"ungrouped/supervisor_no_coarse/foreign_read", string(access.Read), attrs{supervisor: true, noGroup: true}, true},
		{"ungrouped/supervisor_no_coarse/foreign_write", string(access.Write), attrs{supervisor: true, noGroup: true}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo, _, _, _, _, mockGroups, mockPolicies, _ := ticketServiceFixtures()
			accessSvc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)

			actorID, ticketID, groupID := uuid.New(), uuid.New(), uuid.New()
			ticket := &models.Ticket{ID: ticketID, Title: "T", Creator: models.UserShort{ID: uuid.New()}}
			if tc.attrs.assignee {
				ticket.Assignee = &models.UserShort{ID: actorID}
			}
			if tc.attrs.creator {
				ticket.Creator.ID = actorID
			}
			if !tc.attrs.noGroup {
				ticket.Group = &models.GroupShort{ID: groupID, Name: "G"}
			}

			mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(ticket, nil)
			if !tc.attrs.noGroup {
				mockGroups.On("IsMember", mock.Anything, groupID, actorID).
					Return(tc.attrs.member && !tc.attrs.creator && !tc.attrs.assignee && !tc.attrs.manager, nil)
			}
			managed := []uuid.UUID{}
			if tc.attrs.manager && !tc.attrs.noGroup {
				managed = append(managed, groupID)
			}
			mockGroups.On("GetManagedGroups", mock.Anything, actorID, (*uuid.UUID)(nil)).Return(managed, nil)
			mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(tc.attrs.supervisor, nil)
			mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)

			err := accessSvc.CheckAccess(context.Background(), &models.AccessCheckDTO{
				TicketID: ticketID, UserID: actorID, Action: tc.action,
			})
			if tc.allow {
				assert.NoError(t, err, "доступ должен быть разрешён")
			} else {
				assert.ErrorIs(t, err, models.ErrPermissionDenied, "доступ должен быть запрещён")
			}
		})
	}
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
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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

	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	// Начальником области владелец заявки не является: тонкие права проверяются
	// всегда, даже когда правка идёт по пути ownerOnly.
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(oldTicket, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)
	mockNotifications.On("TicketUpdated", mock.Anything, mock.AnythingOfType("*models.Ticket"), actorID, mock.Anything).Return(nil)

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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", actorID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)
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
	mockPolicies.On("Enforce", creatorID.String(), mock.Anything, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", creatorID.String(), mock.Anything, string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, creatorID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, mock.Anything).Return(0, nil)

	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: creatorID},
		Group:   &models.GroupShort{ID: groupID, Name: "Group"},
	}

	flags, err := svc.GetAccessFlags(context.Background(), ticket, creatorID)
	assert.NoError(t, err)
	assert.True(t, flags.CanEditFields)
}

func TestTicketService_GetAccessFlags_IsAdmin(t *testing.T) {
	_, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	adminID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", adminID.String(), mock.Anything, string(access.ResourceTicket), string(access.Write)).Return(true, nil)
	mockPolicies.On("Enforce", adminID.String(), mock.Anything, string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	// IsAdmin = начальник области, а не обладатель ticket:write.
	mockPolicies.On("Enforce", adminID.String(), mock.Anything, string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, adminID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, mock.Anything).Return(0, nil)

	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Group"},
	}

	flags, err := svc.GetAccessFlags(context.Background(), ticket, adminID)
	assert.NoError(t, err)
	assert.True(t, flags.IsAdmin)
	assert.False(t, flags.IsManager)
}

func TestTicketService_GetAccessFlags_IsManager(t *testing.T) {
	_, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	managerID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", managerID.String(), mock.Anything, string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", managerID.String(), mock.Anything, string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockPolicies.On("Enforce", managerID.String(), mock.Anything, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", managerID.String(), mock.Anything, string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, managerID, (*uuid.UUID)(nil)).Return([]uuid.UUID{groupID}, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, mock.Anything).Return(0, nil)

	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Group"},
	}

	flags, err := svc.GetAccessFlags(context.Background(), ticket, managerID)
	assert.NoError(t, err)
	assert.False(t, flags.IsAdmin)
	assert.True(t, flags.IsManager)
}

func TestTicketService_GetAccessFlags_NoRoles(t *testing.T) {
	_, _, mockSubtasks, _, _, mockGroups, mockPolicies, svc := ticketServiceFixtures()

	userID := uuid.New()
	groupID := uuid.New()

	mockPolicies.On("Enforce", userID.String(), mock.Anything, string(access.ResourceTicket), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), mock.Anything, string(access.ResourceTicket), string(access.Delete)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), mock.Anything, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), mock.Anything, string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, userID, (*uuid.UUID)(nil)).Return([]uuid.UUID{}, nil)
	mockSubtasks.On("GetUnresolvedCount", mock.Anything, mock.Anything).Return(0, nil)

	ticket := &models.Ticket{
		ID:      uuid.New(),
		Title:   "Ticket",
		Status:  models.StatusOpen,
		Creator: models.UserShort{ID: uuid.New()},
		Group:   &models.GroupShort{ID: groupID, Name: "Group"},
	}

	flags, err := svc.GetAccessFlags(context.Background(), ticket, userID)
	assert.NoError(t, err)
	assert.False(t, flags.IsAdmin)
	assert.False(t, flags.IsManager)
}

func TestTicketService_GetSummary(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	ticketID := uuid.New()
	expected := &models.Ticket{
		ID:      ticketID,
		Title:   "Summary Ticket",
		RealmID: func() *uuid.UUID { r := uuid.New(); return &r }(),
	}
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(expected, nil)

	got, err := svc.GetSummary(context.Background(), ticketID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	mockRepo.AssertExpectations(t)
}

func TestTicketService_GetSummary_NotFound(t *testing.T) {
	mockRepo, _, _, _, _, _, _, svc := ticketServiceFixtures()

	ticketID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(nil, models.ErrNoRows)

	_, err := svc.GetSummary(context.Background(), ticketID)
	assert.ErrorIs(t, err, models.ErrNoRows)
	mockRepo.AssertExpectations(t)
}
