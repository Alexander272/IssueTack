package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func activityLogFixtures() (*MockActivityLogRepo, *ActivityLogService) {
	mockRepo := new(MockActivityLogRepo)
	svc := &ActivityLogService{
		repo:      mockRepo,
		txManager: &mockTransactionManager{},
	}
	return mockRepo, svc
}

func TestActivityLogService_Get(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	req := &models.GetLogsDTO{}
	expected := []*models.ActivityLog{{ID: uuid.New(), Action: "created"}}
	mockRepo.On("Get", mock.Anything, req).Return(expected, nil)

	got, err := svc.Get(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestActivityLogService_Create_WithTx(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: "ticket"},
	}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestActivityLogService_Create_NilTx(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: "ticket"},
	}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestActivityLogService_Create_EmptyDTO(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	err := svc.Create(context.Background(), nil, []*models.ActivityLogDTO{})
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "Create")
}
