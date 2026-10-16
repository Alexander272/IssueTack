package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentService_KitWithoutType закрепляет: у комплекта нет типа. Комплект
// создаётся без typeId, а присланный тип сервис обнуляет — рендер карточки и
// селекторы единиц не должны получить у комплекта чужой тип. Единица без типа,
// наоборот, отказ (type_id NOT NULL для всего, кроме комплектов).
func TestEquipmentService_KitWithoutType(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	typeID := uuid.New()

	newSvc := func() (*EquipmentService, *fakeEquipmentRepo) {
		repo := &fakeEquipmentRepo{
			eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
		}
		return NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t)), repo
	}

	t.Run("комплект без типа создаётся", func(t *testing.T) {
		svc, repo := newSvc()

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, Name: "Комплект", Status: models.EquipmentStatusInStock, IsKit: true,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created)
		assert.Nil(t, repo.created.TypeID, "у комплекта не должно быть типа")
	})

	t.Run("присланный тип комплекта обнуляется", func(t *testing.T) {
		svc, repo := newSvc()

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, Name: "Комплект", Status: models.EquipmentStatusInStock,
			IsKit: true, TypeID: &typeID,
		}})
		require.NoError(t, err)
		require.NotNil(t, repo.created)
		assert.Nil(t, repo.created.TypeID)
	})

	t.Run("единица без типа — отказ", func(t *testing.T) {
		svc, repo := newSvc()

		_, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
			RealmID: realmID, Name: "Единица", Status: models.EquipmentStatusInStock,
		}})
		require.ErrorIs(t, err, models.ErrInvalidInput)
		assert.Nil(t, repo.created)
	})

	t.Run("обновление комплекта типом не обзаводится", func(t *testing.T) {
		svc, repo := newSvc()

		id := uuid.New()
		_, err := svc.Update(ctx, &models.EquipmentDTO{
			ID: &id, RealmID: realmID, Name: "Комплект",
			Status: models.EquipmentStatusInStock, IsKit: true, TypeID: &typeID,
		})
		require.NoError(t, err)
		require.NotNil(t, repo.updated)
		assert.Nil(t, repo.updated.TypeID)
	})
}
