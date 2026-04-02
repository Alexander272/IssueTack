package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type CategoryService struct {
	repo repository.Categories
}

func NewCategoryService(repo repository.Categories) *CategoryService {
	return &CategoryService{repo: repo}
}

type Categories interface {
	Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error)
	GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error)
	Create(ctx context.Context, dto *models.CategoryDTO) error
	Update(ctx context.Context, dto *models.CategoryDTO) error
	Delete(ctx context.Context, dto *models.DelCategoryDTO) error
}

func (s *CategoryService) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories. error: %w", err)
	}
	return data, nil
}

func (s *CategoryService) GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category by id. error: %w", err)
	}
	return data, nil
}

func (s *CategoryService) Create(ctx context.Context, dto *models.CategoryDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create category. error: %w", err)
	}
	return nil
}

func (s *CategoryService) Update(ctx context.Context, dto *models.CategoryDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update category. error: %w", err)
	}
	return nil
}

func (s *CategoryService) Delete(ctx context.Context, dto *models.DelCategoryDTO) error {
	//TODO возможно надо проверить все ли тикеты в этой категории закрыты
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete category. error: %w", err)
	}
	return nil
}
