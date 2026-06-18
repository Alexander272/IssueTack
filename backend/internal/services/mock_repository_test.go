package services

import (
	"context"
	"io"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockPermissionsRepo struct {
	mock.Mock
}

func (m *MockPermissionsRepo) LoadPolicy(ctx context.Context) ([]*models.Permission, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Permission), args.Error(1)
}
func (m *MockPermissionsRepo) Sync(ctx context.Context, tx postgres.Tx, dto []*models.PermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockPermissionsRepo) GetById(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Permission), args.Error(1)
}
func (m *MockPermissionsRepo) GetAll(ctx context.Context) ([]*models.Permission, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Permission), args.Error(1)
}
func (m *MockPermissionsRepo) GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.Permission), args.Error(1)
}
func (m *MockPermissionsRepo) GetInheritedByRole(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).(map[uuid.UUID]struct{}), args.Error(1)
}
func (m *MockPermissionsRepo) GetRolePermissionsMap(ctx context.Context, tx postgres.Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, tx, roleID)
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}
func (m *MockPermissionsRepo) GetRolePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, tx, roleID)
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}
func (m *MockPermissionsRepo) ReplacePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, roleID, permissionIDs)
	return args.Error(0)
}
func (m *MockPermissionsRepo) Count(ctx context.Context, req *models.GetPermsCountDTO) (*models.PermsWithCount, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.PermsWithCount), args.Error(1)
}
func (m *MockPermissionsRepo) CountForAll(ctx context.Context, roleToDescendants map[string][]string) (map[string]models.PermsWithCount, error) {
	args := m.Called(ctx, roleToDescendants)
	return args.Get(0).(map[string]models.PermsWithCount), args.Error(1)
}
func (m *MockPermissionsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.PermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockPermissionsRepo) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeletePermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockPermissionsRepo) DeleteByKeys(ctx context.Context, tx postgres.Tx, dto []*models.PermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockPermissionsRepo) GetGrouped(ctx context.Context) ([]*models.GroupedPermission, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.GroupedPermission), args.Error(1)
}
func (m *MockPermissionsRepo) GetInherited(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}

type MockRolesRepo struct {
	mock.Mock
}

func (m *MockRolesRepo) GetOne(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}
func (m *MockRolesRepo) GetAll(ctx context.Context) ([]*models.Role, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Role), args.Error(1)
}
func (m *MockRolesRepo) GetUserCount(ctx context.Context, roleIDs []string) (map[string]int, error) {
	args := m.Called(ctx, roleIDs)
	return args.Get(0).(map[string]int), args.Error(1)
}
func (m *MockRolesRepo) GetIDsBySlugs(ctx context.Context, realmID uuid.UUID, slugs []string) (map[string]uuid.UUID, error) {
	args := m.Called(ctx, realmID, slugs)
	return args.Get(0).(map[string]uuid.UUID), args.Error(1)
}
func (m *MockRolesRepo) IsExists(ctx context.Context, realmID uuid.UUID, roleName string) (bool, error) {
	args := m.Called(ctx, realmID, roleName)
	return args.Bool(0), args.Error(1)
}
func (m *MockRolesRepo) IsExistsById(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
func (m *MockRolesRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.RoleDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRolesRepo) Update(ctx context.Context, tx postgres.Tx, dto *models.RoleDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRolesRepo) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteRoleDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRolesRepo) AssignPermission(ctx context.Context, tx postgres.Tx, dto *models.RolePermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRolesRepo) AssignPermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, roleID, permissionIDs)
	return args.Error(0)
}
func (m *MockRolesRepo) DeletePermission(ctx context.Context, tx postgres.Tx, dto *models.RolePermissionDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockRealmsRepo struct {
	mock.Mock
}

func (m *MockRealmsRepo) GetAll(ctx context.Context) ([]*models.Realm, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.Realm), args.Error(1)
}

func (m *MockRealmsRepo) GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Realm), args.Error(1)
}
func (m *MockRealmsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.RealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRealmsRepo) Update(ctx context.Context, tx postgres.Tx, dto *models.RealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRealmsRepo) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockRoleHierarchyRepo struct {
	mock.Mock
}

func (m *MockRoleHierarchyRepo) LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.SyncRoleInheritance), args.Error(1)
}
func (m *MockRoleHierarchyRepo) GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyRepo) GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyRepo) GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyRepo) SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.SyncRoleInheritance), args.Error(1)
}
func (m *MockRoleHierarchyRepo) AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRoleHierarchyRepo) AddInheritances(ctx context.Context, tx postgres.Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, realmID, roleID, parentRoleIDs)
	return args.Error(0)
}
func (m *MockRoleHierarchyRepo) RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRoleHierarchyRepo) RemoveInheritances(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, roleID, parentRoleIDs)
	return args.Error(0)
}

type MockTicketsRepo struct {
	mock.Mock
}

