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

// fakeCompositionRepo подменяет методы, нужные составу логического комплекта:
// сам комплект, составные единицы и открытые членства. Проверяется доменное
// правило «в комплект входят только устройства и периферия».
type fakeCompositionRepo struct {
	repository.Equipment

	kit *models.Equipment
	// units — составные единицы по id (GetByID отдаёт их как компоненты).
	units map[uuid.UUID]*models.Equipment
	// openByComponent — открытые членства компонентов.
	openByComponent map[uuid.UUID]*models.KitComposition

	opened []*models.KitComposition
	events []*models.EquipmentEvent
	// kitReads — сколько раз GetByID вернул комплект (для диагностики).
	kitReads int
}

func (f *fakeCompositionRepo) GetByID(_ context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	if f.kit != nil && req.ID == f.kit.ID {
		f.kitReads++
		kit := *f.kit
		kit.RealmID = req.RealmID
		return &kit, nil
	}
	if unit, ok := f.units[req.ID]; ok {
		copied := *unit
		return &copied, nil
	}
	return nil, models.ErrNotFound
}

func (f *fakeCompositionRepo) GetOpenByComponent(_ context.Context, componentID uuid.UUID) (*models.KitComposition, error) {
	if open, ok := f.openByComponent[componentID]; ok {
		return open, nil
	}
	return nil, models.ErrNotFound
}

func (f *fakeCompositionRepo) OpenComposition(_ context.Context, _ postgres.Tx, c *models.KitComposition) error {
	f.opened = append(f.opened, c)
	return nil
}

func (f *fakeCompositionRepo) CreateEvent(_ context.Context, _ postgres.Tx, e *models.EquipmentEvent) error {
	f.events = append(f.events, e)
	return nil
}

// TestEquipmentService_AddComponents_RejectsParts закрепляет: комплектующее или
// расходник в логический комплект не входит. Логический комплект — это про
// устройства (ноутбук, монитор, мышь), а комплектующие — про физический состав
// (parent_id): оси разные, и смешивать их нельзя. Раньше AddComponents принимал
// любую единицу, и в «Стандартном рабочем месте» оказывалась оперативная память.
func TestEquipmentService_AddComponents_RejectsParts(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	kitID := uuid.New()
	ramID := uuid.New()

	repo := &fakeCompositionRepo{
		kit: &models.Equipment{ID: kitID, RealmID: realmID, IsKit: true},
		units: map[uuid.UUID]*models.Equipment{
			ramID: {ID: ramID, RealmID: realmID, Name: "Память 16 ГБ", TypeKind: models.EquipmentTypeComponent},
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.AddComponents(ctx, &models.KitCompositionOpDTO{
		ID: kitID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: ramID}},
	})

	require.ErrorIs(t, err, models.ErrEquipmentPartInKit)
	assert.Empty(t, repo.opened, "членство не должно открываться")
	assert.Empty(t, repo.events, "событие журнала не должно писаться")
}

// TestEquipmentService_AddComponents_AllowsDevices — зеркальный кейс: устройство
// и периферию в комплект по-прежнему можно включить.
func TestEquipmentService_AddComponents_AllowsDevices(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	kitID := uuid.New()
	monitorID := uuid.New()

	repo := &fakeCompositionRepo{
		kit: &models.Equipment{ID: kitID, RealmID: realmID, IsKit: true},
		units: map[uuid.UUID]*models.Equipment{
			monitorID: {ID: monitorID, RealmID: realmID, Name: "Монитор 27\"", TypeKind: models.EquipmentTypePeriphery},
		},
		openByComponent: map[uuid.UUID]*models.KitComposition{},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.AddComponents(ctx, &models.KitCompositionOpDTO{
		ID: kitID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: monitorID}},
	})

	require.NoError(t, err)
	require.Len(t, repo.opened, 1)
	assert.Equal(t, monitorID, repo.opened[0].ComponentID)
}
