package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

// CategoryService — сервис работы с категориями тикетов.
// Для подсчёта незакрытых заявок использует узкий TicketCounter (см. groups.go),
// т.к. CategoryService — зависимость TicketService, и внедрить сервис тикетов нельзя.
type CategoryService struct {
	repo    repository.Categories
	tickets TicketCounter
}

// NewCategoryService создаёт CategoryService.
func NewCategoryService(repo repository.Categories, tickets TicketCounter) *CategoryService {
	return &CategoryService{repo: repo, tickets: tickets}
}

// Categories — интерфейс работы с категориями.
type Categories interface {
	// Get возвращает список категорий.
	Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error)
	// GetByID возвращает категорию по идентификатору.
	GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error)
	// Create создаёт категорию.
	Create(ctx context.Context, dto *models.CategoryDTO) error
	// Update обновляет категорию.
	Update(ctx context.Context, dto *models.CategoryDTO) error
	// Delete удаляет категорию.
	Delete(ctx context.Context, dto *models.DelCategoryDTO) error
}

// Get возвращает список категорий.
func (s *CategoryService) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories. error: %w", err)
	}
	return data, nil
}

// GetByID возвращает категорию по идентификатору.
func (s *CategoryService) GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category by id. error: %w", err)
	}
	return data, nil
}

// Create создаёт категорию.
func (s *CategoryService) Create(ctx context.Context, dto *models.CategoryDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create category. error: %w", err)
	}
	return nil
}

// Update обновляет категорию.
func (s *CategoryService) Update(ctx context.Context, dto *models.CategoryDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update category. error: %w", err)
	}
	return nil
}

// Delete удаляет категорию.
func (s *CategoryService) Delete(ctx context.Context, dto *models.DelCategoryDTO) error {
	count, err := s.tickets.CountNotClosedByCategory(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to count not closed tickets in category: %w", err)
	}
	if count > 0 {
		return models.ErrCategoryHasOpenTickets
	}

	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete category. error: %w", err)
	}
	return nil
}
