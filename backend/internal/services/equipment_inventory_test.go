package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentService_InventoryNumber закрепляет автоинвентарные номера:
// единица получает только цифры («000001»), комплект — ту же серию с префиксом
// «KIT-» и свой независимый счётчик. Номер генерируется, когда поле пустое, и не
// перегенерируется, когда пришёл руками. Ручной номер, занявший будущее значение
// счётчика, сдвигает генерацию дальше.
func TestEquipmentService_InventoryNumber(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	typeID := uuid.New()

	newSvc := func() (*EquipmentService, *fakeEquipmentRepo) {
		repo := &fakeEquipmentRepo{
			eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
		}
		return NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t)), repo
	}

	t.Run("единица нумеруется с нуля", func(t *testing.T) {
		svc, repo := newSvc()

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created.InventoryNumber)
		assert.Equal(t, "000001", *repo.created.InventoryNumber)
	})

	t.Run("следующая единица получает следующий номер", func(t *testing.T) {
		svc, repo := newSvc()

		for i, want := range []string{"000001", "000002", "000003"} {
			_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
				RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
			}})
			require.NoError(t, err, "создание #%d", i+1)
			require.NotNil(t, repo.created.InventoryNumber)
			assert.Equal(t, want, *repo.created.InventoryNumber)
		}
	})

	t.Run("у комплекта своя серия с префиксом", func(t *testing.T) {
		svc, repo := newSvc()

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, Name: "Комплект", Status: models.EquipmentStatusInStock, IsKit: true,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created.InventoryNumber)
		assert.Equal(t, "KIT-000001", *repo.created.InventoryNumber)

		// Счётчик единиц не тронут: серии независимы.
		_, err = svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
		}})
		require.NoError(t, err)
		assert.Equal(t, "000001", *repo.created.InventoryNumber)
	})

	t.Run("присланный номер не трогаем", func(t *testing.T) {
		svc, repo := newSvc()
		manual := "INV-777"

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
			InventoryNumber: &manual,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created.InventoryNumber)
		assert.Equal(t, manual, *repo.created.InventoryNumber)
	})

	t.Run("занятый ручной номер сдвигает счётчик", func(t *testing.T) {
		svc, repo := newSvc()
		repo.invTaken = map[string]bool{"000001": true}

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created.InventoryNumber)
		assert.Equal(t, "000002", *repo.created.InventoryNumber, "занятый номер пропускается")
	})

	t.Run("пустой номер при правке генерируется", func(t *testing.T) {
		svc, repo := newSvc()
		id := uuid.New()

		_, err := svc.Update(ctx, &models.EquipmentDTO{
			ID: &id, RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.updated.InventoryNumber)
		assert.Equal(t, "000001", *repo.updated.InventoryNumber)
	})

	t.Run("серия при правке берётся из сохранённого состояния", func(t *testing.T) {
		svc, repo := newSvc()
		id := uuid.New()
		// Сохранённая единица — комплект; `is_kit` при правке репозиторий не
		// перезаписывает, поэтому серия определяется именно им.
		repo.parent = &models.Equipment{ID: id, RealmID: realmID, IsKit: true, TypeKind: models.EquipmentTypeDevice}

		_, err := svc.Update(ctx, &models.EquipmentDTO{
			ID: &id, RealmID: realmID, Name: "Комплект", Status: models.EquipmentStatusInStock, IsKit: true,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.updated.InventoryNumber)
		assert.Equal(t, "KIT-000001", *repo.updated.InventoryNumber)
	})

	t.Run("присланный номер при правке сохраняется", func(t *testing.T) {
		svc, repo := newSvc()
		id := uuid.New()
		manual := "INV-777"

		_, err := svc.Update(ctx, &models.EquipmentDTO{
			ID: &id, RealmID: realmID, TypeID: &typeID, Name: "Ноутбук", Status: models.EquipmentStatusInStock,
			InventoryNumber: &manual,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.updated.InventoryNumber)
		assert.Equal(t, manual, *repo.updated.InventoryNumber)
	})
}
