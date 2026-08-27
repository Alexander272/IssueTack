package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// RoleHierarchyService управляет иерархией ролей (наследованием ролей).
type RoleHierarchyService struct {
	repo repository.RoleHierarchy
}

// NewRoleHierarchyService создаёт сервис иерархии ролей.
func NewRoleHierarchyService(repo repository.RoleHierarchy) *RoleHierarchyService {
	return &RoleHierarchyService{
		repo: repo,
	}
}

// RoleHierarchy описывает сервис управления иерархией ролей.
type RoleHierarchy interface {
	LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error)
	GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error)
	AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error
	AddInheritances(ctx context.Context, tx postgres.Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error
	RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error
	RemoveInheritances(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error
}

// LoadPolicy возвращает связи наследования ролей для загрузки политик Casbin.
func (s *RoleHierarchyService) LoadPolicy(ctx context.Context) ([]*models.SyncRoleInheritance, error) {
	data, err := s.repo.LoadPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}

// GetDirectChildren возвращает непосредственных потомков для указанных ролей.
func (s *RoleHierarchyService) GetDirectChildren(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	data, err := s.repo.GetDirectChildren(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct children: %w", err)
	}
	return data, nil
}

// GetInheritedRoles возвращает роли-предков, от которых наследуют указанные роли.
func (s *RoleHierarchyService) GetInheritedRoles(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	data, err := s.repo.GetInheritedRoles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get inherited roles: %w", err)
	}
	return data, nil
}

// GetRoleDescendants возвращает всех потомков для указанных ролей.
func (s *RoleHierarchyService) GetRoleDescendants(ctx context.Context, req *models.GetRolesInheritance) (map[string][]string, error) {
	data, err := s.repo.GetRoleDescendants(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get role descendants: %w", err)
	}
	return data, nil
}

// SyncRoleInheritance возвращает данные наследования ролей для синхронизации политик.
func (s *RoleHierarchyService) SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error) {
	data, err := s.repo.SyncRoleInheritance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sync role inheritance: %w", err)
	}
	return data, nil
}

// AddInheritance добавляет наследование роли от родительской роли, запрещая наследование от самой себя.
func (s *RoleHierarchyService) AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	// Проверка: нельзя наследовать от себя
	if dto.ParentRoleID == dto.RoleID {
		return models.ErrCannotInheritFromSelf
	}

	if err := s.repo.AddInheritance(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to add inheritance. error: %w", err)
	}
	return nil
}

// RemoveInheritance удаляет наследование роли от родительской роли.
func (s *RoleHierarchyService) RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	if err := s.repo.RemoveInheritance(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to remove inheritance. error: %w", err)
	}
	return nil
}

// AddInheritances добавляет наследование роли от нескольких родительских ролей, запрещая наследование от самой себя.
func (s *RoleHierarchyService) AddInheritances(ctx context.Context, tx postgres.Tx, realmID uuid.UUID, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	for _, parentID := range parentRoleIDs {
		if roleID == parentID {
			return models.ErrCannotInheritFromSelf
		}
	}

	if err := s.repo.AddInheritances(ctx, tx, realmID, roleID, parentRoleIDs); err != nil {
		return fmt.Errorf("failed to add inheritances. error: %w", err)
	}
	return nil
}

// RemoveInheritances удаляет наследование роли от нескольких родительских ролей.
func (s *RoleHierarchyService) RemoveInheritances(ctx context.Context, tx postgres.Tx, roleID uuid.UUID, parentRoleIDs []uuid.UUID) error {
	if err := s.repo.RemoveInheritances(ctx, tx, roleID, parentRoleIDs); err != nil {
		return fmt.Errorf("failed to remove inheritances. error: %w", err)
	}
	return nil
}
