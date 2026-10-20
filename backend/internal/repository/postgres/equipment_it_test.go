package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Заготовка окружения для интеграционных тестов оборудования.
type eqFixture struct {
	realmID uuid.UUID
	userID  uuid.UUID
	typeID  uuid.UUID
	kitType uuid.UUID
}

func newEqFixture(t *testing.T, pool *pgxpool.Pool) *eqFixture {
	t.Helper()
	ctx := context.Background()
	f := &eqFixture{realmID: uuid.New(), userID: uuid.New(), typeID: uuid.New(), kitType: uuid.New()}

	_, err := pool.Exec(ctx, `INSERT INTO realms (id, code, name) VALUES ($1, $2, $3)`,
		f.realmID, "eqtest-"+f.realmID.String()[:8], "Equipment Test Realm")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO users (id, username, first_name, last_name) VALUES ($1, $2, $3, $4)`,
		f.userID, "eqtestuser", "Тест", "Тестов")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO equipment_types (id, realm_id, name, kind) VALUES ($1, $2, 'Ноутбук', 'device')`,
		f.typeID, f.realmID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO equipment_types (id, realm_id, name, kind) VALUES ($1, $2, 'RAM', 'component')`,
		f.kitType, f.realmID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM realms WHERE id = $1`, f.realmID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, f.userID)
	})
	return f
}

func eqString(s string) *string { return &s }

func newEquipmentDTO(f *eqFixture, typeID *uuid.UUID, name string) *models.EquipmentDTO {
	return &models.EquipmentDTO{
		RealmID:      f.realmID,
		TypeID:       typeID,
		Name:         name,
		Status:       models.EquipmentStatusInStock,
		Location:     eqString("Склад"),
		CreatedBy:    &f.userID,
		Specs:        []byte("{}"),
		Vendor:       "",
		Model:        "",
		HolderName:   nil,
		SerialNumber: nil,
	}
}

