package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

// CategoryGroupService — сервис работы с разделами категорий (таксономия).
// Раздел не влияет на права и маршрутизацию, в отличие от группы-владельца
// категории (categories.group_id), и не связан с тикетами.
type CategoryGroupService struct {
	repo repository.CategoryGroups
}

// NewCategoryGroupService создаёт CategoryGroupService.
func NewCategoryGroupService(repo repository.CategoryGroups) *CategoryGroupService {
	return &CategoryGroupService{repo: repo}
}

// CategoryGroups — интерфейс работы с разделами категорий.
type CategoryGroups interface {
	// Get возвращает список разделов реалма, отсортированный по sort_order.
	Get(ctx context.Context, req *models.GetCategoryGroupsDTO) ([]*models.CategoryGroup, error)
	// GetByID возвращает раздел по идентификатору.
	GetByID(ctx context.Context, req *models.GetCategoryGroupByIdDTO) (*models.CategoryGroup, error)
	// Create создаёт раздел.
	Create(ctx context.Context, dto *models.CategoryGroupDTO) error
	// Update обновляет раздел.
	Update(ctx context.Context, dto *models.CategoryGroupDTO) error
	// Delete удаляет раздел, если в нём нет категорий.
	Delete(ctx context.Context, dto *models.DelCategoryGroupDTO) error
}

// Get возвращает список разделов.
func (s *CategoryGroupService) Get(ctx context.Context, req *models.GetCategoryGroupsDTO) ([]*models.CategoryGroup, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category groups. error: %w", err)
	}
	return data, nil
}

// GetByID возвращает раздел по идентификатору.
func (s *CategoryGroupService) GetByID(ctx context.Context, req *models.GetCategoryGroupByIdDTO) (*models.CategoryGroup, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category group by id. error: %w", err)
	}
	return data, nil
}

// Create создаёт раздел.
func (s *CategoryGroupService) Create(ctx context.Context, dto *models.CategoryGroupDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create category group. error: %w", err)
	}
	return nil
}

// Update обновляет раздел.
func (s *CategoryGroupService) Update(ctx context.Context, dto *models.CategoryGroupDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update category group. error: %w", err)
	}
	return nil
}

// Delete удаляет раздел. Если на раздел ссылаются категории, удаление
// запрещается — иначе они молча потеряли бы группировку (ON DELETE SET NULL).
func (s *CategoryGroupService) Delete(ctx context.Context, dto *models.DelCategoryGroupDTO) error {
	count, err := s.repo.CountByCategory(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to count categories in category group: %w", err)
	}
	if count > 0 {
		return models.ErrCategoryGroupInUse
	}

	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete category group. error: %w", err)
	}
	return nil
}
