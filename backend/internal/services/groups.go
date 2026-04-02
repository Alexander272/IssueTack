package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type GroupService struct {
	repo repository.Groups
}

func NewGroupService(repo repository.Groups) *GroupService {
	return &GroupService{repo: repo}
}

type Groups interface {
	Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	Create(ctx context.Context, dto *models.GroupDTO) error
	Update(ctx context.Context, dto *models.GroupDTO) error
	Delete(ctx context.Context, dto *models.DelGroupDTO) error
}

func (s *GroupService) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups. error: %w", err)
	}
	return data, nil
}

func (s *GroupService) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by id. error: %w", err)
	}
	return data, nil
}

func (s *GroupService) Create(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create group. error: %w", err)
	}
	return nil
}

func (s *GroupService) Update(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update group. error: %w", err)
	}
	return nil
}

func (s *GroupService) Delete(ctx context.Context, dto *models.DelGroupDTO) error {
	//TODO возможно надо проверить все ли тикеты в этой группе закрыты

	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete group. error: %w", err)
	}
	return nil
}
