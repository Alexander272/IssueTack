package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// CategoryModifierService — справочник подкатегорий. Изменение закрыто
// Casbin-правом category:write (начальник области и выше); сервис дополнительно
// проверяет, что подкатегория принадлежит категории того же реалма.
type CategoryModifierService struct {
	repo       repository.CategoryModifiers
	categories repository.Categories
}

func NewCategoryModifierService(repo repository.CategoryModifiers, categories repository.Categories) *CategoryModifierService {
	return &CategoryModifierService{repo: repo, categories: categories}
}

type CategoryModifiers interface {
	Get(ctx context.Context, req *models.GetCategoryModifiersDTO) ([]*models.CategoryModifier, error)
	GetByID(ctx context.Context, req *models.GetCategoryModifierByIDDTO) (*models.CategoryModifier, error)
	Create(ctx context.Context, dto *models.CategoryModifierDTO) error
	Update(ctx context.Context, dto *models.CategoryModifierDTO) error
	Delete(ctx context.Context, dto *models.DelCategoryModifierDTO) error
}

func (s *CategoryModifierService) Get(ctx context.Context, req *models.GetCategoryModifiersDTO) ([]*models.CategoryModifier, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category modifiers. error: %w", err)
	}
	return data, nil
}

func (s *CategoryModifierService) GetByID(ctx context.Context, req *models.GetCategoryModifierByIDDTO) (*models.CategoryModifier, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get category modifier by id. error: %w", err)
	}
	return data, nil
}

func (s *CategoryModifierService) Create(ctx context.Context, dto *models.CategoryModifierDTO) error {
	if err := s.validate(ctx, dto); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create category modifier. error: %w", err)
	}
	return nil
}

func (s *CategoryModifierService) Update(ctx context.Context, dto *models.CategoryModifierDTO) error {
	if dto.ID == nil {
		return models.ErrInvalidInput
	}
	// Категорию в реалм перевезти нельзя — она остаётся той, с которой
	// подкатегория создана.
	old, err := s.repo.GetByID(ctx, &models.GetCategoryModifierByIDDTO{ID: *dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return err
	}
	dto.CategoryID = old.CategoryID
	if err := s.validate(ctx, dto); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update category modifier. error: %w", err)
	}
	return nil
}

func (s *CategoryModifierService) Delete(ctx context.Context, dto *models.DelCategoryModifierDTO) error {
	count, err := s.repo.CountTickets(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to count tickets with category modifier: %w", err)
	}
	if count > 0 {
		return models.ErrCategoryModifierInUse
	}
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete category modifier. error: %w", err)
	}
	return nil
}

func (s *CategoryModifierService) validate(ctx context.Context, dto *models.CategoryModifierDTO) error {
	if dto.CategoryID == uuid.Nil || dto.Name == "" {
		return models.ErrInvalidInput
	}
	// Ключи хранятся в нормализованном виде (без пустых и дублей) — матчер
	// рассчитывает на это и не должен разбирать мусор из формы.
	dto.Keywords = normalizeModifierKeywords(dto.Keywords)
	if dto.DefaultDueDays != nil && *dto.DefaultDueDays < 0 {
		return models.ErrInvalidInput
	}
	if dto.Priority != nil && !dto.Priority.IsValid() {
		return models.ErrInvalidInput
	}
	if _, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: dto.RealmID}); err != nil {
		return err
	}
	return nil
}
