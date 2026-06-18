package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func roleHierarchyFixtures() (*MockRoleHierarchyRepo, *RoleHierarchyService) {
	mockRepo := new(MockRoleHierarchyRepo)
	svc := NewRoleHierarchyService(mockRepo)
	return mockRepo, svc
}

func TestRoleHierarchyService_LoadPolicy(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	expected := []*models.SyncRoleInheritance{
		{Role: "admin", ParentRole: "user", Realm: "default"},
	}
	mockRepo.On("LoadPolicy", mock.Anything).Return(expected, nil)

	got, err := svc.LoadPolicy(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleHierarchyService_GetDirectChildren(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	req := &models.GetRolesInheritance{Roles: []string{"admin"}}
	expected := map[string][]string{"admin": {"user"}}
	mockRepo.On("GetDirectChildren", mock.Anything, req).Return(expected, nil)

	got, err := svc.GetDirectChildren(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleHierarchyService_GetInheritedRoles(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	req := &models.GetRolesInheritance{Roles: []string{"user"}}
	expected := map[string][]string{"user": {"admin"}}
	mockRepo.On("GetInheritedRoles", mock.Anything, req).Return(expected, nil)

	got, err := svc.GetInheritedRoles(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleHierarchyService_GetRoleDescendants(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	req := &models.GetRolesInheritance{Roles: []string{"admin"}}
	expected := map[string][]string{"admin": {"user", "moderator"}}
	mockRepo.On("GetRoleDescendants", mock.Anything, req).Return(expected, nil)

	got, err := svc.GetRoleDescendants(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleHierarchyService_SyncRoleInheritance(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	req := &models.GetRoleInheritance{Role: "admin", Realm: "default"}
	expected := []*models.SyncRoleInheritance{
		{Role: "admin", ParentRole: "user", Realm: "default"},
	}
	mockRepo.On("SyncRoleInheritance", mock.Anything, req).Return(expected, nil)

	got, err := svc.SyncRoleInheritance(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleHierarchyService_AddInheritance_Success(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	roleID := uuid.New()
	parentID := uuid.New()
	dto := &models.RoleHierarchyDTO{
		RoleID:       roleID,
		ParentRoleID: parentID,
		RealmID:      uuid.New(),
	}

	mockRepo.On("AddInheritance", mock.Anything, nil, dto).Return(nil)

	err := svc.AddInheritance(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestRoleHierarchyService_AddInheritance_SelfInherit(t *testing.T) {
	_, svc := roleHierarchyFixtures()

	roleID := uuid.New()
	dto := &models.RoleHierarchyDTO{
		RoleID:       roleID,
		ParentRoleID: roleID,
	}

	err := svc.AddInheritance(context.Background(), nil, dto)
	assert.ErrorIs(t, err, models.ErrCannotInheritFromSelf)
}

func TestRoleHierarchyService_AddInheritances_Success(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	roleID := uuid.New()
	realmID := uuid.New()
	parentIDs := []uuid.UUID{uuid.New(), uuid.New()}

	mockRepo.On("AddInheritances", mock.Anything, nil, realmID, roleID, parentIDs).Return(nil)

	err := svc.AddInheritances(context.Background(), nil, realmID, roleID, parentIDs)
	assert.NoError(t, err)
}

func TestRoleHierarchyService_AddInheritances_SelfInherit(t *testing.T) {
	_, svc := roleHierarchyFixtures()

	roleID := uuid.New()
	parentIDs := []uuid.UUID{uuid.New(), roleID}

	err := svc.AddInheritances(context.Background(), nil, uuid.New(), roleID, parentIDs)
	assert.ErrorIs(t, err, models.ErrCannotInheritFromSelf)
}

func TestRoleHierarchyService_RemoveInheritance(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	dto := &models.RoleHierarchyDTO{RoleID: uuid.New(), ParentRoleID: uuid.New()}
	mockRepo.On("RemoveInheritance", mock.Anything, nil, dto).Return(nil)

	err := svc.RemoveInheritance(context.Background(), nil, dto)
	assert.NoError(t, err)
}

func TestRoleHierarchyService_RemoveInheritances(t *testing.T) {
	mockRepo, svc := roleHierarchyFixtures()

	roleID := uuid.New()
	parentIDs := []uuid.UUID{uuid.New()}
	mockRepo.On("RemoveInheritances", mock.Anything, nil, roleID, parentIDs).Return(nil)

	err := svc.RemoveInheritances(context.Background(), nil, roleID, parentIDs)
	assert.NoError(t, err)
}
