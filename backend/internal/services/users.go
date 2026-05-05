package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type userService struct {
	repo repository.Users
	tm   TransactionManager
}

func NewUserService(repo repository.Users, tm TransactionManager) *userService {
	return &userService{
		repo: repo,
		tm:   tm,
	}
}

type Users interface {
	LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.UserRole, error)
	AssignRole(ctx context.Context, dto *models.UserRoleDTO) error
	DeleteRole(ctx context.Context, dto *models.UserRoleDTO) error
}

func (s *userService) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.UserRole, error) {
	data, err := s.repo.LoadPolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}

func (s *userService) AssignRole(ctx context.Context, dto *models.UserRoleDTO) error {
	//TODO возможно транзакция все же нужна
	// да она нужна для сохранения записи в audit log
	if err := s.repo.AssignRole(ctx, nil, dto); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

func (s *userService) DeleteRole(ctx context.Context, dto *models.UserRoleDTO) error {
	if err := s.repo.DeleteRole(ctx, nil, dto); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}
