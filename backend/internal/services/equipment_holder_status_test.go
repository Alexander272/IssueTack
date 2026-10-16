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
			// Ремонт/дефект/резерв/списание от держателя не зависят.
			name:   "ремонт с держателем не трогаем",
			status: models.EquipmentStatusRepair, holderID: &holderID, want: models.EquipmentStatusRepair,
		},
		{
			name:   "дефектная без держателя не трогаем",
			status: models.EquipmentStatusDefective, want: models.EquipmentStatusDefective,
		},
		{
			name:   "резерв без держателя не трогаем",
			status: models.EquipmentStatusReserved, want: models.EquipmentStatusReserved,
		},
		{
			name:   "списано с держателем не трогаем",
			status: models.EquipmentStatusWrittenOff, holderID: &holderID, want: models.EquipmentStatusWrittenOff,
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
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			id := uuid.New()
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
