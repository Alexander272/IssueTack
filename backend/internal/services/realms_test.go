package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func realmServiceFixtures() (*MockRealmsRepo, *RealmService) {
	mockRepo := new(MockRealmsRepo)
	svc := &RealmService{
		repo:      mockRepo,
		txManager: &mockTransactionManager{},
	}
	return mockRepo, svc
}

func TestRealmService_GetByID(t *testing.T) {
	mockRepo, svc := realmServiceFixtures()

	req := &models.GetRealmByIdDTO{ID: uuid.New()}
	expected := &models.Realm{ID: req.ID, Name: "Test Realm"}
	mockRepo.On("GetByID", mock.Anything, req).Return(expected, nil)

	got, err := svc.GetByID(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRealmService_Create(t *testing.T) {
	mockRepo, svc := realmServiceFixtures()

	dto := &models.RealmDTO{ID: uuid.New(), Name: "New Realm"}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
}

func TestRealmService_Update(t *testing.T) {
	mockRepo, svc := realmServiceFixtures()

	dto := &models.RealmDTO{ID: uuid.New(), Name: "Updated"}
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestRealmService_Delete(t *testing.T) {
	mockRepo, svc := realmServiceFixtures()

	dto := &models.DeleteRealmDTO{ID: uuid.New()}
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
}
