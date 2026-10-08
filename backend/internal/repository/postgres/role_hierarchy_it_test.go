package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/migrate"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Полноценные SQL-тесты на реальной PostgreSQL. Гейт — TEST_DATABASE_URL:
// без переменной окружения тесты пропускаются, прогон без БД не падает.
// Миграции накатываются автоматически (goose.Up), поэтому БД достаточно пустой.

type itFixture struct {
	realmIDs []uuid.UUID
	permIDs  []uuid.UUID
}

func newItFixture() *itFixture {
	return &itFixture{}
}

func itPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, migrate.Migrate(pool))
	return pool
}

func (f *itFixture) cleanup(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if len(f.permIDs) > 0 {
		// Удаляем только «наши» пермишены, которыми ещё никто не пользуется:
		// если строка существовала до теста (общие данные), не трогаем её.
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			DELETE FROM %s p
			WHERE p.id = ANY($1)
			  AND NOT EXISTS (SELECT 1 FROM %s rp WHERE rp.permission_id = p.id)`,
			Tables.Permissions, Tables.RolePermissions), f.permIDs)
		require.NoError(t, err)
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1)`, Tables.Realms), f.realmIDs)
	require.NoError(t, err)
}

// slug строит уникальный идентификатор роли: метод GetRolesInheritance и сам SQL не
// фильтруют по реалму, поэтому одинаковые slug'и в разных реалмах (например, seed
// 'it' с ролями admin/user/chief) наслоились бы на фикстуры теста.
func (f *itFixture) slug(realmID uuid.UUID, name string) string {
	return name + "-" + realmID.String()[:8]
}

func (f *itFixture) realm(ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, code, name, description, is_active)
		VALUES ($1, $2, $3, '', true)`,
		Tables.Realms), id, "test-"+id.String(), "Тест-реалм "+id.String())
	if err != nil {
		panic(err)
	}
	f.realmIDs = append(f.realmIDs, id)
	return id
}

func (f *itFixture) role(ctx context.Context, pool *pgxpool.Pool, slug string, realmID uuid.UUID) uuid.UUID {
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, slug, name, realm_id, description, level, is_active, is_system, is_editable)
		VALUES ($1, $2, $3, $4, '', 1, true, false, true)`,
		Tables.Roles), id, slug, "Роль "+slug, realmID)
	if err != nil {
		panic(err)
	}
	return id
}

func (f *itFixture) grant(ctx context.Context, pool *pgxpool.Pool, roleID uuid.UUID, object, action string) {
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (object, action) VALUES ($1, $2) ON CONFLICT (object, action) DO NOTHING`,
		Tables.Permissions), object, action)
	if err != nil {
		panic(err)
	}
	var permID uuid.UUID
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT id FROM %s WHERE object = $1 AND action = $2`,
		Tables.Permissions), object, action).Scan(&permID)
	if err != nil {
		panic(err)
	}
	f.permIDs = append(f.permIDs, permID)
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		Tables.RolePermissions), roleID, permID)
	if err != nil {
		panic(err)
	}
}

func (f *itFixture) user(ctx context.Context, pool *pgxpool.Pool, username string, active ...bool) uuid.UUID {
	id := uuid.New()
	isActive := true
	if len(active) > 0 {
		isActive = active[0]
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, username, first_name, last_name, email, is_active)
		VALUES ($1, $2, '', '', '', $3)`,
		Tables.Users), id, username, isActive)
	if err != nil {
		panic(err)
	}
	return id
}

func (f *itFixture) joinRealm(ctx context.Context, pool *pgxpool.Pool, userID, realmID, roleID uuid.UUID, isActive ...bool) {
	active := true
	if len(isActive) > 0 {
		active = isActive[0]
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (user_id, realm_id, role_id, is_active) VALUES ($1, $2, $3, $4)`,
		Tables.UserRealms), userID, realmID, roleID, active)
	if err != nil {
		panic(err)
	}
}

func (f *itFixture) group(ctx context.Context, pool *pgxpool.Pool, realmID uuid.UUID, name string) uuid.UUID {
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, realm_id, name, description) VALUES ($1, $2, $3, '')`,
		Tables.Groups), id, realmID, name)
	if err != nil {
		panic(err)
	}
	return id
}

func (f *itFixture) setGroupManager(ctx context.Context, pool *pgxpool.Pool, groupID, managerID uuid.UUID) {
	_, err := pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET manager_id = $1 WHERE id = $2`, Tables.Groups), managerID, groupID)
	if err != nil {
		panic(err)
	}
}

func (f *itFixture) joinGroup(ctx context.Context, pool *pgxpool.Pool, groupID, userID uuid.UUID) {
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (group_id, user_id) VALUES ($1, $2)`,
		Tables.GroupMembers), groupID, userID)
	if err != nil {
		panic(err)
	}
}

func (f *itFixture) category(ctx context.Context, pool *pgxpool.Pool, realmID uuid.UUID, name string, groupID *uuid.UUID) uuid.UUID {
	id := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, realm_id, name, description, group_id)
		VALUES ($1, $2, $3, '', $4)`,
		Tables.Categories), id, realmID, name, groupID)
	if err != nil {
		panic(err)
	}
	return id
}

