package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeKitTemplateRepo подменяет только методы шаблонов комплектов: проверяется
// нормализация состава на стороне сервиса, остальной интерфейс не вызывается.
type fakeKitTemplateRepo struct {
	repository.Equipment
	created *models.EquipmentKitTemplateDTO
	updated *models.EquipmentKitTemplateDTO
	deleted *models.DelEquipmentKitTemplateDTO
}

func (f *fakeKitTemplateRepo) CreateKitTemplate(_ context.Context, _ postgres.Tx, dto *models.EquipmentKitTemplateDTO) error {
	f.created = dto
	return nil
}

func (f *fakeKitTemplateRepo) UpdateKitTemplate(_ context.Context, _ postgres.Tx, dto *models.EquipmentKitTemplateDTO) error {
	f.updated = dto
	return nil
}

func (f *fakeKitTemplateRepo) DeleteKitTemplate(_ context.Context, dto *models.DelEquipmentKitTemplateDTO) error {
	f.deleted = dto
	return nil
}

// TestEquipmentService_KitTemplateNormalize закрепляет чистку состава шаблона:
// пустые типы отбрасываются, sort_order идёт по порядку, а ПОВТОРЯЮЩИЕСЯ типы
// сохраняются — в шаблоне «Стандартное рабочее место» два монитора это две
// позиции одного типа. Нулевой typeId ломал бы FK, поэтому отбрасывается.
func TestEquipmentService_KitTemplateNormalize(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	typeA, typeB := uuid.New(), uuid.New()

	repo := &fakeKitTemplateRepo{}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	dto := &models.EquipmentKitTemplateDTO{
		RealmID: realmID,
		Name:    "Стандартное рабочее место",
		Items: []models.EquipmentKitTemplateItemDTO{
			{TypeID: uuid.Nil},
			{TypeID: typeA},
			{TypeID: typeB},
			{TypeID: typeA},
		},
	}
	require.NoError(t, svc.CreateKitTemplate(ctx, dto))

	require.NotNil(t, repo.created)
	require.Len(t, repo.created.Items, 3)
	assert.Equal(t, typeA, repo.created.Items[0].TypeID)
	assert.Equal(t, 0, repo.created.Items[0].SortOrder)
	assert.Equal(t, typeB, repo.created.Items[1].TypeID)
	assert.Equal(t, 1, repo.created.Items[1].SortOrder)
	// Один и тот же тип во второй раз — отдельная позиция (второй монитор).
	assert.Equal(t, typeA, repo.created.Items[2].TypeID)
	assert.Equal(t, 2, repo.created.Items[2].SortOrder)
}

// TestEquipmentService_UpdateKitTemplate_RequiresID проверяет, что правка без id
// не уходит в репозиторий: запрос с WHERE id вернул бы ложный ErrNotFound.
func TestEquipmentService_UpdateKitTemplate_RequiresID(t *testing.T) {
	ctx := context.Background()
	repo := &fakeKitTemplateRepo{}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.UpdateKitTemplate(ctx, &models.EquipmentKitTemplateDTO{RealmID: uuid.New(), Name: "Без id"})

	require.ErrorIs(t, err, models.ErrInvalidInput)
	assert.Nil(t, repo.updated)
}

// TestEquipmentService_UpdateKitTemplate_ItemsContract закрепляет, как правка
// понимает состав: nil — «позиции не трогать» (иначе PUT без items молча стирал
// бы состав), а явный пустой массив — осознанная очистка. Различие живёт до
// репозитория, поэтому его и проверяем.
func TestEquipmentService_UpdateKitTemplate_ItemsContract(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("nil сохраняется", func(t *testing.T) {
		repo := &fakeKitTemplateRepo{}
		svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

		dto := &models.EquipmentKitTemplateDTO{ID: &id, RealmID: uuid.New(), Name: "Без состава"}
		require.NoError(t, svc.UpdateKitTemplate(ctx, dto))

		require.NotNil(t, repo.updated)
		assert.Nil(t, repo.updated.Items)
	})

	t.Run("пустой массив сохраняется пустым", func(t *testing.T) {
		repo := &fakeKitTemplateRepo{}
		svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

		dto := &models.EquipmentKitTemplateDTO{
			ID: &id, RealmID: uuid.New(), Name: "Очистить состав",
			Items: []models.EquipmentKitTemplateItemDTO{},
		}
		require.NoError(t, svc.UpdateKitTemplate(ctx, dto))

		require.NotNil(t, repo.updated)
		assert.NotNil(t, repo.updated.Items)
		assert.Empty(t, repo.updated.Items)
	})
}
