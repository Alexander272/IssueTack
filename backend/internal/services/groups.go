package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// GroupService — сервис работы с группами и их участниками.
type GroupService struct {
	repo      repository.Groups
	txManager TransactionManager
}

// NewGroupService создаёт GroupService.
func NewGroupService(repo repository.Groups, txManager TransactionManager) *GroupService {
	return &GroupService{repo: repo, txManager: txManager}
}

// Groups — интерфейс работы с группами.
type Groups interface {
	// Get возвращает список групп вместе с их участниками.
	Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	// GetByID возвращает группу вместе с участниками.
	GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	// Create создаёт группу.
	Create(ctx context.Context, dto *models.GroupDTO) error
	// Update обновляет группу.
	Update(ctx context.Context, dto *models.GroupDTO) error
	// Delete удаляет группу.
	Delete(ctx context.Context, dto *models.DelGroupDTO) error

	// AddMember добавляет участника в группу.
	AddMember(ctx context.Context, dto *models.GroupMemberDTO) error
	// RemoveMember удаляет участника из группы.
	RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error
	// GetMembers возвращает участников группы.
	GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error)
	// GetMemberCount возвращает количество участников группы.
	GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error)
	// GetManagedGroups возвращает идентификаторы групп, которыми управляет пользователь.
	GetManagedGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
	// GetMemberGroups возвращает идентификаторы групп, участником которых является пользователь.
	GetMemberGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
	// IsMember проверяет, является ли пользователь участником группы.
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// AddMember добавляет участника в группу.
func (s *GroupService) AddMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	if err := s.repo.AddMember(ctx, dto); err != nil {
		return fmt.Errorf("failed to add member. error: %w", err)
	}
	return nil
}

// RemoveMember удаляет участника из группы.
func (s *GroupService) RemoveMember(ctx context.Context, dto *models.GroupMemberDTO) error {
	if err := s.repo.RemoveMember(ctx, dto); err != nil {
		return fmt.Errorf("failed to remove member. error: %w", err)
	}
	return nil
}

// GetMembers возвращает участников группы.
func (s *GroupService) GetMembers(ctx context.Context, req *models.GetGroupDTO) ([]*models.UserShort, error) {
	data, err := s.repo.GetMembers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get members. error: %w", err)
	}
	return data, nil
}

// GetMemberCount возвращает количество участников группы.
func (s *GroupService) GetMemberCount(ctx context.Context, groupID uuid.UUID) (int, error) {
	count, err := s.repo.GetMemberCount(ctx, groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count. error: %w", err)
	}
	return count, nil
}

// GetManagedGroups возвращает идентификаторы групп, которыми управляет пользователь.
func (s *GroupService) GetManagedGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.repo.GetManagedGroups(ctx, userID, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed groups. error: %w", err)
	}
	return ids, nil
}

// GetMemberGroups возвращает идентификаторы групп, участником которых является пользователь.
func (s *GroupService) GetMemberGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.repo.GetMemberGroups(ctx, userID, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member groups. error: %w", err)
	}
	return ids, nil
}

// IsMember проверяет, является ли пользователь участником группы.
func (s *GroupService) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	ok, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check membership. error: %w", err)
	}
	return ok, nil
}

// Get возвращает список групп вместе с их участниками.
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

// GetByID возвращает группу вместе с её участниками.
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

// Create создаёт группу в рамках транзакции.
func (s *GroupService) Create(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.Create(ctx, tx, dto)
	}); err != nil {
		return fmt.Errorf("failed to create group. error: %w", err)
	}
	return nil
}

// Update обновляет группу в рамках транзакции.
func (s *GroupService) Update(ctx context.Context, dto *models.GroupDTO) error {
	if err := s.txManager.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.Update(ctx, tx, dto)
	}); err != nil {
		return fmt.Errorf("failed to update group. error: %w", err)
	}
	return nil
}

// Delete удаляет группу.
func (s *GroupService) Delete(ctx context.Context, dto *models.DelGroupDTO) error {
	//TODO возможно надо проверить все ли тикеты в этой группе закрыты

	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete group. error: %w", err)
	}
	return nil
}
