package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCategoryModifierRepo_RoundTrip_IT проверяет SQL-путь подкатегории:
// запись nullable-приоритета в enum, чтение, вложенность в категориях,
// подпись в заявке и счётчики для защиты удаления справочника.
func TestCategoryModifierRepo_RoundTrip_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	groupID := f.group(ctx, pool, realmID, "Группа "+realmID.String())
	categoryID := f.category(ctx, pool, realmID, "Категория "+realmID.String(), &groupID)
	// TicketRepo.GetByID делает JOIN sites по ticket.site_id. Фикстура заявки
	// площадку не ставит, поэтому создаём её здесь; внешний ключ — SET NULL,
	// так что порядок удаления в cleanup не важен.
	siteID := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`INSERT INTO %s (id, name) VALUES ($1, $2)`, Tables.Sites),
		siteID, "Площадка "+realmID.String())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := pool.Exec(context.Background(), fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Sites), siteID)
		require.NoError(t, cleanupErr)
	})
	repo := NewCategoryModifierRepo(pool)

	days := 4
	priority := models.PriorityHigh
	name := "Модификатор " + realmID.String()[:8]
	dto := &models.CategoryModifierDTO{
		RealmID:        realmID,
		CategoryID:     categoryID,
		Name:           name,
		Description:    "описание",
		Keywords:       []string{"принтер", "тонер картридж"},
		DefaultDueDays: &days,
		Priority:       &priority,
		IsActive:       true,
	}
	require.NoError(t, repo.Create(ctx, dto))
	require.NotNil(t, dto.ID)

	got, err := repo.GetByID(ctx, &models.GetCategoryModifierByIDDTO{ID: *dto.ID, RealmID: realmID})
	require.NoError(t, err)
	assert.Equal(t, name, got.Name)
	assert.Equal(t, categoryID, got.CategoryID)
	assert.Equal(t, []string{"принтер", "тонер картридж"}, got.Keywords)
	assert.Equal(t, &days, got.DefaultDueDays)
	require.NotNil(t, got.Priority)
	assert.Equal(t, models.PriorityHigh, *got.Priority)
	assert.True(t, got.IsActive)

	list, err := repo.Get(ctx, &models.GetCategoryModifiersDTO{RealmID: realmID, CategoryID: &categoryID})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, *dto.ID, list[0].ID)

	// CategoryRepo приклеивает подкатегории к своим категориям — отсюда их
	// получают селекты форм веба и плагина.
	categories, err := NewCategoryRepo(pool).Get(ctx, &models.GetCategoriesDTO{RealmID: realmID, IncludeHidden: true})
	require.NoError(t, err)
	var attached *models.Category
	for _, c := range categories {
		if c.ID == categoryID {
			attached = c
		}
	}
	require.NotNil(t, attached)
	require.Len(t, attached.Modifiers, 1)
	assert.Equal(t, name, attached.Modifiers[0].Name)

	// Заявка читает подпись подкатегории тем же запросом, что и остальные
	// справочники; счётчики нужны сервису, чтобы запретить удаление.
	userID := f.user(ctx, pool, "mod-"+realmID.String()[:8])
	ticketID := f.ticket(ctx, pool, realmID, groupID, userID, userID)
	// GetByID делает JOIN категории и площадки — фикстура f.ticket их не
	// подставляет, поэтому заполняем обе связи перед чтением.
	_, err = pool.Exec(ctx, `UPDATE tickets SET category_modifier_id = $1, site_id = $2, category_id = $3 WHERE id = $4`,
		dto.ID, siteID, categoryID, ticketID)
	require.NoError(t, err)

	ticket, err := NewTicketRepo(pool, nil).GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	require.NoError(t, err)
	require.NotNil(t, ticket.CategoryModifier)
	assert.Equal(t, *dto.ID, ticket.CategoryModifier.ID)
	assert.Equal(t, name, ticket.CategoryModifier.Name)

	count, err := repo.CountTickets(ctx, *dto.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	count, err = repo.CountTicketsByCategory(ctx, categoryID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Обнуление nullable-полей: унаследованное значение снимается вручную.
	require.NoError(t, repo.Update(ctx, &models.CategoryModifierDTO{
		ID: dto.ID, RealmID: realmID, CategoryID: categoryID, Name: name,
		IsActive: false,
	}))
	got, err = repo.GetByID(ctx, &models.GetCategoryModifierByIDDTO{ID: *dto.ID, RealmID: realmID})
	require.NoError(t, err)
	assert.Empty(t, got.Keywords, "непереданные ключи пишутся пустым массивом, а не NULL")
	assert.Nil(t, got.Priority)
	assert.Nil(t, got.DefaultDueDays)
	assert.False(t, got.IsActive)
}