// Тест направления наследования: запись (role_id=ребёнок, parent_role_id=родитель)
// означает «родитель включает ребёнка». GetInheritedRoles идёт к предкам (вверх),
// GetRoleDescendants/GetDirectChildren — к потомкам (вниз).
func TestRoleHierarchyRepo_InheritanceDirections_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	admin := f.role(ctx, pool, f.slug(realmID, "admin"), realmID)
	chief := f.role(ctx, pool, f.slug(realmID, "chief"), realmID)
	user := f.role(ctx, pool, f.slug(realmID, "user"), realmID)

	rh := NewRoleHierarchyRepo(pool, NewTransactionRepo(pool))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: user, ParentRoleID: chief, RealmID: realmID}))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: chief, ParentRoleID: admin, RealmID: realmID}))

	inherited, err := rh.GetInheritedRoles(ctx, &models.GetRolesInheritance{Roles: []string{f.slug(realmID, "user")}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{f.slug(realmID, "chief"), f.slug(realmID, "admin")}, inherited[f.slug(realmID, "user")])

	inheritedChief, err := rh.GetInheritedRoles(ctx, &models.GetRolesInheritance{Roles: []string{f.slug(realmID, "chief")}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{f.slug(realmID, "admin")}, inheritedChief[f.slug(realmID, "chief")])

	desc, err := rh.GetRoleDescendants(ctx, &models.GetRolesInheritance{Roles: []string{f.slug(realmID, "admin")}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{f.slug(realmID, "chief"), f.slug(realmID, "user")}, desc[f.slug(realmID, "admin")])

	direct, err := rh.GetDirectChildren(ctx, &models.GetRolesInheritance{Roles: []string{f.slug(realmID, "admin")}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{f.slug(realmID, "chief")}, direct[f.slug(realmID, "admin")])

	// У самой нижней роли потомков нет (пусто, а не nil — контракт «data всегда массив»).
	descUser, err := rh.GetRoleDescendants(ctx, &models.GetRolesInheritance{Roles: []string{f.slug(realmID, "user")}})
	require.NoError(t, err)
	assert.Empty(t, descUser[f.slug(realmID, "user")])

	// SyncRoleInheritance возвращает только прямых предков (родителей).
	synced, err := rh.SyncRoleInheritance(ctx, &models.GetRoleInheritance{Role: f.slug(realmID, "user"), Realm: realmID.String()})
	require.NoError(t, err)
	require.Len(t, synced, 1)
	assert.Equal(t, f.slug(realmID, "chief"), synced[0].ParentRole)
	assert.Equal(t, f.slug(realmID, "user"), synced[0].Role)
}

// LoadPolicy отдаёт связи в направлении Casbin: g(parent_role_id, role_id).
// Каждая запись — ровно одна пара родитель→потомок, без развёрнутых цепочек.
func TestRoleHierarchyRepo_LoadPolicy_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	admin := f.role(ctx, pool, f.slug(realmID, "admin"), realmID)
	chief := f.role(ctx, pool, f.slug(realmID, "chief"), realmID)
	user := f.role(ctx, pool, f.slug(realmID, "user"), realmID)

	rh := NewRoleHierarchyRepo(pool, NewTransactionRepo(pool))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: user, ParentRoleID: chief, RealmID: realmID}))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: chief, ParentRoleID: admin, RealmID: realmID}))

	policy, err := rh.LoadPolicy(ctx)
	require.NoError(t, err)

	got := map[string]string{}
	for _, p := range policy {
		if p.Realm == realmID.String() {
			got[p.Role] = p.ParentRole
		}
	}
	assert.Equal(t, map[string]string{
		f.slug(realmID, "user"):   f.slug(realmID, "chief"),
		f.slug(realmID, "chief"):  f.slug(realmID, "admin"),
	}, got)

	// Неактивная родительская роль выпадает из политики (Casbin её не получит).
	_, err = pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET is_active = false WHERE id = $1`, Tables.Roles), admin)
	require.NoError(t, err)
	policy, err = rh.LoadPolicy(ctx)
	require.NoError(t, err)
	got = map[string]string{}
	for _, p := range policy {
		if p.Realm == realmID.String() {
			got[p.Role] = p.ParentRole
		}
	}
	assert.Equal(t, map[string]string{f.slug(realmID, "user"): f.slug(realmID, "chief")}, got)
}

// GetRealmSupervisors: надзитель реалма — пользователь, чья роль (по цепочке ВНИЗ —
// через потомков, как в Casbin) имеет realm-wide право управления (category:write/site:write).
// Регресс: старый запрос шёл вверх по ролям-предкам, и роль user, которая являлась потомком
// admin, «цепляла» пермишены admin — начальниками становились все рядовые пользователи.
func TestGetRealmSupervisors_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	adminRole := f.role(ctx, pool, f.slug(realmID, "admin"), realmID)
	chiefRole := f.role(ctx, pool, f.slug(realmID, "chief"), realmID)
	userRole := f.role(ctx, pool, f.slug(realmID, "user"), realmID)

	rh := NewRoleHierarchyRepo(pool, NewTransactionRepo(pool))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: userRole, ParentRoleID: chiefRole, RealmID: realmID}))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: chiefRole, ParentRoleID: adminRole, RealmID: realmID}))

	// Право управления областью есть только у роли admin.
	f.grant(ctx, pool, adminRole, "category", "write")

	adminUser := f.user(ctx, pool, "admin-user")
	chiefUser := f.user(ctx, pool, "chief-user")
	plainUser := f.user(ctx, pool, "plain-user")

	f.joinRealm(ctx, pool, adminUser, realmID, adminRole)
	f.joinRealm(ctx, pool, chiefUser, realmID, chiefRole)
	f.joinRealm(ctx, pool, plainUser, realmID, userRole)

	ur := NewUserRealmRepo(pool, NewTransactionRepo(pool))

	supervisors, err := ur.GetRealmSupervisors(ctx, realmID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{adminUser}, supervisors)
	// Явно проверяем, что рядовой пользователь (потомок admin по иерархии) НЕ надзитель.
	assert.NotContains(t, supervisors, plainUser)

	// site:write тоже наделяет роль признаком надзителя.
	siteRole := f.role(ctx, pool, f.slug(realmID, "site-owner"), realmID)
	siteUser := f.user(ctx, pool, "site-user")
	f.grant(ctx, pool, siteRole, "site", "write")
	f.joinRealm(ctx, pool, siteUser, realmID, siteRole)

	// Неактивное членство в реалме не считается.
	inactiveUser := f.user(ctx, pool, "inactive-user")
	f.joinRealm(ctx, pool, inactiveUser, realmID, adminRole, false)

	supervisors, err = ur.GetRealmSupervisors(ctx, realmID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{adminUser, siteUser}, supervisors)

	// Чужой реалм с той же иерархией: рядовой пользователь там тоже не надзитель.
	otherRealm := f.realm(ctx, pool)
	otherAdmin := f.role(ctx, pool, f.slug(otherRealm, "admin"), otherRealm)
	otherChief := f.role(ctx, pool, f.slug(otherRealm, "chief"), otherRealm)
	otherUserRole := f.role(ctx, pool, f.slug(otherRealm, "user"), otherRealm)
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: otherUserRole, ParentRoleID: otherChief, RealmID: otherRealm}))
	require.NoError(t, rh.AddInheritance(ctx, nil, &models.RoleHierarchyDTO{RoleID: otherChief, ParentRoleID: otherAdmin, RealmID: otherRealm}))
	f.grant(ctx, pool, otherAdmin, "category", "write")
	otherPlain := f.user(ctx, pool, "other-plain")
	f.joinRealm(ctx, pool, otherPlain, otherRealm, otherUserRole)

	supervisorsOther, err := ur.GetRealmSupervisors(ctx, otherRealm)
	require.NoError(t, err)
	assert.Empty(t, supervisorsOther)

	// Неактивная роль пользователя исключает его из выборки (siteUser при этом остаётся).
	_, err = pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET is_active = false WHERE id = $1`, Tables.Roles), adminRole)
	require.NoError(t, err)
	supervisors, err = ur.GetRealmSupervisors(ctx, realmID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{siteUser}, supervisors)
}

