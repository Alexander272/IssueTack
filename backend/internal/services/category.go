package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// CategoryService — сервис работы с категориями тикетов.
// Для подсчёта незакрытых заявок использует узкий TicketCounter (см. groups.go),
// т.к. CategoryService — зависимость TicketService, и внедрить сервис тикетов нельзя.
type CategoryService struct {
	repo       repository.Categories
	tickets    TicketCounter
	visibility *CategoryVisibility
}

// NewCategoryService создаёт CategoryService.
func NewCategoryService(repo repository.Categories, tickets TicketCounter, visibility *CategoryVisibility) *CategoryService {
	return &CategoryService{repo: repo, tickets: tickets, visibility: visibility}
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
	// Reorder меняет глобальный порядок категорий реалма.
	Reorder(ctx context.Context, dto *models.ReorderCategoriesDTO) error
	// CanSeeHidden сообщает, видит ли пользователь скрытые категории/разделы
	// (менеджеры и исполнители — да, безгрупповой заказчик — нет). Списочные
	// обработчики кладут результат в GetCategoriesDTO.IncludeHidden.
	CanSeeHidden(ctx context.Context, userID, realmID uuid.UUID) bool
}

// CanSeeHidden делегирует единому правилу видимости скрытых категорий.
func (s *CategoryService) CanSeeHidden(ctx context.Context, userID, realmID uuid.UUID) bool {
	return s.visibility.CanSeeHidden(ctx, userID, realmID)
}

// Get возвращает список категорий.
func (s *CategoryService) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories. error: %w", err)
	}
	return data, nil
}

// filterVisibleCategories убирает эффективно скрытые категории (собственная
// скрытость или скрытый раздел). Используется там, где список категорий берётся
// из кэша целиком (plugin context) и его нужно сузить под безгруппового
// заказчика: менеджеры и исполнители получают полный список.
func filterVisibleCategories(categories []*models.Category) []*models.Category {
	visible := make([]*models.Category, 0, len(categories))
	for _, c := range categories {
		if c == nil || c.IsHidden {
			continue
		}
		if c.CategoryGroup != nil && c.CategoryGroup.IsHidden {
			continue
		}
		visible = append(visible, c)
	}
	return visible
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

// Reorder меняет глобальный порядок категорий реалма: категории из запроса
// идут в указанном порядке, остальные дописываются следом в текущем —
// последовательность остаётся сплошной (1..N), без дыр и дубликатов.
func (s *CategoryService) Reorder(ctx context.Context, dto *models.ReorderCategoriesDTO) error {
	// Скрытые категории тоже участвуют в порядке — иначе они «всплыли» бы
	// в начало при первой же перестановке видимых.
	all, err := s.repo.Get(ctx, &models.GetCategoriesDTO{RealmID: dto.RealmID, IncludeHidden: true})
	if err != nil {
		return fmt.Errorf("failed to get categories for reorder. error: %w", err)
	}

	known := make(map[uuid.UUID]struct{}, len(all))
	for _, category := range all {
		known[category.ID] = struct{}{}
	}

	ordered := make([]uuid.UUID, 0, len(all))
	seen := make(map[uuid.UUID]struct{}, len(dto.CategoryIDs))
	for _, id := range dto.CategoryIDs {
		if _, ok := known[id]; !ok {
			return fmt.Errorf("%w: категория %s не найдена в реалме", models.ErrInvalidInput, id)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	for _, category := range all {
		if _, ok := seen[category.ID]; !ok {
			ordered = append(ordered, category.ID)
		}
	}

	if err := s.repo.Reorder(ctx, &models.ReorderCategoriesDTO{RealmID: dto.RealmID, CategoryIDs: ordered}); err != nil {
		return fmt.Errorf("failed to reorder categories. error: %w", err)
	}
	return nil
}
