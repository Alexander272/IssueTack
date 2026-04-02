package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type RoleService struct {
	repo repository.Roles
}

func NewRolesService(repo repository.Roles) *RoleService {
	return &RoleService{
		repo: repo,
	}
}

type Roles interface {
	Get(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error)
	IsExists(ctx context.Context, roleName string) (bool, error)
	Create(ctx context.Context, dto *models.RoleDTO) error
	Update(ctx context.Context, dto *models.RoleDTO) error
	Delete(ctx context.Context, dto *models.DeleteRoleDTO) error
	AssignPermission(ctx context.Context, dto *models.RolePermissionDTO) error
	DeletePermission(ctx context.Context, dto *models.RolePermissionDTO) error
}

func (s *RoleService) Get(ctx context.Context, req *models.GetRoleDTO) (*models.Role, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return data, nil
}

func (s *RoleService) IsExists(ctx context.Context, roleName string) (bool, error) {
	data, err := s.repo.IsExists(ctx, roleName)
	if err != nil {
		return false, fmt.Errorf("failed to check if role exists: %w", err)
	}
	return data, nil
}

func (s *RoleService) Create(ctx context.Context, dto *models.RoleDTO) error {
	err := s.repo.Create(ctx, nil, dto)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (s *RoleService) Update(ctx context.Context, dto *models.RoleDTO) error {
	err := s.repo.Update(ctx, nil, dto)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

func (s *RoleService) Delete(ctx context.Context, dto *models.DeleteRoleDTO) error {
	err := s.repo.Delete(ctx, nil, dto)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}
