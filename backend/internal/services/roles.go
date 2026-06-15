package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func parseUUIDs(ids []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id %s: %w", id, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}

type RoleService struct {
	repo      repository.Roles
	realms    repository.Realms
	hierarchy RoleHierarchy
	perms     Permissions
	eventBus  *events.PolicyEventManager
	tm        TransactionManager
}

type RoleDeps struct {
	Repo        repository.Roles
	Realms      repository.Realms
	Hierarchy   RoleHierarchy
	Permissions Permissions
	EventBus    *events.PolicyEventManager
	TM          TransactionManager
}

func NewRolesService(deps *RoleDeps) *RoleService {
	return &RoleService{
		repo:      deps.Repo,
		realms:    deps.Realms,
		hierarchy: deps.Hierarchy,
		perms:     deps.Permissions,
		eventBus:  deps.EventBus,
		tm:        deps.TM,
	}
}

type Roles interface {
	GetOne(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error)
	GetAll(ctx context.Context) ([]*models.Role, error)
	GetOneWithPermissions(ctx context.Context, req *models.GetRoleDTO) (*models.RoleWithPerms, error)
	GetWithStats(ctx context.Context) ([]*models.RoleWithStats, error)
	GetPermissionsGrouped(ctx context.Context, req *models.GetRoleDTO) ([]*models.RolePermissionsGrouped, error)
	IsExists(ctx context.Context, realmID uuid.UUID, roleName string) (bool, error)
	Create(ctx context.Context, dto *models.RoleDTO) error
	Update(ctx context.Context, dto *models.RoleDTO) error
	Delete(ctx context.Context, dto *models.DeleteRoleDTO) error
	AssignPermission(ctx context.Context, dto *models.RolePermissionDTO) error
	DeletePermission(ctx context.Context, dto *models.RolePermissionDTO) error
	SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error
}

func (s *RoleService) GetOne(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error) {
	data, err := s.repo.GetOne(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return data, nil
}

func (s *RoleService) GetAll(ctx context.Context) ([]*models.Role, error) {
	data, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all roles: %w", err)
	}
	return data, nil
}

func (s *RoleService) GetOneWithPermissions(ctx context.Context, req *models.GetRoleDTO) (*models.RoleWithPerms, error) {
	data, err := s.GetOne(ctx, req)
	if err != nil {
		return nil, err
	}

	children, err := s.hierarchy.GetDirectChildren(ctx, &models.GetRolesInheritance{Roles: []string{data.Slug}})
	if err != nil {
		return nil, err
	}

	perms, err := s.GetPermissionsGrouped(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &models.RoleWithPerms{
		Role:     *data,
		Inherits: children[data.Slug],
		Perms:    perms,
	}
	return res, nil
}

func (s *RoleService) GetWithStats(ctx context.Context) ([]*models.RoleWithStats, error) {
	roles, err := s.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]string, 0, len(roles))
	slugs := make([]string, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID.String())
		slugs = append(slugs, role.Slug)
	}

	var (
		userCounts  map[string]int
		permsCounts map[string]models.PermsWithCount
		children    map[string][]string
		descendants map[string][]string
	)

	g, asyncCtx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		var err error
		children, err = s.hierarchy.GetDirectChildren(asyncCtx, &models.GetRolesInheritance{Roles: slugs})
		return err
	})
	g.Go(func() error {
		var err error
		descendants, err = s.hierarchy.GetRoleDescendants(asyncCtx, &models.GetRolesInheritance{Roles: slugs})
		return err
	})
	g.Go(func() error {
		var err error
		userCounts, err = s.repo.GetUserCount(asyncCtx, roleIDs)
		if err != nil {
			return fmt.Errorf("failed to get user count: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if len(descendants) > 0 {
		permsCounts, err = s.perms.CountForAll(ctx, descendants)
		if err != nil {
			return nil, err
		}
	}

	result := make([]*models.RoleWithStats, 0, len(roles))
	for _, r := range roles {
		result = append(result, &models.RoleWithStats{
			Role:       *r,
			Children:   children[r.Slug],
			UserCount:  userCounts[r.ID.String()],
			PermsCount: permsCounts[r.Slug],
		})
	}

	return result, nil
}

func (s *RoleService) GetPermissionsGrouped(ctx context.Context, req *models.GetRoleDTO) ([]*models.RolePermissionsGrouped, error) {
	allPerms, err := s.perms.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	assigned, err := s.perms.GetRolePermissions(ctx, nil, req.ID)
	if err != nil {
		return nil, err
	}

	inherited, err := s.perms.GetInherited(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	logger.Debug("permissions", logger.IntAttr("assigned", len(assigned)), logger.IntAttr("inherited", len(inherited)))

	grouped := make(map[string][]*models.RolePermissionItem)
	for _, p := range allPerms {
		item := &models.RolePermissionItem{
			PermissionID: p.ID,
			Object:       p.Object,
			Action:       p.Action,
			IsAssigned:   assigned[p.ID],
			IsInherited:  inherited[p.ID],
		}
		grouped[p.Object] = append(grouped[p.Object], item)
	}

	result := make([]*models.RolePermissionsGrouped, 0, len(grouped))
	for group, resources := range grouped {
		res, _ := access.Reg.GetBySlug(access.ResourceSlug(group))
		result = append(result, &models.RolePermissionsGrouped{
			Group:     group,
			Title:     res.Name,
			Resources: resources,
		})
	}

	return result, nil
}

func (s *RoleService) IsExists(ctx context.Context, realmID uuid.UUID, roleName string) (bool, error) {
	data, err := s.repo.IsExists(ctx, realmID, roleName)
	if err != nil {
		return false, fmt.Errorf("failed to check if role exists: %w", err)
	}
	return data, nil
}

func (s *RoleService) realmName(ctx context.Context, id uuid.UUID) string {
	realm, err := s.realms.GetByID(ctx, &models.GetRealmByIdDTO{ID: id})
	if err != nil {
		return ""
	}
	return realm.Name
}

func (s *RoleService) Create(ctx context.Context, dto *models.RoleDTO) error {
	if len(dto.Permissions) == 0 {
		return models.ErrInvalidInput
	}

	permIDs, err := parseUUIDs(dto.Permissions)
	if err != nil {
		return err
	}

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		err := s.repo.Create(ctx, tx, dto)
		if err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}

		inheritIDs := make([]uuid.UUID, 0, len(dto.Inherits))
		if len(dto.Inherits) > 0 {
			roleIDs, err := s.repo.GetIDsBySlugs(ctx, dto.RealmID, dto.Inherits)
			if err != nil {
				return fmt.Errorf("failed to get role IDs: %w", err)
			}
			for _, slug := range dto.Inherits {
				if id, ok := roleIDs[slug]; ok {
					inheritIDs = append(inheritIDs, id)
				} else {
					return fmt.Errorf("parent role not found: %s", slug)
				}
			}

			if err := s.hierarchy.AddInheritances(ctx, tx, dto.RealmID, dto.ID, inheritIDs); err != nil {
				return fmt.Errorf("failed to add inheritances: %w", err)
			}
		}

		if err := s.repo.AssignPermissions(ctx, tx, dto.ID, permIDs); err != nil {
			return fmt.Errorf("failed to assign permissions: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	entity := dto.Name
	newValues, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("failed to marshal new values: %w", err)
	}
	s.eventBus.Notify(events.PolicyEvent{
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		Action:        "CREATE",
		EntityType:    "role",
		Entity:        &entity,
		EntityID:      &dto.ID,
		RealmID:       &dto.RealmID,
		RealmName:     s.realmName(ctx, dto.RealmID),
		NewValues:     newValues,
	})

	return nil
}

func (s *RoleService) Update(ctx context.Context, dto *models.RoleDTO) error {
	oldRole, err := s.GetOne(ctx, &models.GetRoleDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if !oldRole.IsEditable {
		return fmt.Errorf("error while updating role %s: %w", dto.Name, models.ErrRoleNotEditable)
	}

	roleChange := false
	inheritChange := false
	permChange := false
	fieldsChanged := oldRole.Name != dto.Name || oldRole.Slug != dto.Slug || oldRole.Level != dto.Level || oldRole.Description != dto.Description
	if fieldsChanged {
		roleChange = true
	}

	var assigned map[uuid.UUID]bool
	var oldValues, newValues []byte

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.Update(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}

		var currentInherits map[string][]string
		if len(dto.Inherits) > 0 {
			currentInherits, err = s.hierarchy.GetRoleDescendants(ctx, &models.GetRolesInheritance{Roles: []string{dto.Slug}})
			if err != nil {
				return fmt.Errorf("failed to get current inherits: %w", err)
			}

			currentSlugs := currentInherits[dto.Slug]
			toAdd := make([]string, 0, len(dto.Inherits))
			toRemove := make([]string, 0, len(currentSlugs))

			currentMap := make(map[string]bool)
			for _, s := range currentSlugs {
				currentMap[s] = true
			}

			newMap := make(map[string]bool)
			for _, s := range dto.Inherits {
				newMap[s] = true
			}

			for _, s := range dto.Inherits {
				if !currentMap[s] {
					toAdd = append(toAdd, s)
				}
			}
			for _, s := range currentSlugs {
				if !newMap[s] {
					toRemove = append(toRemove, s)
				}
			}

			if len(toAdd) > 0 || len(toRemove) > 0 {
				inheritChange = true
			}

			if len(toAdd) > 0 {
				addIDs, err := s.repo.GetIDsBySlugs(ctx, dto.RealmID, toAdd)
				if err != nil {
					return fmt.Errorf("failed to get role IDs: %w", err)
				}
				inheritIDs := make([]uuid.UUID, 0, len(toAdd))
				for _, s := range toAdd {
					if id, ok := addIDs[s]; ok {
						inheritIDs = append(inheritIDs, id)
					}
				}

				logger.Debug("inheritances",
					logger.IntAttr("count", len(inheritIDs)),
					logger.StringAttr("role", dto.ID.String()),
					logger.StringAttr("parents", strings.Join(dto.Inherits, ", ")))

				if err := s.hierarchy.AddInheritances(ctx, tx, dto.RealmID, dto.ID, inheritIDs); err != nil {
					return err
				}
			}

			if len(toRemove) > 0 {
				removeIDs, err := s.repo.GetIDsBySlugs(ctx, dto.RealmID, toRemove)
				if err != nil {
					return fmt.Errorf("failed to get role IDs: %w", err)
				}
				parentIDs := make([]uuid.UUID, 0, len(toRemove))
				for _, s := range toRemove {
					if id, ok := removeIDs[s]; ok {
						parentIDs = append(parentIDs, id)
					}
				}
				if err := s.hierarchy.RemoveInheritances(ctx, tx, dto.ID, parentIDs); err != nil {
					return err
				}
			}
		}

		if dto.Permissions != nil {
			permChange = true

			assigned, err = s.perms.GetRolePermissions(ctx, tx, dto.ID)
			if err != nil {
				return err
			}

			permIDs, err := parseUUIDs(dto.Permissions)
			if err != nil {
				return err
			}
			if err := s.perms.ReplacePermissions(ctx, tx, dto.ID, permIDs); err != nil {
				return err
			}
		}

		oldMap := make(map[string]interface{})
		newM := make(map[string]interface{})
		if roleChange {
			oldMap["name"] = oldRole.Name
			oldMap["slug"] = oldRole.Slug
			oldMap["level"] = oldRole.Level
			oldMap["description"] = oldRole.Description

			newM["name"] = dto.Name
			newM["slug"] = dto.Slug
			newM["level"] = dto.Level
			newM["description"] = dto.Description
		}
		if inheritChange {
			oldMap["inherits"] = currentInherits
			newM["inherits"] = dto.Inherits
		}
		if permChange {
			perms := make([]string, 0, len(assigned))
			for p := range assigned {
				perms = append(perms, p.String())
			}
			oldMap["permissions"] = perms
			newM["permissions"] = dto.Permissions
		}

		oldValues, err = json.Marshal(oldMap)
		if err != nil {
			return fmt.Errorf("failed to marshal old values: %w", err)
		}
		newValues, err = json.Marshal(newM)
		if err != nil {
			return fmt.Errorf("failed to marshal new values: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	entity := dto.Name
	s.eventBus.Notify(events.PolicyEvent{
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		Action:        "UPDATE",
		EntityType:    "role",
		Entity:        &entity,
		EntityID:      &dto.ID,
		RealmID:       &dto.RealmID,
		RealmName:     s.realmName(ctx, dto.RealmID),
		OldValues:     oldValues,
		NewValues:     newValues,
	})

	return nil
}

func (s *RoleService) Delete(ctx context.Context, dto *models.DeleteRoleDTO) error {
	role, err := s.repo.GetOne(ctx, &models.GetRoleDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	if role.IsSystem || !role.IsEditable {
		return models.ErrReservedRole
	}

	oldValues, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to marshal old values: %w", err)
	}

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.Delete(ctx, tx, dto)
	})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	entity := role.Name
	realmID, err := uuid.Parse(role.Realm)
	if err != nil {
		return fmt.Errorf("failed to parse realm id: %w", err)
	}
	s.eventBus.Notify(events.PolicyEvent{
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		Action:        "DELETE",
		EntityType:    "role",
		Entity:        &entity,
		EntityID:      &dto.ID,
		RealmID:       &realmID,
		RealmName:     s.realmName(ctx, realmID),
		OldValues:     oldValues,
	})

	return nil
}

func (s *RoleService) AssignPermission(ctx context.Context, dto *models.RolePermissionDTO) error {
	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		err := s.repo.AssignPermission(ctx, tx, dto)
		if err != nil {
			return fmt.Errorf("failed to assign permission: %w", err)
		}
		return nil
	})
}

func (s *RoleService) DeletePermission(ctx context.Context, dto *models.RolePermissionDTO) error {
	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		err := s.repo.DeletePermission(ctx, tx, dto)
		if err != nil {
			return fmt.Errorf("failed to delete permission: %w", err)
		}
		return nil
	})
}

func (s *RoleService) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("failed to parse role id: %w", err)
	}

	perms, err := parseUUIDs(permissionIDs)
	if err != nil {
		return err
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.perms.ReplacePermissions(ctx, tx, roleUUID, perms); err != nil {
			return fmt.Errorf("failed to replace permissions: %w", err)
		}
		return nil
	})
}
