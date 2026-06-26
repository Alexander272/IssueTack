package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type GroupService struct {
	repo      repository.Groups
	txManager TransactionManager
}

func NewGroupService(repo repository.Groups, txManager TransactionManager) *GroupService {
	return &GroupService{repo: repo, txManager: txManager}
}

type Groups interface {
	Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	Create(ctx context.Context, dto *models.GroupDTO) error
	Update(ctx context.Context, dto *models.GroupDTO) error
	Delete(ctx context.Context, dto *models.DelGroupDTO) error

	AddMember(ctx context.Context, dto *models.GroupMemberDTO) error
	RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error
	GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error)
	GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error)
	GetManagedGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetMemberGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

func (s *GroupService) AddMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	if err := s.repo.AddMember(ctx, dto); err != nil {
		return fmt.Errorf("failed to add member. error: %w", err)
	}
	return nil
}

func (s *GroupService) RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	if err := s.repo.RemoveMember(ctx, dto); err != nil {
		return fmt.Errorf("failed to remove member. error: %w", err)
	}
	return nil
}

func (s *GroupService) GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error) {
	data, err := s.repo.GetMembers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get members. error: %w", err)
	}
	return data, nil
}

func (s *GroupService) GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	count, err := s.repo.GetMemberCount(ctx, groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count. error: %w", err)
	}
	return count, nil
}

func (s *GroupService) GetManagedGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.repo.GetManagedGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed groups. error: %w", err)
	}
	return ids, nil
}

func (s *GroupService) GetMemberGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.repo.GetMemberGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member groups. error: %w", err)
	}
	return ids, nil
}

func (s *GroupService) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	ok, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check membership. error: %w", err)
	}
	return ok, nil
}

func (s *GroupService) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups. error: %w", err)
	}
	if len(data) == 0 {
		return data, nil
	}

	ids := make([]uuid.UUID, len(data))
	for i, g := range data {
		ids[i] = g.ID
	}

	members, err := s.repo.GetMembersMap(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get members map. error: %w", err)
	}

	for _, g := range data {
		if m, ok := members[g.ID]; ok {
			g.Members = m
		}
	}

	return data, nil
}

func (s *GroupService) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by id. error: %w", err)
	}

	members, err := s.repo.GetMembers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get members. error: %w", err)
	}
	data.Members = members

	return data, nil
}

func (s *GroupService) Create(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.Create(ctx, tx, dto)
	}); err != nil {
		return fmt.Errorf("failed to create group. error: %w", err)
	}
	return nil
}

func (s *GroupService) Update(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.Update(ctx, tx, dto)
	}); err != nil {
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
