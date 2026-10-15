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
		INSERT INTO %s (id, realm_id, name, description, sort_order, is_hidden)
		VALUES ($1, $2, $3, '', 0, $4)`,
		Tables.CategoryGroups), id, realmID, name, hidden)
	require.NoError(t, err)
	return id
}

// itCategory создаёт категорию с заданными скрытостью и разделом.
func itCategory(t *testing.T, ctx context.Context, pool *pgxpool.Pool, realmID, groupID uuid.UUID, name string, hidden bool, sectionID *uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, realm_id, name, description, group_id, category_group_id, is_hidden)
		VALUES ($1, $2, $3, '', $4, $5, $6)`,
		Tables.Categories), id, realmID, name, groupID, sectionID, hidden)
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

	itCategory(t, ctx, pool, realmID, groupID, "обычная", false, nil)
	hiddenID := itCategory(t, ctx, pool, realmID, groupID, "скрытая", true, nil)
	itCategory(t, ctx, pool, realmID, groupID, "в обычном разделе", false, &visibleSection)
	itCategory(t, ctx, pool, realmID, groupID, "в скрытом разделе", false, &hiddenSection)

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
