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

func roleServiceFixtures() (*MockRolesRepo, *MockRealmsRepo, *MockRoleHierarchyService, *MockPermissionsRepo, *RoleService) {
	mockRepo := new(MockRolesRepo)
	mockRealms := new(MockRealmsRepo)
	mockHierarchy := new(MockRoleHierarchyService)
	mockPerms := new(MockPermissionsRepo)

	svc := &RoleService{
		repo:      mockRepo,
		realms:    mockRealms,
		hierarchy: mockHierarchy,
		perms:     mockPerms,
		eventBus:  &events.PolicyEventManager{},
		tm:        &mockTransactionManager{},
	}
	return mockRepo, mockRealms, mockHierarchy, mockPerms, svc
}

func TestRoleService_GetOne(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	req := &models.GetRoleDTO{ID: uuid.New()}
	expected := &models.Role{ID: req.ID, Name: "admin", Slug: "admin"}
	mockRepo.On("GetOne", mock.Anything, req).Return(expected, nil)

	got, err := svc.GetOne(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestRoleService_GetOne_NotFound(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	req := &models.GetRoleDTO{ID: uuid.New()}
	mockRepo.On("GetOne", mock.Anything, req).Return(nil, models.ErrNotFound)

	got, err := svc.GetOne(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, models.ErrNotFound)
}

func TestRoleService_GetAll(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	expected := []*models.Role{
		{ID: uuid.New(), Name: "admin"},
		{ID: uuid.New(), Name: "user"},
	}
	mockRepo.On("GetAll", mock.Anything).Return(expected, nil)

	got, err := svc.GetAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestRoleService_IsExists(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	realmID := uuid.New()
	mockRepo.On("IsExists", mock.Anything, realmID, "admin").Return(true, nil)

	exists, err := svc.IsExists(context.Background(), realmID, "admin")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRoleService_Create_Success(t *testing.T) {
	mockRepo, mockRealms, mockHierarchy, _, svc := roleServiceFixtures()

	realmID := uuid.New()
	permID := uuid.New().String()
	dto := &models.RoleDTO{
		ID:          uuid.New(),
		RealmID:     realmID,
		Name:        "moderator",
		Slug:        "moderator",
		Permissions: []string{permID},
		Inherits:    []string{"admin"},
		Actor:       &models.Actor{ID: uuid.New(), Name: "test"},
	}

	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockRepo.On("GetIDsBySlugs", mock.Anything, realmID, []string{"admin"}).Return(map[string]uuid.UUID{"admin": uuid.New()}, nil)
	mockRepo.On("AssignPermissions", mock.Anything, nil, dto.ID, mock.Anything).Return(nil)
	mockHierarchy.On("AddInheritances", mock.Anything, nil, realmID, dto.ID, mock.Anything).Return(nil)
	mockRealms.On("GetByID", mock.Anything, &models.GetRealmByIdDTO{ID: realmID}).Return(&models.Realm{Name: "test-realm"}, nil)

	err := svc.Create(context.Background(), dto)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockHierarchy.AssertExpectations(t)
}

func TestRoleService_Create_EmptyPermissions(t *testing.T) {
	_, _, _, _, svc := roleServiceFixtures()

	dto := &models.RoleDTO{
		ID:          uuid.New(),
		RealmID:     uuid.New(),
		Permissions: []string{},
		Actor:       &models.Actor{ID: uuid.New(), Name: "test"},
	}

	err := svc.Create(context.Background(), dto)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestRoleService_Create_ParentNotFound(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	realmID := uuid.New()
	permID := uuid.New().String()
	dto := &models.RoleDTO{
		ID:          uuid.New(),
		RealmID:     realmID,
		Name:        "moderator",
		Permissions: []string{permID},
		Inherits:    []string{"nonexistent"},
		Actor:       &models.Actor{ID: uuid.New(), Name: "test"},
	}

	mockRepo.On("Create", mock.Anything, nil, dto).Return(nil)
	mockRepo.On("GetIDsBySlugs", mock.Anything, realmID, []string{"nonexistent"}).Return(map[string]uuid.UUID{}, nil)

	err := svc.Create(context.Background(), dto)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "parent role not found")
}

func TestRoleService_Update_Success(t *testing.T) {
	mockRepo, mockRealms, mockHierarchy, mockPerms, svc := roleServiceFixtures()

	roleID := uuid.New()
	realmID := uuid.New()
	dto := &models.RoleDTO{
		ID:          roleID,
		RealmID:     realmID,
		Name:        "moderator-updated",
		Slug:        "moderator",
		Level:       2,
		Description: "updated desc",
		Permissions: []string{},
		Actor:       &models.Actor{ID: uuid.New(), Name: "test"},
	}

	oldRole := &models.Role{
		ID:         roleID,
		Name:       "moderator",
		Slug:       "moderator",
		Level:      1,
		Realm:      realmID.String(),
		IsEditable: true,
	}

	mockRepo.On("GetOne", mock.Anything, &models.GetRoleDTO{ID: roleID}).Return(oldRole, nil)
	mockRepo.On("Update", mock.Anything, nil, dto).Return(nil)
	mockRepo.On("GetIDsBySlugs", mock.Anything, realmID, mock.Anything).Return(map[string]uuid.UUID{}, nil)
	mockHierarchy.On("GetRoleDescendants", mock.Anything, &models.GetRolesInheritance{Roles: []string{"moderator"}}).Return(map[string][]string{"moderator": {}}, nil)
	mockPerms.On("GetRolePermissions", mock.Anything, nil, roleID).Return(map[uuid.UUID]bool{}, nil)
	mockPerms.On("ReplacePermissions", mock.Anything, nil, roleID, []uuid.UUID{}).Return(nil)
	mockRealms.On("GetByID", mock.Anything, &models.GetRealmByIdDTO{ID: realmID}).Return(&models.Realm{Name: "test-realm"}, nil)

	err := svc.Update(context.Background(), dto)
	assert.NoError(t, err)
}

func TestRoleService_Update_RoleNotEditable(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	roleID := uuid.New()
	dto := &models.RoleDTO{ID: roleID, Actor: &models.Actor{ID: uuid.New(), Name: "test"}}

	mockRepo.On("GetOne", mock.Anything, &models.GetRoleDTO{ID: roleID}).Return(&models.Role{
		ID: roleID, IsEditable: false,
	}, nil)

	err := svc.Update(context.Background(), dto)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrRoleNotEditable)
}

func TestRoleService_Delete_Success(t *testing.T) {
	mockRepo, mockRealms, _, _, svc := roleServiceFixtures()

	roleID := uuid.New()
	realmID := uuid.New()
	dto := &models.DeleteRoleDTO{ID: roleID, Actor: &models.Actor{ID: uuid.New(), Name: "test"}}

	role := &models.Role{
		ID: roleID, Name: "custom-role", Slug: "custom-role",
		IsSystem: false, IsEditable: true, Realm: realmID.String(),
	}

	mockRepo.On("GetOne", mock.Anything, &models.GetRoleDTO{ID: roleID}).Return(role, nil)
	mockRepo.On("Delete", mock.Anything, nil, dto).Return(nil)
	mockRealms.On("GetByID", mock.Anything, &models.GetRealmByIdDTO{ID: realmID}).Return(&models.Realm{Name: "test-realm"}, nil)

	err := svc.Delete(context.Background(), dto)
	assert.NoError(t, err)
}

func TestRoleService_Delete_Reserved(t *testing.T) {
	mockRepo, _, _, _, svc := roleServiceFixtures()

	roleID := uuid.New()
	dto := &models.DeleteRoleDTO{ID: roleID, Actor: &models.Actor{ID: uuid.New(), Name: "test"}}

	mockRepo.On("GetOne", mock.Anything, &models.GetRoleDTO{ID: roleID}).Return(&models.Role{
		ID: roleID, IsSystem: true, IsEditable: false,
	}, nil)

	err := svc.Delete(context.Background(), dto)
	assert.Error(t, err)
	assert.ErrorIs(t, err, models.ErrReservedRole)
}

func TestRoleService_GetOneWithPermissions(t *testing.T) {
	mockRepo, _, mockHierarchy, mockPerms, svc := roleServiceFixtures()

	roleID := uuid.New()
	req := &models.GetRoleDTO{ID: roleID}
	role := &models.Role{ID: roleID, Name: "admin", Slug: "admin"}

	mockRepo.On("GetOne", mock.Anything, req).Return(role, nil)
	mockHierarchy.On("GetDirectChildren", mock.Anything, &models.GetRolesInheritance{Roles: []string{"admin"}}).Return(map[string][]string{"admin": {"user"}}, nil)
	mockPerms.On("GetAll", mock.Anything).Return([]*models.Permission{}, nil)
	mockPerms.On("GetRolePermissions", mock.Anything, nil, roleID).Return(map[uuid.UUID]bool{}, nil)
	mockPerms.On("GetInherited", mock.Anything, roleID).Return(map[uuid.UUID]bool{}, nil)

	result, err := svc.GetOneWithPermissions(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "admin", result.Role.Slug)
	assert.Equal(t, []string{"user"}, result.Inherits)
}

func TestRoleService_SetPermissions(t *testing.T) {
	mockPerms := new(MockPermissionsRepo)
	_, _, _, _, svc := roleServiceFixtures()
	svc.perms = mockPerms

	roleID := uuid.New()
	permID := uuid.New().String()
	mockPerms.On("ReplacePermissions", mock.Anything, nil, roleID, mock.Anything).Return(nil)

	err := svc.SetPermissions(context.Background(), roleID.String(), []string{permID})
	assert.NoError(t, err)
}

func TestRoleService_SetPermissions_InvalidRoleID(t *testing.T) {
	_, _, _, _, svc := roleServiceFixtures()

	err := svc.SetPermissions(context.Background(), "invalid-uuid", []string{})
	assert.Error(t, err)
}

func TestRoleService_GetWithStats(t *testing.T) {
	mockRepo, _, mockHierarchy, mockPerms, svc := roleServiceFixtures()

	roleID := uuid.New()
	roles := []*models.Role{
		{ID: roleID, Name: "admin", Slug: "admin"},
	}

	mockRepo.On("GetAll", mock.Anything).Return(roles, nil)
	mockHierarchy.On("GetDirectChildren", mock.Anything, &models.GetRolesInheritance{Roles: []string{"admin"}}).Return(map[string][]string{"admin": {}}, nil)
	mockHierarchy.On("GetRoleDescendants", mock.Anything, &models.GetRolesInheritance{Roles: []string{"admin"}}).Return(map[string][]string{}, nil)
	mockPerms.On("CountForAll", mock.Anything, mock.Anything).Return(map[string]models.PermsWithCount{}, nil)
	mockRepo.On("GetUserCount", mock.Anything, []string{roleID.String()}).Return(map[string]int{roleID.String(): 5}, nil)

	result, err := svc.GetWithStats(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "admin", result[0].Role.Slug)
	assert.Equal(t, 5, result[0].UserCount)
}
