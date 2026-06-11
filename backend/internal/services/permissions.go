package services

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type PermissionService struct {
	repo     repository.Permissions
	tm       TransactionManager
	eventBus *events.PolicyEventManager
}

func NewPermissionService(repo repository.Permissions, tm TransactionManager, eventBus *events.PolicyEventManager) *PermissionService {
	s := &PermissionService{
		repo:     repo,
		tm:       tm,
		eventBus: eventBus,
	}

	if err := s.Sync(context.Background()); err != nil {
		panic(err)
	}

	return s
}

type Permissions interface {
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.Permission, error)
	GetResources(ctx context.Context) []access.Resource
	GetAll(ctx context.Context) ([]*models.Permission, error)
	GetGrouped(ctx context.Context) ([]*models.GroupedPermission, error)
	GetRolePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error)
	GetInherited(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]bool, error)
	ReplacePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error)
	Count(ctx context.Context, req *models.GetPermsCountDTO) (*models.PermsWithCount, error)
	CountForAll(ctx context.Context, roleToDescendants map[string][]string) (map[string]models.PermsWithCount, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.PermissionDTO) error
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DeletePermissionDTO) error
}

func (s *PermissionService) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.Permission, error) {
	data, err := s.repo.LoadPolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}

func (s *PermissionService) Sync(ctx context.Context) error {
	accesses := access.Reg.List()
	dto := make([]*models.PermissionDTO, 0, len(accesses))

	for _, res := range accesses {
		// Собираем список действий для текущего ресурса
		var actionsToSync []access.ActionCode

		if _, ok := res.AllowedActions[access.All]; ok {
			actionsToSync = access.AllActions
		} else {
			for action := range res.AllowedActions {
				actionsToSync = append(actionsToSync, action)
			}
		}

		// Наполняем DTO и ключи
		for _, act := range actionsToSync {
			objStr := string(res.Slug)
			actStr := string(act)

			dto = append(dto, &models.PermissionDTO{
				Object:      objStr,
				Action:      actStr,
				Description: res.Description,
			})
		}
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Sync(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to sync permissions: %w", err)
		}

		if err := s.DeleteByKeys(ctx, tx, dto); err != nil {
			return err
		}
		return nil
	})
}

func (s *PermissionService) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	data, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission by id: %w", err)
	}
	return data, nil
}

// ! Deprecated
func (s *PermissionService) GetResources(ctx context.Context) []access.Resource {
	accesses := access.Reg.List()

	for i, res := range accesses {
		// Собираем список действий для текущего ресурса
		var actionsToSync []access.ActionCode

		if _, ok := res.AllowedActions[access.All]; ok {
			actionsToSync = access.AllActions
		} else {
			for action := range res.AllowedActions {
				actionsToSync = append(actionsToSync, action)
			}
		}

		actions := map[access.ActionCode]struct{}{}
		for _, act := range actionsToSync {
			actions[act] = struct{}{}
		}

		accesses[i].AllowedActions = actions
	}

	return accesses
}

func (s *PermissionService) GetByRole(ctx context.Context, req *models.GetPermsByRoleDTO) ([]*models.Permission, error) {
	data, err := s.repo.GetByRole(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions by role: %w", err)
	}
	return data, nil
}

func (s *PermissionService) GetAll(ctx context.Context) ([]*models.Permission, error) {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all permissions: %w", err)
	}
	return data, nil
}

func (s *PermissionService) GetGrouped(ctx context.Context) ([]*models.GroupedPermission, error) {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all permissions: %w", err)
	}

	res := make([]*models.GroupedPermission, 0, len(data))
	for i := 0; i < len(data); i++ {
		if i == 0 || data[i].Object != data[i-1].Object {
			resource, _ := access.Reg.GetBySlug(access.ResourceSlug(data[i].Object))
			res = append(res, &models.GroupedPermission{
				Group: data[i].Object,
				Title: resource.Name,
				Items: make([]*models.Permission, 0),
			})
		}
		res[len(res)-1].Items = append(res[len(res)-1].Items, data[i])
	}

	slices.SortFunc(res, func(a, b *models.GroupedPermission) int {
		return cmp.Compare(
			access.OrderOfResources[access.ResourceSlug(a.Group)],
			access.OrderOfResources[access.ResourceSlug(b.Group)],
		)
	})

	return res, nil
}

func (s *PermissionService) GetRolePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	data, err := s.repo.GetRolePermissionsMap(ctx, tx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	return data, nil
}

func (s *PermissionService) GetInherited(ctx context.Context, roleID uuid.UUID) (map[uuid.UUID]bool, error) {
	inheritedIDs, err := s.repo.GetInheritedByRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inherited permissions: %w", err)
	}

	result := make(map[uuid.UUID]bool)
	for id := range inheritedIDs {
		result[id] = true
	}
	return result, nil
}

func (s *PermissionService) ReplacePermissions(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	err := s.repo.ReplacePermissions(ctx, tx, roleID, permissionIDs)
	if err != nil {
		return fmt.Errorf("failed to replace permissions: %w", err)
	}
	return nil
}

func (s *PermissionService) Count(ctx context.Context, req *models.GetPermsCountDTO) (*models.PermsWithCount, error) {
	data, err := s.repo.Count(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions count: %w", err)
	}
	return data, nil
}
func (s *PermissionService) CountForAll(ctx context.Context, roleToDescendants map[string][]string) (map[string]models.PermsWithCount, error) {
	data, err := s.repo.CountForAll(ctx, roleToDescendants)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions count for all: %w", err)
	}
	return data, nil
}

func (s *PermissionService) Create(ctx context.Context, tx postgres.Tx, dto *models.PermissionDTO) error {
	// if constants.ResourcesList.Permissions

	err := s.repo.Create(ctx, tx, dto)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (s *PermissionService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DeletePermissionDTO) error {
	err := s.repo.Delete(ctx, tx, dto)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}
	return nil
}

func (s *PermissionService) DeleteByKeys(ctx context.Context, tx postgres.Tx, dto []*models.PermissionDTO) error {
	err := s.repo.DeleteByKeys(ctx, tx, dto)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}
	return nil
}
