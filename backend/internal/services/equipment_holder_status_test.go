package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentService_DeriveHolderStatus закрепляет инвариант: статусы
// «на складе»/«в эксплуатации» выводятся из держателя, а не принимаются из формы.
//
// Регресс, ради которого правило заведено: форма правки присылает статус и
// держателя независимо, поэтому сохранённая единица могла остаться «в
// эксплуатации» без держателя, а блок «Держатель» показывал на неё «На складе»
// (hasHolder считался по держателю, а не по статусу). Проверяются оба входа —
// Create и Update: оба идут через deriveHolderStatus.
func TestEquipmentService_DeriveHolderStatus(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, holderID := uuid.New(), uuid.New(), uuid.New()
	holderName := "Внешний сотрудник"
	emptyName := ""

	cases := []struct {
		name       string
		status     models.EquipmentStatus
		holderID   *uuid.UUID
		holderName *string
		want       models.EquipmentStatus
		stored     models.EquipmentStatus
	}{
		{
			name:   "эксплуатация без держателя -> на складе",
			status: models.EquipmentStatusInUse, want: models.EquipmentStatusInStock,
		},
		{
			name:   "склад с держателем-пользователем -> в эксплуатации",
			status: models.EquipmentStatusInStock, holderID: &holderID, want: models.EquipmentStatusInUse,
		},
		{
			name:   "склад с держателем-ФИО -> в эксплуатации",
			status: models.EquipmentStatusInStock, holderName: &holderName, want: models.EquipmentStatusInUse,
		},
		{
			name:   "эксплуатация с держателем остаётся",
			status: models.EquipmentStatusInUse, holderID: &holderID, want: models.EquipmentStatusInUse,
		},
		{
			name:   "склад без держателя остаётся",
			status: models.EquipmentStatusInStock, want: models.EquipmentStatusInStock,
		},
		{
			// Пустая строка — «держателя нет»: нормализация формы шлёт её вместо null.
			name:   "пустое ФИО держателем не считается",
			status: models.EquipmentStatusInUse, holderName: &emptyName, want: models.EquipmentStatusInStock,
		},
		{
			// Ремонт/дефект/резерв/списание от держателя не зависят. В правке они
			// берутся из сохранённого состояния, а не из тела: форма их не присылает,
			// а API не должен позволять переписать статус в обход UI.
			name:   "ремонт с держателем не трогаем",
			status: models.EquipmentStatusRepair, holderID: &holderID,
			want: models.EquipmentStatusRepair, stored: models.EquipmentStatusRepair,
		},
		{
			name:   "дефектная без держателя не трогаем",
			status: models.EquipmentStatusDefective,
			want:   models.EquipmentStatusDefective, stored: models.EquipmentStatusDefective,
		},
		{
			name:   "резерв без держателя не трогаем",
			status: models.EquipmentStatusReserved,
			want:   models.EquipmentStatusReserved, stored: models.EquipmentStatusReserved,
		},
		{
			name:   "списано с держателем не трогаем",
			status: models.EquipmentStatusWrittenOff, holderID: &holderID,
			want: models.EquipmentStatusWrittenOff, stored: models.EquipmentStatusWrittenOff,
		},
	}

	newRepo := func() *fakeEquipmentRepo {
		return &fakeEquipmentRepo{eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice}}
	}

	for _, tt := range cases {
		t.Run("create/"+tt.name, func(t *testing.T) {
			repo := newRepo()
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
				RealmID: realmID, TypeID: &typeID, Name: "Единица",
				Status: tt.status, HolderUserID: tt.holderID, HolderName: tt.holderName,
			}})
			require.NoError(t, err)
			require.NotNil(t, repo.created)
			assert.Equal(t, tt.want, repo.created.Status)
		})

		t.Run("update/"+tt.name, func(t *testing.T) {
			repo := newRepo()
			// Правка статус из тела не берёт: сохранённое состояние лежит в
			// репозитории, а запрос лишь повторяет то, что прислала форма.
			id := uuid.New()
			stored := tt.stored
			if stored == "" {
				stored = models.EquipmentStatusInStock
			}
			repo.parent = &models.Equipment{
				ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice,
				Status: stored,
			}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			_, err := svc.Update(ctx, &models.EquipmentDTO{
				ID: &id, RealmID: realmID, TypeID: &typeID, Name: "Единица",
				Status: tt.status, HolderUserID: tt.holderID, HolderName: tt.holderName,
			})
			require.NoError(t, err)
			require.NotNil(t, repo.updated)
			assert.Equal(t, tt.want, repo.updated.Status)
		})
	}
}

// TestEquipmentService_UpdateKeepsStatus закрепляет правило: правка статус не
// меняет. Ручные значения (дефект, списание, ремонт, резерв) берутся из
// сохранённого состояния, а не из тела запроса — форма поле не присылает, и API
// не должен позволять переписать статус в обход UI. Выводимая пара
// (in_stock/in_use) по-прежнему считается от держателя.
//
// Регресс, ради которого правило заведено: через PUT можно было выставить
// «Дефектное»/«Списано», хотя отдельного механизма смены статуса нет.
func TestEquipmentService_UpdateKeepsStatus(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, holderID := uuid.New(), uuid.New(), uuid.New()

	cases := []struct {
		name   string
		stored models.EquipmentStatus
		body   models.EquipmentStatus
		holder *uuid.UUID
		want   models.EquipmentStatus
	}{
		{
			// Дефектную правкой не вернуть в строй: тело просит «на складе», а
			// сохранённый ручной статус остаётся.
			name:   "ручной статус не перезаписывается телом",
			stored: models.EquipmentStatusDefective, body: models.EquipmentStatusInStock,
			want: models.EquipmentStatusDefective,
		},
		{
			name:   "списание телом не снимается",
			stored: models.EquipmentStatusWrittenOff, body: models.EquipmentStatusInStock,
			holder: &holderID, want: models.EquipmentStatusWrittenOff,
		},
		{
			name:   "тело не ставит ремонт",
			stored: models.EquipmentStatusInStock, body: models.EquipmentStatusRepair,
			want: models.EquipmentStatusInStock,
		},
		{
			// Пары держателя-статуса остаются выводимыми: сохранённый «в
			// эксплуатации» без держателя в теле становится «на складе».
			name:   "выводимый статус считается от держателя",
			stored: models.EquipmentStatusInUse, body: models.EquipmentStatusInUse,
			want: models.EquipmentStatusInStock,
		},
		{
			name:   "выдача держателя делает эксплуатацию",
			stored: models.EquipmentStatusInStock, body: models.EquipmentStatusInStock,
			holder: &holderID, want: models.EquipmentStatusInUse,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeEquipmentRepo{eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice}}
			id := uuid.New()
			repo.parent = &models.Equipment{
				ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice,
				Status: tt.stored,
			}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			_, err := svc.Update(ctx, &models.EquipmentDTO{
				ID: &id, RealmID: realmID, TypeID: &typeID, Name: "Единица",
				Status: tt.body, HolderUserID: tt.holder,
			})
			require.NoError(t, err)
			require.NotNil(t, repo.updated)
			assert.Equal(t, tt.want, repo.updated.Status)
		})
	}
}
