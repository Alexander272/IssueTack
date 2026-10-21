package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentService_CreateKeepsCallerDTO закрепляет правило: сервис не
// мутирует тело запроса. Нормализация пустых строк, вывод статуса из держателя
// и сгенерированный инвентарный номер — всё это происходит на копии.
//
// Регресс, ради которого правило заведено: сервис правил DTO вызывающего
// (normalizeEquipmentDTO, deriveHolderStatus, выдача номера) — и тот же DTO,
// переиспользованный в тесте или в ретрае, вёл себя бы уже по правилам
// сервиса, а не по присланным данным.
func TestEquipmentService_CreateKeepsCallerDTO(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, holderID := uuid.New(), uuid.New(), uuid.New()

	repo := &fakeEquipmentRepo{
		eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	empty := ""
	dto := &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
		RealmID:      realmID,
		TypeID:       &typeID,
		Name:         "Единица",
		Status:       models.EquipmentStatusInStock,
		SerialNumber: &empty,
		HolderUserID: &holderID,
	}}

	created, err := svc.Create(ctx, dto)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Сервис правит копию: пустая строка — в NULL, статус выведен из держателя,
	// номер выдан сервером.
	require.NotNil(t, repo.created)
	assert.Nil(t, repo.created.SerialNumber)
	assert.Equal(t, models.EquipmentStatusInUse, repo.created.Status)
	assert.NotNil(t, repo.created.InventoryNumber)

	// Тело запроса осталось как было.
	require.NotNil(t, dto.SerialNumber)
	assert.Equal(t, "", *dto.SerialNumber)
	assert.Equal(t, models.EquipmentStatusInStock, dto.Status)
	assert.Nil(t, dto.InventoryNumber)
	assert.Nil(t, dto.ID)
}

// TestEquipmentService_UpdateKeepsCallerDTO — то же для правки: сервис берёт
// сохранённый статус и сгенерированный номер, но в тело запроса не пишет.
func TestEquipmentService_UpdateKeepsCallerDTO(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, id := uuid.New(), uuid.New(), uuid.New()

	repo := &fakeEquipmentRepo{
		eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
		// Сохранённый ручной статус правка не переписывает (см.
		// TestEquipmentService_UpdateKeepsStatus).
		parent: &models.Equipment{
			ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice,
			Status: models.EquipmentStatusDefective,
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	dto := &models.EquipmentDTO{
		ID:      &id,
		RealmID: realmID,
		TypeID:  &typeID,
		Name:    "Единица",
		Status:  models.EquipmentStatusInStock,
	}

	_, err := svc.Update(ctx, dto)
	require.NoError(t, err)

	require.NotNil(t, repo.updated)
	assert.Equal(t, models.EquipmentStatusDefective, repo.updated.Status)
	assert.NotNil(t, repo.updated.InventoryNumber)

	// Тело запроса осталось как было.
	assert.Equal(t, models.EquipmentStatusInStock, dto.Status)
	assert.Nil(t, dto.InventoryNumber)
}
