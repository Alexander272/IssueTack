package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// itCategoryGroup создаёт раздел категорий с заданной скрытостью.
func itCategoryGroup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, realmID uuid.UUID, name string, hidden bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, realm_id, name, description, is_hidden)
		VALUES ($1, $2, $3, '', $4)`,
		Tables.CategoryGroups), id, realmID, name, hidden)
	require.NoError(t, err)
	return id
}

// itCategory создаёт категорию с заданными скрытостью, разделом и позицией
// в глобальном порядке.
func itCategory(t *testing.T, ctx context.Context, pool *pgxpool.Pool, realmID, groupID uuid.UUID, name string, hidden bool, sectionID *uuid.UUID, sortOrder int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, realm_id, name, description, group_id, category_group_id, is_hidden, sort_order)
		VALUES ($1, $2, $3, '', $4, $5, $6, $7)`,
		Tables.Categories), id, realmID, name, groupID, sectionID, hidden, sortOrder)
	require.NoError(t, err)
	return id
}

// Скрытые категории не попадают в Get без IncludeHidden; скрытость раздела
// распространяется на его категории (эффективная скрытость). GetByID скрытость
// не фильтрует — он используется внутренними путями (тикеты, логи).
func TestCategoryRepo_Get_HiddenFilter_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа "+realmID.String())

	visibleSection := itCategoryGroup(t, ctx, pool, realmID, "Обычный раздел", false)
	hiddenSection := itCategoryGroup(t, ctx, pool, realmID, "Скрытый раздел", true)

	itCategory(t, ctx, pool, realmID, groupID, "обычная", false, nil, 1)
	hiddenID := itCategory(t, ctx, pool, realmID, groupID, "скрытая", true, nil, 2)
	itCategory(t, ctx, pool, realmID, groupID, "в обычном разделе", false, &visibleSection, 3)
	itCategory(t, ctx, pool, realmID, groupID, "в скрытом разделе", false, &hiddenSection, 4)

	repo := NewCategoryRepo(pool)

	visible, err := repo.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"обычная", "в обычном разделе"}, categoryNames(visible))

	all, err := repo.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID, IncludeHidden: true})
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{"обычная", "скрытая", "в обычном разделе", "в скрытом разделе"},
		categoryNames(all),
	)

	// GetByID намеренно не фильтрует скрытость.
	byID, err := repo.GetByID(ctx, &models.GetCategoryByIdDTO{ID: hiddenID, RealmID: realmID})
	require.NoError(t, err)
	assert.Equal(t, "скрытая", byID.Name)
	assert.True(t, byID.IsHidden)
}

func categoryNames(categories []*models.Category) []string {
	names := make([]string, 0, len(categories))
	for _, c := range categories {
		names = append(names, c.Name)
	}
	return names
}

// Порядок категорий глобальный: sort_order задаёт позицию в реалме, раздел
// на позицию не влияет — блок раздела стоит там, где стоит его первая
// категория. Reorder перезаписывает последовательность.
func TestCategoryRepo_GlobalOrder_And_Reorder_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа "+realmID.String())
	sectionID := itCategoryGroup(t, ctx, pool, realmID, "Раздел", false)

	first := itCategory(t, ctx, pool, realmID, groupID, "первая", false, &sectionID, 1)
	second := itCategory(t, ctx, pool, realmID, groupID, "вторая", false, nil, 2)
	third := itCategory(t, ctx, pool, realmID, groupID, "третья", false, &sectionID, 3)

	repo := NewCategoryRepo(pool)

	data, err := repo.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	require.NoError(t, err)
	assert.Equal(t, []string{"первая", "вторая", "третья"}, categoryNames(data))

	// Переупорядочивание: категория без раздела встаёт в начало, раздел —
	// следом, порядок внутри раздела сохраняется.
	err = repo.Reorder(ctx, &models.ReorderCategoriesDTO{
		RealmID:     realmID,
		CategoryIDs: []uuid.UUID{second, third, first},
	})
	require.NoError(t, err)

	data, err = repo.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	require.NoError(t, err)
	assert.Equal(t, []string{"вторая", "третья", "первая"}, categoryNames(data))
}