// GetResponsibleByCategory теперь возвращает только менеджера группы-владельца категории
// (узкие «ответственные»), а не всех участников группы.
func TestNotificationRepo_GetResponsibleByCategory_IT(t *testing.T) {
	pool := itPool(t)
	ctx := context.Background()
	f := newItFixture()
	t.Cleanup(func() { f.cleanup(t, pool) })

	realmID := f.realm(ctx, pool)
	g1 := f.group(ctx, pool, realmID, "Группа 1")
	g2 := f.group(ctx, pool, realmID, "Группа 2")

	mgr := f.user(ctx, pool, "manager")
	inactiveMgr := f.user(ctx, pool, "inactive-mgr", false)
	member := f.user(ctx, pool, "member")

	f.setGroupManager(ctx, pool, g1, mgr)
	f.setGroupManager(ctx, pool, g2, inactiveMgr)
	f.joinGroup(ctx, pool, g1, member)

	catWithManager := f.category(ctx, pool, realmID, "Категория с менеджером", &g1)
	catNoManager := f.category(ctx, pool, realmID, "Категория без менеджера", nil)
	catInactiveMgr := f.category(ctx, pool, realmID, "Категория с неактивным менеджером", &g2)

	notif := NewNotificationRepo(pool, NewTransactionRepo(pool))

	resp, err := notif.GetResponsibleByCategory(ctx, catWithManager)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{mgr}, resp)
	// Рядовой участник группы не считается ответственным (сужение семантики).
	assert.NotContains(t, resp, member)

	resp, err = notif.GetResponsibleByCategory(ctx, catNoManager)
	require.NoError(t, err)
	assert.Empty(t, resp)

	resp, err = notif.GetResponsibleByCategory(ctx, catInactiveMgr)
	require.NoError(t, err)
	assert.Empty(t, resp)
}