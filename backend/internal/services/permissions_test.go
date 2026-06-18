package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPermissionService_GetAll(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	expected := []*models.Permission{
		{ID: uuid.New(), Object: "ticket", Action: "read"},
	}
	mockRepo.On("GetAll", mock.Anything).Return(expected, nil)

	got, err := svc.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	mockRepo.AssertExpectations(t)
}

func TestPermissionService_GetAll_RepoError(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	mockRepo.On("GetAll", mock.Anything).Return([]*models.Permission(nil), assert.AnError)

	got, err := svc.GetAll(context.Background())
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPermissionService_GetByID(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	id := uuid.New()
	expected := &models.Permission{ID: id, Object: "ticket", Action: "read"}
	mockRepo.On("GetById", mock.Anything, id).Return(expected, nil)

	got, err := svc.GetByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestPermissionService_Create(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	dto := &models.PermissionDTO{Object: "ticket", Action: "read"}
	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)

	err := svc.Create(context.Background(), nil, dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPermissionService_Delete(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	dto := &models.DeletePermissionDTO{ID: uuid.New()}
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)

	err := svc.Delete(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestPermissionService_GetGrouped(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	perms := []*models.Permission{
		{ID: uuid.New(), Object: "ticket", Action: "read"},
		{ID: uuid.New(), Object: "ticket", Action: "write"},
		{ID: uuid.New(), Object: "role", Action: "read"},
	}
	mockRepo.On("GetAll", mock.Anything).Return(perms, nil)

	got, err := svc.GetGrouped(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 2) // ticket + role groups

	// Verify groups are sorted by OrderOfResources
	assert.Equal(t, "ticket", got[0].Group)
	assert.Equal(t, "role", got[1].Group)
}

func TestPermissionService_GetGrouped_Empty(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	mockRepo.On("GetAll", mock.Anything).Return([]*models.Permission{}, nil)

	got, err := svc.GetGrouped(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestPermissionService_GetGrouped_UnknownSlug(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	perms := []*models.Permission{
		{ID: uuid.New(), Object: "unknown_resource", Action: "read"},
	}
	mockRepo.On("GetAll", mock.Anything).Return(perms, nil)

	got, err := svc.GetGrouped(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	// Unknown slug should fall back to its string value as title
	assert.Equal(t, "unknown_resource", got[0].Group)
}

func TestPermissionService_GetRolePermissions(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	roleID := uuid.New()
	expected := map[uuid.UUID]bool{uuid.New(): true}
	mockRepo.On("GetRolePermissionsMap", mock.Anything, nil, roleID).Return(expected, nil)

	got, err := svc.GetRolePermissions(context.Background(), nil, roleID)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestPermissionService_GetInherited(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	roleID := uuid.New()
	id1 := uuid.New()
	mockRepo.On("GetInheritedByRole", mock.Anything, roleID).Return(map[uuid.UUID]struct{}{id1: {}}, nil)

	got, err := svc.GetInherited(context.Background(), roleID)
	assert.NoError(t, err)
	assert.True(t, got[id1])
}

func TestPermissionService_ReplacePermissions(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	roleID := uuid.New()
	permIDs := []uuid.UUID{uuid.New()}
	mockRepo.On("ReplacePermissions", mock.Anything, nil, roleID, permIDs).Return(nil)

	err := svc.ReplacePermissions(context.Background(), nil, roleID, permIDs)
	assert.NoError(t, err)
}

func TestPermissionService_DeleteByKeys(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	svc := &PermissionService{repo: mockRepo}

	dto := []*models.PermissionDTO{{Object: "ticket", Action: "read"}}
	mockRepo.On("DeleteByKeys", mock.Anything, nil, dto).Return(nil)

	err := svc.DeleteByKeys(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestNewPermissionService_SyncSuccess(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	eventBus := &events.PolicyEventManager{}

	mockRepo.On("Sync", mock.Anything, nil, mock.Anything).Return(nil)
	mockRepo.On("DeleteByKeys", mock.Anything, nil, mock.Anything).Return(nil)

	svc, err := NewPermissionService(mockRepo, &mockTransactionManager{}, eventBus)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
	mockRepo.AssertExpectations(t)
}

func TestNewPermissionService_SyncFail(t *testing.T) {
	mockRepo := new(MockPermissionsRepo)
	eventBus := &events.PolicyEventManager{}

	mockRepo.On("Sync", mock.Anything, nil, mock.Anything).Return(assert.AnError)

	svc, err := NewPermissionService(mockRepo, &mockTransactionManager{}, eventBus)
	assert.Error(t, err)
	assert.Nil(t, svc)
}
