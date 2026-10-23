package services

import (
	"context"
	"strings"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// fakeImportUsersRepo — заглушка справочника пользователей: импорту нужен
// список пользователей реалма, чтобы найти получателя по ФИО.
type fakeImportUsersRepo struct {
	repository.Users
	users []*models.UserData
}

func (f *fakeImportUsersRepo) GetAll(_ context.Context, _ *uuid.UUID) ([]*models.UserData, error) {
	return f.users, nil
}

// fakeImportSitesRepo — заглушка справочника площадок: в файле площадка
// приходит названием.
type fakeImportSitesRepo struct {
	repository.Sites
	sites []*models.Site
}

func (f *fakeImportSitesRepo) Get(_ context.Context, _ *models.GetSitesDTO) ([]*models.Site, error) {
	return f.sites, nil
}

// failingCreateRepo — репозиторий, который отказывает на вставке: так
// проверяется, что ошибка одной строки не отменяет импорт остальных.
type failingCreateRepo struct {
	*fakeEquipmentRepo
	err error
}

func (f *failingCreateRepo) Create(_ context.Context, _ postgres.Tx, _ *models.EquipmentDTO) (*models.Equipment, error) {
	return nil, f.err
}

// GetTypes отдаёт справочник типов: импорту нужен весь список реалма, а не
// один тип по идентификатору.
func (f *fakeEquipmentRepo) GetTypes(_ context.Context, _ *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error) {
	if f.eqType == nil {
		return nil, nil
	}
	return []*models.EquipmentType{f.eqType}, nil
}

// newImportTestService собирает сервис импорта над настоящим сервисом
// оборудования: единицы создаются обычным путём, поэтому правила сервиса
// (вывод статуса из держателя, генерация номера) действуют и при импорте.
func newImportTestService(t *testing.T, repo repository.Equipment, users []*models.UserData, sites []*models.Site) *EquipmentImportService {
	t.Helper()
	equipment := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
	return NewEquipmentImportService(equipment, &fakeImportSitesRepo{sites: sites}, &fakeImportUsersRepo{users: users})
}

// importTestRefs — типичный реалм: тип «Ноутбук», площадка и сотрудник.
type importTestRefs struct {
	realmID   uuid.UUID
	typeID    uuid.UUID
	siteID    uuid.UUID
	userID    uuid.UUID
	equipment *fakeEquipmentRepo
	users     []*models.UserData
	sites     []*models.Site
}

func newImportTestRefs(t *testing.T) *importTestRefs {
	t.Helper()
	refs := &importTestRefs{
		realmID: uuid.New(),
		typeID:  uuid.New(),
		siteID:  uuid.New(),
		userID:  uuid.New(),
		equipment: &fakeEquipmentRepo{
			eqType: &models.EquipmentType{ID: uuid.New(), RealmID: uuid.New(), Name: "Ноутбук", Kind: models.EquipmentTypeDevice},
		},
		users: []*models.UserData{{
			ID: uuid.New(), Username: "ivanov", FirstName: "Иван", LastName: "Иванов", IsActive: true,
		}},
		sites: []*models.Site{{ID: uuid.New(), Name: "Офис на Мира 10"}},
	}
	refs.equipment.eqType.RealmID = refs.realmID
	refs.users[0].ID = refs.userID
	refs.sites[0].ID = refs.siteID
	return refs
}

// TestEquipmentImport_CreatesUnitWithHolder проверяет главный сценарий: строка
// файла с получателем-сотрудником создаёт единицу со ссылкой на пользователя.
func TestEquipmentImport_CreatesUnitWithHolder(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Получатель;Площадка;Локация;Дата покупки;Серийный номер;Производитель",
		"Ноутбук Dell Latitude;Ноутбук;Иванов Иван;Офис на Мира 10;Кабинет 312;15.01.2026;ABC123;Dell",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{
		RealmID: refs.realmID,
		File:    strings.NewReader(csv),
		Actor:   &models.Actor{ID: uuid.New()},
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Failed)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	assert.Equal(t, 2, result.Rows[0].Row)
	assert.True(t, result.Rows[0].HolderFound)

	require.NotNil(t, refs.equipment.created)
	assert.Equal(t, "Ноутбук Dell Latitude", refs.equipment.created.Name)
	// Держатель найден среди пользователей — это ссылка, а не свободный текст.
	require.NotNil(t, refs.equipment.created.HolderUserID)
	assert.Equal(t, refs.userID, *refs.equipment.created.HolderUserID)
	assert.Nil(t, refs.equipment.created.HolderName)
	require.NotNil(t, refs.equipment.created.SiteID)
	assert.Equal(t, refs.siteID, *refs.equipment.created.SiteID)
	require.NotNil(t, refs.equipment.created.PurchaseDate)
	assert.Equal(t, "2026-01-15", *refs.equipment.created.PurchaseDate)
	require.NotNil(t, refs.equipment.created.SerialNumber)
	assert.Equal(t, "ABC123", *refs.equipment.created.SerialNumber)
	// Держатель есть — статус выведен как «в эксплуатации».
	assert.Equal(t, models.EquipmentStatusInUse, refs.equipment.created.Status)
}

// TestEquipmentImport_UnknownHolderBecomesFreeValue — получатель не найден
// среди пользователей: ФИО записывается свободным значением (holder_name).
func TestEquipmentImport_UnknownHolderBecomesFreeValue(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Получатель",
		"Принтер HP;Ноутбук;Петров Пётр",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	assert.False(t, result.Rows[0].HolderFound)
	assert.Contains(t, result.Rows[0].Note, "не найден среди пользователей")

	require.NotNil(t, refs.equipment.created)
	assert.Nil(t, refs.equipment.created.HolderUserID)
	require.NotNil(t, refs.equipment.created.HolderName)
	assert.Equal(t, "Петров Пётр", *refs.equipment.created.HolderName)
	// Свободное значение — тоже держатель, статус выводится из него.
	assert.Equal(t, models.EquipmentStatusInUse, refs.equipment.created.Status)
}

// TestEquipmentImport_UnknownTypeFailsRow — неизвестный тип не создаётся:
// вид типа (устройство/комплектующее/…) из названия не выводится.
func TestEquipmentImport_UnknownTypeFailsRow(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Получатель",
		"Принтер HP;Принтер;Петров Пётр",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Rows, 1)
	assert.Contains(t, result.Rows[0].Error, "не найден в справочнике")
	assert.Nil(t, refs.equipment.created)
}

