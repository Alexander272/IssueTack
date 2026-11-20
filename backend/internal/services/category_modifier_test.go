package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockModifierRepo — двойник репозиторийного интерфейса подкатегорий: в
// отличие от mock-сервиса из тикетов, сюда входят счётчики_used для защиты
// удаления справочника.
type mockModifierRepo struct {
	mock.Mock
}

func (m *mockModifierRepo) Get(ctx context.Context, req *models.GetCategoryModifiersDTO) ([]*models.CategoryModifier, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CategoryModifier), args.Error(1)
}
func (m *mockModifierRepo) GetByID(ctx context.Context, req *models.GetCategoryModifierByIDDTO) (*models.CategoryModifier, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CategoryModifier), args.Error(1)
}
func (m *mockModifierRepo) Create(ctx context.Context, dto *models.CategoryModifierDTO) error {
	return m.Called(ctx, dto).Error(0)
}
func (m *mockModifierRepo) Update(ctx context.Context, dto *models.CategoryModifierDTO) error {
	return m.Called(ctx, dto).Error(0)
}
func (m *mockModifierRepo) Delete(ctx context.Context, dto *models.DelCategoryModifierDTO) error {
	return m.Called(ctx, dto).Error(0)
}
func (m *mockModifierRepo) CountTickets(ctx context.Context, id uuid.UUID) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}
func (m *mockModifierRepo) CountTicketsByCategory(ctx context.Context, categoryID uuid.UUID) (int, error) {
	args := m.Called(ctx, categoryID)
	return args.Int(0), args.Error(1)
}

// TestCategoryModifierService_Delete_InUseProtected: подкатегорию, на которую
// ссылаются заявки, удалять нельзя — иначе потерялась бы подпись в истории.
func TestCategoryModifierService_Delete_InUseProtected(t *testing.T) {
	repo := new(mockModifierRepo)
	svc := NewCategoryModifierService(repo, new(MockCategoriesRepo))
	id := uuid.New()
	dto := &models.DelCategoryModifierDTO{ID: id, RealmID: uuid.New()}
	repo.On("CountTickets", mock.Anything, id).Return(3, nil)

	err := svc.Delete(context.Background(), dto)
	require.ErrorIs(t, err, models.ErrCategoryModifierInUse)
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestCategoryModifierService_Delete_UnusedAllowed(t *testing.T) {
	repo := new(mockModifierRepo)
	svc := NewCategoryModifierService(repo, new(MockCategoriesRepo))
	id := uuid.New()
	dto := &models.DelCategoryModifierDTO{ID: id, RealmID: uuid.New()}
	repo.On("CountTickets", mock.Anything, id).Return(0, nil)
	repo.On("Delete", mock.Anything, dto).Return(nil)

	require.NoError(t, svc.Delete(context.Background(), dto))
	repo.AssertExpectations(t)
}

// TestCategoryService_Delete_ModifierInUseProtected: удаление категории с
// подкатегориями CASCADE обнулило бы category_modifier_id у заявок — запрещаем.
func TestCategoryService_Delete_ModifierInUseProtected(t *testing.T) {
	categories := new(MockCategoriesRepo)
	tickets := new(MockTicketsRepo)
	modifiers := new(mockModifierRepo)
	svc := &CategoryService{repo: categories, tickets: tickets, modifiers: modifiers}

	categoryID := uuid.New()
	dto := &models.DelCategoryDTO{ID: categoryID}
	tickets.On("CountNotClosedByCategory", mock.Anything, categoryID).Return(0, nil)
	modifiers.On("CountTicketsByCategory", mock.Anything, categoryID).Return(2, nil)

	err := svc.Delete(context.Background(), dto)
	assert.ErrorIs(t, err, models.ErrCategoryModifierInUse)
	categories.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}