func (m *MockTicketsRepo) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.Ticket), args.Error(1)
}
func (m *MockTicketsRepo) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}
func (m *MockTicketsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.TicketDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockTicketsRepo) Update(ctx context.Context, tx postgres.Tx, dto *models.TicketDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockTicketsRepo) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeleteTicketDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockGroupsRepo struct {
	mock.Mock
}

func (m *MockGroupsRepo) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Group), args.Error(1)
}
func (m *MockGroupsRepo) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.Group), args.Error(1)
}
func (m *MockGroupsRepo) Create(ctx context.Context, dto *models.GroupDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockGroupsRepo) Update(ctx context.Context, dto *models.GroupDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockGroupsRepo) Delete(ctx context.Context, dto *models.DelGroupDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockGroupsRepo) GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.User), args.Error(1)
}
func (m *MockGroupsRepo) GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	args := m.Called(ctx, groupID)
	return args.Int(0), args.Error(1)
}
func (m *MockGroupsRepo) GetManagedGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockGroupsRepo) GetMemberGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockGroupsRepo) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, groupID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *MockGroupsRepo) AddMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockGroupsRepo) RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}

type MockActivityLogService struct {
	mock.Mock
}

func (m *MockActivityLogService) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.ActivityLog), args.Error(1)
}
func (m *MockActivityLogService) Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockSubtaskService struct {
	mock.Mock
}

func (m *MockSubtaskService) GetByTicketID(ctx context.Context, ticketID, actorID uuid.UUID) ([]*models.Subtask, error) {
	args := m.Called(ctx, ticketID, actorID)
	return args.Get(0).([]*models.Subtask), args.Error(1)
}
func (m *MockSubtaskService) GetByID(ctx context.Context, req *models.GetSubtaskDTO, actorID uuid.UUID) (*models.Subtask, error) {
	args := m.Called(ctx, req, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subtask), args.Error(1)
}
func (m *MockSubtaskService) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtaskService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtaskService) Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtaskService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockAttachmentService struct {
	mock.Mock
}

func (m *MockAttachmentService) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID, actorID uuid.UUID) ([]*models.Attachment, error) {
	args := m.Called(ctx, entityType, entityID, actorID)
	return args.Get(0).([]*models.Attachment), args.Error(1)
}
func (m *MockAttachmentService) Upload(ctx context.Context, tx postgres.Tx, entityType string, entityID uuid.UUID, fileName string, file io.Reader, uploadedBy uuid.UUID) (*models.Attachment, error) {
	args := m.Called(ctx, tx, entityType, entityID, fileName, file, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Attachment), args.Error(1)
}
func (m *MockAttachmentService) Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID, actorID uuid.UUID) error {
	args := m.Called(ctx, tx, id, actorID)
	return args.Error(0)
}

type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) TicketCreated(ctx context.Context, dto *models.TicketDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockNotificationService) TicketUpdated(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, changes []*models.FieldChange) error {
	args := m.Called(ctx, ticketID, actorID, changes)
	return args.Error(0)
}
func (m *MockNotificationService) TicketDeleted(ctx context.Context, ticket *models.Ticket) error {
	args := m.Called(ctx, ticket)
	return args.Error(0)
}
func (m *MockNotificationService) SendUnread(ctx context.Context, client *ws_hub.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) LoadPolicy(ctx context.Context) ([]*models.UserRole, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.UserRole), args.Error(1)
}
func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*models.UserData, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserData), args.Error(1)
}
func (m *MockUserService) GetByLogin(ctx context.Context, login string) (*models.UserData, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserData), args.Error(1)
}
func (m *MockUserService) GetAll(ctx context.Context) ([]*models.UserData, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.UserData), args.Error(1)
}
func (m *MockUserService) Sync(ctx context.Context, actor *models.Actor) error {
	args := m.Called(ctx, actor)
	return args.Error(0)
}
func (m *MockUserService) UpdateAccount(ctx context.Context, dto *models.UpdateAccountDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}

type mockTransactionManager struct{}

func (m *mockTransactionManager) WithinTransaction(ctx context.Context, fn func(tx postgres.Tx) error) error {
	return fn(nil)
}

type MockAccessPolices struct {
	mock.Mock
}

func (m *MockAccessPolices) Enforce(sub, dom, obj, act string) (bool, error) {
	args := m.Called(sub, dom, obj, act)
	return args.Bool(0), args.Error(1)
}
func (m *MockAccessPolices) Reload() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockAccessPolices) GetPolicies(user, domain string) (*models.Access, error) {
	args := m.Called(user, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Access), args.Error(1)
}

type MockRoleHierarchyService struct {
	mock.Mock
}

