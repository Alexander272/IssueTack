package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func userRealmFixtures() (*MockUserRealmsRepo, *UserRealmService) {
	mockRepo := new(MockUserRealmsRepo)
	svc := &UserRealmService{
		repo:      mockRepo,
		txManager: &mockTransactionManager{},
	}
	return mockRepo, svc
}

func TestUserRealmService_GetAll(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	expected := []*models.UserRealm{{UserID: uuid.New(), RealmID: uuid.New()}}
	mockRepo.On("GetAll", mock.Anything).Return(expected, nil)

	got, err := svc.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUserRealmService_GetByUserID(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	userID := uuid.New()
	expected := []*models.UserRealm{{UserID: userID, RealmID: uuid.New()}}
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expected, nil)

	got, err := svc.GetByUserID(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUserRealmService_Create_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dto := &models.UserRealmDTO{UserID: uuid.New(), RealmID: uuid.New()}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestUserRealmService_Create_NoTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dto := &models.UserRealmDTO{UserID: uuid.New(), RealmID: uuid.New()}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestUserRealmService_CreateSeveral_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dtos := []*models.UserRealmDTO{{UserID: uuid.New(), RealmID: uuid.New()}}
	mockRepo.On("CreateSeveral", mock.Anything, nil, dtos).Return(nil)

	err := svc.CreateSeveral(context.Background(), nil, dtos)
	assert.NoError(t, err)
}

func TestUserRealmService_CreateSeveral_NoTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dtos := []*models.UserRealmDTO{{UserID: uuid.New(), RealmID: uuid.New()}}
	mockRepo.On("CreateSeveral", mock.Anything, nil, dtos).Return(nil)

	err := svc.CreateSeveral(context.Background(), nil, dtos)
	assert.NoError(t, err)
}

func TestUserRealmService_Update_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dto := &models.UserRealmDTO{UserID: uuid.New(), RealmID: uuid.New()}
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)

	err := svc.Update(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestUserRealmService_UpdateSeveral_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dtos := []*models.UserRealmDTO{{UserID: uuid.New(), RealmID: uuid.New()}}
	mockRepo.On("UpdateSeveral", mock.Anything, nil, dtos).Return(nil)

	err := svc.UpdateSeveral(context.Background(), nil, dtos)
	assert.NoError(t, err)
}

func TestUserRealmService_Delete_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	mockRepo.On("DeleteByUserAndRealm", mock.Anything, nil, userID, realmID).Return(nil)

	err := svc.Delete(context.Background(), nil, userID, realmID)
	assert.NoError(t, err)
}

func TestUserRealmService_DeleteSeveral_WithTx(t *testing.T) {
	mockRepo, svc := userRealmFixtures()

	dtos := []*models.UserRealmDTO{{UserID: uuid.New(), RealmID: uuid.New()}}
	mockRepo.On("DeleteSeveral", mock.Anything, nil, dtos).Return(nil)

	err := svc.DeleteSeveral(context.Background(), nil, dtos)
	assert.NoError(t, err)
}