// TestEquipmentImport_UnknownSiteFailsRow — неизвестная площадка не создаётся
// молча: площадка — справочник, а не текст в карточке.
func TestEquipmentImport_UnknownSiteFailsRow(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка;Получатель",
		"Ноутбук;Ноутбук;Склад №5;Иванов Иван",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Rows, 1)
	assert.Contains(t, result.Rows[0].Error, "площадка")
	assert.Nil(t, refs.equipment.created)
}

// TestEquipmentImport_StatusInUseWithoutHolderFails — «в эксплуатации» без
// получателя противоречит правилу «выданная единица имеет держателя».
func TestEquipmentImport_StatusInUseWithoutHolderFails(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка;Статус",
		"Ноутбук;Ноутбук;Офис на Мира 10;в эксплуатации",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Rows, 1)
	assert.Contains(t, result.Rows[0].Error, "получатель не задан")
	assert.Nil(t, refs.equipment.created)
}

// TestEquipmentImport_ManualStatusKept — ручные статусы (ремонт, дефект,
// резерв, списание) не выводятся из держателя и переносятся как есть.
func TestEquipmentImport_ManualStatusKept(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка;Статус",
		"Ноутбук;Ноутбук;Офис на Мира 10;ремонт",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	require.NotNil(t, refs.equipment.created)
	assert.Equal(t, models.EquipmentStatusRepair, refs.equipment.created.Status)
}

// TestEquipmentImport_UnknownColumnBecomesSpec — колонка, которая не стала
// полем, не теряется: характеристики — свободный словарь.
func TestEquipmentImport_UnknownColumnBecomesSpec(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка;Цвет;№ п/п",
		"Ноутбук;Ноутбук;Офис на Мира 10;серый;1",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	require.NotNil(t, refs.equipment.created)

	specs := map[string]string{}
	require.NoError(t, json.Unmarshal(refs.equipment.created.Specs, &specs))
	assert.Equal(t, "серый", specs["Цвет"])
	// Служебная колонка в характеристики не попадает.
	assert.NotContains(t, specs, "№ п/п")
}