func (m *MockRoleHierarchyService) LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.SyncRoleInheritance), args.Error(1)
}
func (m *MockRoleHierarchyService) GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyService) GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyService) GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(map[string][]string), args.Error(1)
}
func (m *MockRoleHierarchyService) AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRoleHierarchyService) AddInheritances(ctx context.Context, tx postgres.Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, realmID, roleID, parentRoleIDs)
	return args.Error(0)
}
func (m *MockRoleHierarchyService) RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockRoleHierarchyService) RemoveInheritances(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	args := m.Called(ctx, tx, roleID, parentRoleIDs)
	return args.Error(0)
}

type MockTicketAccessChecker struct {
	mock.Mock
}

func (m *MockTicketAccessChecker) CheckAccess(ctx context.Context, ticketID, userID uuid.UUID, action string) error {
	args := m.Called(ctx, ticketID, userID, action)
	return args.Error(0)
}

type MockActivityLogRepo struct {
	mock.Mock
}

func (m *MockActivityLogRepo) Get(ctx context.Context, req *models.GetLogsDTO) ([]*models.ActivityLog, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.ActivityLog), args.Error(1)
}
func (m *MockActivityLogRepo) Create(ctx context.Context, tx postgres.Tx, dto []*models.ActivityLogDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockSubtasksRepo struct {
	mock.Mock
}

func (m *MockSubtasksRepo) GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*models.Subtask, error) {
	args := m.Called(ctx, ticketID)
	return args.Get(0).([]*models.Subtask), args.Error(1)
}
func (m *MockSubtasksRepo) GetByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subtask), args.Error(1)
}
func (m *MockSubtasksRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtasksRepo) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtasksRepo) Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockSubtasksRepo) Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}

type MockAttachmentsRepo struct {
	mock.Mock
}

func (m *MockAttachmentsRepo) GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.Attachment, error) {
	args := m.Called(ctx, entityType, entityID)
	return args.Get(0).([]*models.Attachment), args.Error(1)
}
func (m *MockAttachmentsRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Attachment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Attachment), args.Error(1)
}
func (m *MockAttachmentsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.Attachment) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockAttachmentsRepo) Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID) error {
	args := m.Called(ctx, tx, id)
	return args.Error(0)
}

type MockNotificationsRepo struct {
	mock.Mock
}

func (m *MockNotificationsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.CreateNotificationDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockNotificationsRepo) GetUnread(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Notification), args.Error(1)
}
func (m *MockNotificationsRepo) MarkRead(ctx context.Context, tx postgres.Tx, id uuid.UUID) error {
	args := m.Called(ctx, tx, id)
	return args.Error(0)
}
func (m *MockNotificationsRepo) MarkAllRead(ctx context.Context, tx postgres.Tx, userID uuid.UUID) error {
	args := m.Called(ctx, tx, userID)
	return args.Error(0)
}
func (m *MockNotificationsRepo) GetResponsibleByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *MockNotificationsRepo) GetSettings(ctx context.Context, userID uuid.UUID) (*models.NotificationSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationSettings), args.Error(1)
}

type MockChecklistsRepo struct {
	mock.Mock
}

func (m *MockChecklistsRepo) Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]*models.ChecklistTemplate), args.Error(1)
}
func (m *MockChecklistsRepo) GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO) (*models.ChecklistTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChecklistTemplate), args.Error(1)
}
func (m *MockChecklistsRepo) Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockChecklistsRepo) Update(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockChecklistsRepo) Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO) error {
	args := m.Called(ctx, dto)
	return args.Error(0)
}
func (m *MockChecklistsRepo) GetItems(ctx context.Context, templateID uuid.UUID) ([]*models.ChecklistTemplateItem, error) {
	args := m.Called(ctx, templateID)
	return args.Get(0).([]*models.ChecklistTemplateItem), args.Error(1)
}
func (m *MockChecklistsRepo) SetItems(ctx context.Context, tx postgres.Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO) error {
	args := m.Called(ctx, tx, templateID, items)
	return args.Error(0)
}

type MockUserRealmsRepo struct {
	mock.Mock
}

func (m *MockUserRealmsRepo) GetAll(ctx context.Context) ([]*models.UserRealm, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.UserRealm), args.Error(1)
}
func (m *MockUserRealmsRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserRealm, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.UserRealm), args.Error(1)
}
func (m *MockUserRealmsRepo) GetByUserAndRealm(ctx context.Context, userID, realmID uuid.UUID) (*models.UserRealm, error) {
	args := m.Called(ctx, userID, realmID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserRealm), args.Error(1)
}
func (m *MockUserRealmsRepo) Create(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) Update(ctx context.Context, tx postgres.Tx, dto *models.UserRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) UpdateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) Delete(ctx context.Context, tx postgres.Tx, id uuid.UUID) error {
	args := m.Called(ctx, tx, id)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) DeleteByUserAndRealm(ctx context.Context, tx postgres.Tx, userID, realmID uuid.UUID) error {
	args := m.Called(ctx, tx, userID, realmID)
	return args.Error(0)
}
func (m *MockUserRealmsRepo) DeleteSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserRealmDTO) error {
	args := m.Called(ctx, tx, dto)
	return args.Error(0)
}
