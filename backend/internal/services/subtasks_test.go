package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func subtaskServiceFixtures() (*MockSubtasksRepo, *MockActivityLogService, *MockTicketAccessChecker, *SubtaskService) {
	mockRepo := new(MockSubtasksRepo)
	mockLogs := new(MockActivityLogService)
	mockAccess := new(MockTicketAccessChecker)

	svc := &SubtaskService{
		repo:         mockRepo,
		logs:         mockLogs,
		ticketAccess: mockAccess,
	}
	return mockRepo, mockLogs, mockAccess, svc
}

func TestSubtaskService_GetByTicketID_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	expected := []*models.Subtask{
		{ID: uuid.New(), Title: "Subtask 1", TicketID: ticketID},
	}

	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "read").Return(nil)
	mockRepo.On("GetByTicketID", mock.Anything, ticketID).Return(expected, nil)

	got, err := svc.GetByTicketID(context.Background(), ticketID, actorID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	mockAccess.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestSubtaskService_GetByTicketID_NoAccess(t *testing.T) {
	mockRepo, _, _, svc := subtaskServiceFixtures()
	svc.ticketAccess = nil

	_, err := svc.GetByTicketID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "GetByTicketID")
}

func TestSubtaskService_GetByTicketID_AccessDenied(t *testing.T) {
	mockRepo, _, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	mockAccess.On("CheckAccess", mock.Anything, ticketID, mock.Anything, "read").Return(models.ErrPermissionDenied)

	_, err := svc.GetByTicketID(context.Background(), ticketID, uuid.New())
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "GetByTicketID")
}

func TestSubtaskService_GetByID_Success(t *testing.T) {
	mockRepo, _, mockAccess, svc := subtaskServiceFixtures()

	subtaskID := uuid.New()
	ticketID := uuid.New()
	actorID := uuid.New()
	req := &models.GetSubtaskDTO{ID: subtaskID}
	expected := &models.Subtask{ID: subtaskID, Title: "Subtask", TicketID: ticketID}

	mockRepo.On("GetByID", mock.Anything, req).Return(expected, nil)
	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "read").Return(nil)

	got, err := svc.GetByID(context.Background(), req, actorID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestSubtaskService_GetByID_NoAccess(t *testing.T) {
	mockRepo, _, _, svc := subtaskServiceFixtures()
	svc.ticketAccess = nil

	subtaskID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(&models.Subtask{}, nil)

	_, err := svc.GetByID(context.Background(), &models.GetSubtaskDTO{ID: subtaskID}, uuid.New())
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestSubtaskService_Create_Success(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	dto := &models.SubtaskDTO{
		ID:       uuid.New(),
		TicketID: ticketID,
		Title:    "New Subtask",
		Actor:    &models.Actor{ID: actorID, Name: "test"},
	}

	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "write").Return(nil)
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockLogs.AssertExpectations(t)
}

func TestSubtaskService_Create_NoAccess(t *testing.T) {
	_, _, _, svc := subtaskServiceFixtures()
	svc.ticketAccess = nil

	err := svc.Create(context.Background(), nil, &models.SubtaskDTO{})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestSubtaskService_CreateSeveral_Success(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	actorID := uuid.New()
	dtos := []*models.SubtaskDTO{
		{ID: uuid.New(), TicketID: ticketID, Title: "S1", Actor: &models.Actor{ID: actorID, Name: "test"}},
		{ID: uuid.New(), TicketID: ticketID, Title: "S2", Actor: &models.Actor{ID: actorID, Name: "test"}},
	}

	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "write").Return(nil)
	mockRepo.On("CreateSeveral", mock.Anything, nil, dtos).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.CreateSeveral(context.Background(), nil, dtos)
	assert.NoError(t, err)
}

func TestSubtaskService_CreateSeveral_NoAccess(t *testing.T) {
	_, _, _, svc := subtaskServiceFixtures()
	svc.ticketAccess = nil

	err := svc.CreateSeveral(context.Background(), nil, []*models.SubtaskDTO{{}})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestSubtaskService_CreateSeveral_EmptyList(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	mockRepo.On("CreateSeveral", mock.Anything, nil, []*models.SubtaskDTO{}).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.CreateSeveral(context.Background(), nil, []*models.SubtaskDTO{})
	assert.NoError(t, err)
	mockAccess.AssertNotCalled(t, "CheckAccess")
}

func TestSubtaskService_Update_Success(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	subtaskID := uuid.New()
	actorID := uuid.New()
	dto := &models.SubtaskDTO{
		ID:       subtaskID,
		TicketID: ticketID,
		Title:    "Updated",
		Status:   "done",
		Actor:    &models.Actor{ID: actorID, Name: "test"},
	}
	old := &models.Subtask{
		ID: subtaskID, TicketID: ticketID, Title: "Original", Status: "open",
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(old, nil)
	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "write").Return(nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Update(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestSubtaskService_Update_NoChanges(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	subtaskID := uuid.New()
	ticketID := uuid.New()
	actorID := uuid.New()
	dto := &models.SubtaskDTO{
		ID:       subtaskID,
		TicketID: ticketID,
		Title:    "Same",
		Status:   "open",
		Actor:    &models.Actor{ID: actorID, Name: "test"},
	}
	old := &models.Subtask{
		ID: subtaskID, TicketID: ticketID, Title: "Same", Status: "open",
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(old, nil)
	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "write").Return(nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)

	err := svc.Update(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockLogs.AssertNotCalled(t, "Create")
}

func TestSubtaskService_Delete_Success(t *testing.T) {
	mockRepo, mockLogs, mockAccess, svc := subtaskServiceFixtures()

	ticketID := uuid.New()
	subtaskID := uuid.New()
	actorID := uuid.New()
	dto := &models.DelSubtaskDTO{ID: subtaskID, Actor: &models.Actor{ID: actorID, Name: "test"}}
	old := &models.Subtask{
		ID: subtaskID, TicketID: ticketID, Title: "To Delete", Status: "open", Priority: models.PriorityMedium,
	}

	mockRepo.On("GetByID", mock.Anything, &models.GetSubtaskDTO{ID: subtaskID}).Return(old, nil)
	mockAccess.On("CheckAccess", mock.Anything, ticketID, actorID, "write").Return(nil)
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)
	mockLogs.On("Create", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}