// TestEquipmentImport_RowErrorDoesNotStopImport — ошибка одной строки не
// отменяет импорт остальных: отчёт по строкам показывает, что не получилось.
func TestEquipmentImport_RowErrorDoesNotStopImport(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	failing := &failingCreateRepo{fakeEquipmentRepo: refs.equipment, err: models.ErrEquipmentSerialExists}
	svc := newImportTestService(t, failing, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка",
		"Первый;Ноутбук;Офис на Мира 10",
		"Второй;Ноутбук;Офис на Мира 10",
	}, "\n")

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{RealmID: refs.realmID, File: strings.NewReader(csv)})
	require.NoError(t, err)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 2, result.Failed)
	require.Len(t, result.Rows, 2)
	assert.Contains(t, result.Rows[0].Error, "серийным номером")
}

// TestEquipmentImport_ReadsXLSX проверяет разбор книги Excel: даты в ячейках
// с датой читаются, а получатель ищется среди пользователей.
func TestEquipmentImport_ReadsXLSX(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	require.NoError(t, file.SetSheetRow(sheet, "A1", &[]any{
		"Наименование", "Тип", "Получатель", "Площадка", "Дата покупки",
	}))
	require.NoError(t, file.SetSheetRow(sheet, "A2", &[]any{
		"Ноутбук Dell", "Ноутбук", "Иванов Иван", "Офис на Мира 10", "2026-01-15",
	}))

	buf, err := file.WriteToBuffer()
	require.NoError(t, err)

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{
		RealmID:  refs.realmID,
		FileName: "units.xlsx",
		File:     buf,
	})
	require.NoError(t, err)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	require.NotNil(t, refs.equipment.created)
	require.NotNil(t, refs.equipment.created.HolderUserID)
	assert.Equal(t, refs.userID, *refs.equipment.created.HolderUserID)
	require.NotNil(t, refs.equipment.created.PurchaseDate)
	assert.Equal(t, "2026-01-15", *refs.equipment.created.PurchaseDate)
}

// TestEquipmentImport_ReadsWindows1251CSV — русский Excel сохраняет csv в
// windows-1251: такой файл тоже должен прочитаться.
func TestEquipmentImport_ReadsWindows1251CSV(t *testing.T) {
	ctx := context.Background()
	refs := newImportTestRefs(t)
	svc := newImportTestService(t, refs.equipment, refs.users, refs.sites)

	csv := strings.Join([]string{
		"Наименование;Тип;Площадка",
		"Ноутбук;Ноутбук;Офис на Мира 10",
	}, "\n")
	encoded, _, err := transform.Bytes(charmap.Windows1251.NewEncoder(), []byte(csv))
	require.NoError(t, err)

	result, err := svc.Import(ctx, &models.EquipmentImportDTO{
		RealmID: refs.realmID,
		File:    strings.NewReader(string(encoded)),
	})
	require.NoError(t, err)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, models.EquipmentImportRowCreated, result.Rows[0].Status)
	assert.Equal(t, "Ноутбук", refs.equipment.created.Name)
}

// TestMapImportColumns_RequiresName — без колонки «Наименование» импорту
// нечего заводить: это отказ всего файла, а не строки.
func TestMapImportColumns_RequiresName(t *testing.T) {
	_, err := mapImportColumns([]string{"Тип", "Получатель"})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
	assert.Contains(t, err.Error(), "Наименование")
}

// TestMapImportColumns_Aliases — синонимы заголовков: «Серийный №» и
// «серийный номер» — одна колонка, а «№ п/п» в характеристики не идёт.
func TestMapImportColumns_Aliases(t *testing.T) {
	columns, err := mapImportColumns([]string{"Наименование", "Серийный №", "Инвентарный номер", "Получатель (ФИО)"})
	require.NoError(t, err)

	_, ok := columns.byKey[importColSerial]
	assert.True(t, ok)
	_, ok = columns.byKey[importColInventory]
	assert.True(t, ok)
	_, ok = columns.byKey[importColHolder]
	assert.True(t, ok)
}

// TestParseImportDate — даты читаются в привычном русском формате (сначала
// день, потом месяц) и в ISO.
func TestParseImportDate(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{text: "15.01.2026", want: "2026-01-15"},
		{text: "2026-01-15", want: "2026-01-15"},
		{text: "15/01/2026", want: "2026-01-15"},
		{text: "15.01.26", want: "2026-01-15"},
	}
	for _, tt := range cases {
		got, ok := parseImportDate(tt.text, "")
		require.True(t, ok, tt.text)
		assert.Equal(t, tt.want, got)
	}

	_, ok := parseImportDate("не дата", "")
	assert.False(t, ok)
}
