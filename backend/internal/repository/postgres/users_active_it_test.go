package postgres

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Realm-выборки пользователей питают селекты заказчика/исполнителя (веб,
// диалог Mattermost и плагин), поэтому неактивные учётные записи и неактивное
// членство в realm не должны в них попадать. Глобальный список /users остаётся
// полным — администратор должен видеть отключённых пользователей.
func TestUserRepo_RealmListsSkipInactive_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	roleID := f.role(ctx, pool, f.slug(realmID, "user"), realmID)
	groupID := f.group(ctx, pool, realmID, f.slug(realmID, "executors"))

	suffix := realmID.String()
	active := f.user(ctx, pool, "active-"+suffix)
	groupless := f.user(ctx, pool, "groupless-"+suffix)
	inactiveUser := f.user(ctx, pool, "inactive-user-"+suffix, false)
	inactiveMember := f.user(ctx, pool, "inactive-member-"+suffix)

	f.joinRealm(ctx, pool, active, realmID, roleID)
	f.joinRealm(ctx, pool, groupless, realmID, roleID)
	f.joinRealm(ctx, pool, inactiveUser, realmID, roleID)
	f.joinRealm(ctx, pool, inactiveMember, realmID, roleID, false)
	f.joinGroup(ctx, pool, groupID, active)
	f.joinGroup(ctx, pool, groupID, inactiveUser)
	f.joinGroup(ctx, pool, groupID, inactiveMember)

	repo := NewUserRepo(pool, NewTransactionRepo(pool))

	ids := func(users []*models.UserData) []uuid.UUID {
		t.Helper()
		result := make([]uuid.UUID, 0, len(users))
		for _, u := range users {
			require.NotNil(t, u)
			result = append(result, u.ID)
		}
		return result
	}

	byRealm, err := repo.GetAll(ctx, &realmID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{active, groupless}, ids(byRealm))

	allMembers, err := repo.GetByMembership(ctx, realmID, models.MembershipAll)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{active, groupless}, ids(allMembers))

	executors, err := repo.GetByMembership(ctx, realmID, models.MembershipExecutors)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{active}, ids(executors))

	customers, err := repo.GetByMembership(ctx, realmID, models.MembershipCustomers)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{groupless}, ids(customers))

	global, err := repo.GetAll(ctx, nil)
	require.NoError(t, err)
	globalIDs := ids(global)
	assert.Contains(t, globalIDs, inactiveUser)
	assert.Contains(t, globalIDs, inactiveMember)
}
