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

// Reorder задаёт глобальный порядок категорий: присланные id идут первыми,
// не присланные дописываются следом в текущем порядке — последовательность
// остаётся сплошной.
func TestCategoryService_Reorder_KeepsUnlistedCategories(t *testing.T) {
	repo := new(MockCategoriesRepo)
	svc := NewCategoryService(repo, nil, nil)

	realmID := uuid.New()
	first := &models.Category{ID: uuid.New(), Name: "первая"}
	second := &models.Category{ID: uuid.New(), Name: "вторая"}
	third := &models.Category{ID: uuid.New(), Name: "третья"}

	repo.On("Get", mock.Anything, &models.GetCategoriesDTO{RealmID: realmID, IncludeHidden: true}).
		Return([]*models.Category{first, second, third}, nil)

	var applied *models.ReorderCategoriesDTO
	repo.On("Reorder", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			applied = args.Get(1).(*models.ReorderCategoriesDTO)
		}).
		Return(nil)

	err := svc.Reorder(context.Background(), &models.ReorderCategoriesDTO{
		RealmID:     realmID,
		CategoryIDs: []uuid.UUID{third.ID, first.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, applied)
	assert.Equal(t, []uuid.UUID{third.ID, first.ID, second.ID}, applied.CategoryIDs)
	repo.AssertExpectations(t)
}

// Чужую (не из реалма) категорию в порядок не пускаем: иначе можно было бы
// «размыть» последовательность реалма, дописав чужой id.
func TestCategoryService_Reorder_RejectsForeignCategory(t *testing.T) {
	repo := new(MockCategoriesRepo)
	svc := NewCategoryService(repo, nil, nil)

	realmID := uuid.New()
	own := &models.Category{ID: uuid.New(), Name: "своя"}
	repo.On("Get", mock.Anything, mock.Anything).Return([]*models.Category{own}, nil)

	err := svc.Reorder(context.Background(), &models.ReorderCategoriesDTO{
		RealmID:     realmID,
		CategoryIDs: []uuid.UUID{own.ID, uuid.New()},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
	repo.AssertNotCalled(t, "Reorder", mock.Anything, mock.Anything)
}
