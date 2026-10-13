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

	realmID := uuid.New()
	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: models.EntityTypeTicket, RealmID: &realmID},
	}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	// Реалм уже задан — резолвер не трогаем.
	mockRepo.AssertNotCalled(t, "GetRealmByEntity", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestActivityLogService_Create_NilTx(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	realmID := uuid.New()
	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: models.EntityTypeTicket, RealmID: &realmID},
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

// Записи без RealmID дозаполняются реалмом сущности до вызова Create: без этого
// фильтр GET /activity-log по авторизованному реалму никогда не вернёт их.
func TestActivityLogService_Create_FillsRealm(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	realmID := uuid.New()
	entityID := uuid.New()
	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: models.EntityTypeTicket, EntityID: entityID},
	}
	mockRepo.On("GetRealmByEntity", mock.Anything, mock.Anything, models.EntityTypeTicket, entityID).
		Return(realmID, "IT", true, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(got []*models.ActivityLogDTO) bool {
		return len(got) == 1 &&
			got[0].RealmID != nil && *got[0].RealmID == realmID &&
			got[0].RealmName == "IT"
	})).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	assert.Equal(t, &realmID, dto[0].RealmID)
	assert.Equal(t, "IT", dto[0].RealmName)
	mockRepo.AssertExpectations(t)
}

// Неизвестный тип сущности или осиротевшая запись реалм не резолвят: запись
// остаётся без реалма (невидима realm-выборкам), но операция не падает.
func TestActivityLogService_Create_SkipsUnresolvableRealm(t *testing.T) {
	mockRepo, svc := activityLogFixtures()

	entityID := uuid.New()
	dto := []*models.ActivityLogDTO{
		{Action: "created", EntityType: "comment", EntityID: entityID},
	}
	mockRepo.On("GetRealmByEntity", mock.Anything, mock.Anything, "comment", entityID).Return(uuid.Nil, "", false, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, mock.MatchedBy(func(got []*models.ActivityLogDTO) bool {
		return len(got) == 1 && got[0].RealmID == nil
	})).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	assert.Nil(t, dto[0].RealmID)
	mockRepo.AssertExpectations(t)
}