func TestEquipmentRepo_Stage3Flow(t *testing.T) {
	pool := itPool(t)
	f := newEqFixture(t, pool)
	ctx := context.Background()

	repo := NewEquipmentRepo(pool, NewTransactionRepo(pool))

	// Комплект — без типа: у комплекта типа нет (`type_id` NULL), а `Create`
	// репозитория это не проверяет (правило — в сервисе). Так этот прогон заодно
	// проверяет чтение типового NULL через `LEFT JOIN`.
	kit, err := repo.Create(ctx, nil, newEquipmentDTO(f, nil, "Комплект A"))
	require.NoError(t, err)

	// Помечаем как комплект напрямую (Create не выставляет is_kit).
	_, err = pool.Exec(ctx, `UPDATE equipment SET is_kit = true WHERE id = $1`, kit.ID)
	require.NoError(t, err)

	// Чтение комплекта не должно ни падать на NULL-типе, ни подставлять чужой тип:
	// name/kind пустые, id типа — NULL.
	fetched, err := repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: kit.ID, RealmID: f.realmID})
	require.NoError(t, err)
	assert.Nil(t, fetched.TypeID)
	assert.Empty(t, fetched.TypeName)
	assert.Empty(t, fetched.TypeKind)
	assert.True(t, fetched.IsKit)

	comp, err := repo.Create(ctx, nil, newEquipmentDTO(f, &f.kitType, "Модуль памяти"))
	require.NoError(t, err)

	// addComposition + setHolderAndStatus + event в одной транзакции.
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.OpenComposition(ctx, tx, &models.KitComposition{
		KitID: kit.ID, ComponentID: comp.ID, StartedAt: time.Now(), StartedBy: &f.userID, Reason: "сборка",
	}))
	require.NoError(t, repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
		EquipmentID: kit.ID, EventType: models.EquipmentEventComponentAdded, OperationID: uuid.New(),
		ComponentID: &comp.ID, UserID: &f.userID,
	}))
	require.NoError(t, tx.Commit(ctx))

	members, err := repo.GetOpenComposition(ctx, kit.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, comp.ID, members[0].ID)

	// Выдача.
	now := time.Now()
	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
		EquipmentID: kit.ID, UserID: &f.userID, AssignedAt: now, AssignedBy: &f.userID,
	}))
	require.NoError(t, repo.SetHolderAndStatus(ctx, tx, kit.ID, &f.userID, nil, nil, nil, models.EquipmentStatusInUse))
	require.NoError(t, tx.Commit(ctx))

	open, err := repo.GetOpenAssignment(ctx, kit.ID)
	require.NoError(t, err)
	assert.Equal(t, f.userID, *open.UserID)

	// Складского периода ещё не было: возврату нечего восстанавливать, и «дом»
	// возвращает ErrNotFound (сервис тогда берёт текущее место единицы).
	_, _, err = repo.GetLastStoredPlace(ctx, kit.ID)
	assert.ErrorIs(t, err, models.ErrNotFound)

	// Возврат: выдача закрывается, открывается строка без держателя — это «дом»
	// единицы, куда её поставит следующий возврат.
	siteID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO sites (id, name) VALUES ($1, $2)`, siteID, "ППРЗ")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sites WHERE id = $1`, siteID) })

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.CloseAssignment(ctx, tx, open.ID, now, ""))
	require.NoError(t, repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
		EquipmentID: kit.ID, SiteID: &siteID, Location: eqString("Склад, полка A"),
		AssignedAt: now.Add(time.Second), AssignedBy: &f.userID,
	}))
	require.NoError(t, tx.Commit(ctx))

	storedSite, storedLocation, err := repo.GetLastStoredPlace(ctx, kit.ID)
	require.NoError(t, err)
	require.NotNil(t, storedSite)
	assert.Equal(t, siteID, *storedSite)
	require.NotNil(t, storedLocation)
	assert.Equal(t, "Склад, полка A", *storedLocation)

	history, err := repo.GetAssignmentHistory(ctx, f.realmID, kit.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	// Свежий период — складской, держателя в нём нет; подписи ссылок в прошлой
	// выдаче резолвит SQL: клиент не тянет справочник пользователей ради строки
	// истории.
	assert.Empty(t, history[0].HolderFullName)
	assert.Equal(t, "ППРЗ", history[0].SiteName)
	assert.Equal(t, "Тестов Тест", history[1].HolderFullName)
	assert.Equal(t, "Тестов Тест", history[1].AssignedByName)

	events, err := repo.GetEvents(ctx, f.realmID, kit.ID, 100, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, models.EquipmentEventComponentAdded, events[0].EventType)
	assert.Equal(t, "Тестов Тест", events[0].UserName)

	// Резолв подписей для снимка события: в запросе по id обязателен алиас
	// таблицы users — без него Postgres отвечает 42P01 «missing FROM-clause
	// entry for table "u"», и выдача падает с 500 (регресс, из-за которого
	// GetUserFullName собирал выражение по алиасу, а сам алиаса не объявлял).
	name, err := repo.GetUserFullName(ctx, f.userID)
	require.NoError(t, err)
	assert.Equal(t, "Тестов Тест", name)

	siteName, err := repo.GetSiteName(ctx, siteID)
	require.NoError(t, err)
	assert.Equal(t, "ППРЗ", siteName)

	// Акт.
	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	seq, err := repo.NextActSeq(ctx, tx, f.realmID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, seq, 1)
	require.NoError(t, repo.CreateAct(ctx, tx, &models.EquipmentAct{
		RealmID: f.realmID, Number: "АКТ-2026-0001", KitEquipmentID: &kit.ID,
		RecipientUserID: &f.userID, IssuedBy: &f.userID, IssuedAt: now,
		Content: []byte(`[{"name":"Комплект A"}]`),
	}))
	require.NoError(t, tx.Commit(ctx))

	acts, total, err := repo.GetActs(ctx, &models.EquipmentActsFilter{RealmID: f.realmID, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, acts, 1)
	act, err := repo.GetActByID(ctx, &models.GetEquipmentActByIDDTO{ID: acts[0].ID, RealmID: f.realmID})
	require.NoError(t, err)
	assert.Equal(t, "АКТ-2026-0001", act.Number)

	// Закрытие состава.
	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.CloseOpenByComponent(ctx, tx, comp.ID, &f.userID, "разборка", nil))
	require.NoError(t, tx.Commit(ctx))

	members, err = repo.GetOpenComposition(ctx, kit.ID)
	require.NoError(t, err)
	assert.Empty(t, members)

	compHistory, err := repo.GetCompositionHistory(ctx, kit.ID)
	require.NoError(t, err)
	require.Len(t, compHistory, 1)
	require.NotNil(t, compHistory[0].EndedAt)
	assert.Equal(t, "Модуль памяти", compHistory[0].ComponentName)
	assert.Equal(t, "Тестов Тест", compHistory[0].StartedByName)
	assert.Equal(t, "Тестов Тест", compHistory[0].EndedByName)
}

