package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
)

type RoleHierarchyService struct {
	repo repository.RoleHierarchy
}

func NewRoleHierarchyService(repo repository.RoleHierarchy) *RoleHierarchyService {
	return &RoleHierarchyService{
		repo: repo,
	}
}

type RoleHierarchy interface {
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.SyncRoleInheritance, error)
	AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error
	RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error
}

func (s *RoleHierarchyService) GetInheritedRoles(ctx context.Context, req *models.GetRoleInheritance) ([]string, error) {
	data, err := s.repo.GetInheritedRoles(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get inherited roles: %w", err)
	}
	return data, nil
}

func (s *RoleHierarchyService) SyncRoleInheritance(ctx context.Context, req *models.GetRoleInheritance) ([]*models.SyncRoleInheritance, error) {
	data, err := s.repo.SyncRoleInheritance(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sync role inheritance: %w", err)
	}

	//TODO надо результат передавать в casbin

	return data, nil
}

func (s *RoleHierarchyService) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.SyncRoleInheritance, error) {
	data, err := s.repo.LoadPolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}

func (s *RoleHierarchyService) AddInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	// Проверка: нельзя наследовать от себя
	if dto.ParentRoleID == dto.RoleID {
		return models.ErrCannotInheritFromSelf
	}

	if err := s.repo.AddInheritance(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to add inheritance. error: %w", err)
	}

	//TODO вызов SyncRoleInheritance

	return nil
}

func (s *RoleHierarchyService) RemoveInheritance(ctx context.Context, tx postgres.Tx, dto *models.RoleHierarchyDTO) error {
	if err := s.repo.RemoveInheritance(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to remove inheritance. error: %w", err)
	}

	//TODO В Casbin удаляем g-политику

	return nil
}
