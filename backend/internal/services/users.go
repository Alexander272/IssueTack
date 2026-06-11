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
}

func (s *userService) LoadPolicy(ctx context.Context, req *models.GetPoliciesDTO) ([]*models.UserRole, error) {
	data, err := s.repo.LoadPolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}
	return data, nil
}