// TestEquipmentRepo_InventoryNumberCounter проверяет счётчик инвентарных номеров
// на реальном upsert: серии независимы, а откат транзакции возвращает счётчик
// назад — номер не «сгорает» при неудачном создании.
func TestEquipmentRepo_InventoryNumberCounter(t *testing.T) {
	pool := itPool(t)
	f := newEqFixture(t, pool)
	ctx := context.Background()

	repo := NewEquipmentRepo(pool, NewTransactionRepo(pool))

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	first, err := repo.NextInventoryNumber(ctx, tx, f.realmID, models.InventorySeriesUnit)
	require.NoError(t, err)
	second, err := repo.NextInventoryNumber(ctx, tx, f.realmID, models.InventorySeriesUnit)
	require.NoError(t, err)
	kit, err := repo.NextInventoryNumber(ctx, tx, f.realmID, models.InventorySeriesKit)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
	assert.Equal(t, 1, kit, "у серии комплектов свой счётчик")

	// Откат не «сжигает» номер: счётчик откатывается вместе с транзакцией.
	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	_, err = repo.NextInventoryNumber(ctx, tx, f.realmID, models.InventorySeriesUnit)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	third, err := repo.NextInventoryNumber(ctx, tx, f.realmID, models.InventorySeriesUnit)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	assert.Equal(t, 3, third, "после отката следующий номер — 3, а не 4")
}

// TestEquipmentRepo_InventoryNumberExists проверяет проверку занятости номера:
// сервис опирается на неё, чтобы пропустить значение, занятое ручным вводом.
func TestEquipmentRepo_InventoryNumberExists(t *testing.T) {
	pool := itPool(t)
	f := newEqFixture(t, pool)
	ctx := context.Background()

	repo := NewEquipmentRepo(pool, NewTransactionRepo(pool))

	exists, err := repo.InventoryNumberExists(ctx, nil, f.realmID, "000001")
	require.NoError(t, err)
	assert.False(t, exists)

	dto := newEquipmentDTO(f, &f.typeID, "Ноутбук с номером")
	dto.InventoryNumber = eqString("000001")
	_, err = repo.Create(ctx, nil, dto)
	require.NoError(t, err)

	exists, err = repo.InventoryNumberExists(ctx, nil, f.realmID, "000001")
	require.NoError(t, err)
	assert.True(t, exists)

	// Номер уникален в пределах реалма: в другом реалме то же значение свободно.
	other := newEqFixture(t, pool)
	exists, err = repo.InventoryNumberExists(ctx, nil, other.realmID, "000001")
	require.NoError(t, err)
	assert.False(t, exists, "уникальность номера — в пределах реалма")
}
