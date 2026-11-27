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
	// updates — сохранённые обновления единицы (физический состав через Update).
	updates []*models.EquipmentDTO
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

func (f *fakeCompositionRepo) Update(_ context.Context, _ postgres.Tx, dto *models.EquipmentDTO) (*models.Equipment, error) {
	f.updates = append(f.updates, dto)
	return &models.Equipment{ID: *dto.ID, RealmID: dto.RealmID, Name: dto.Name, Status: dto.Status, ParentID: dto.ParentID, IsKit: dto.IsKit}, nil
}

func (f *fakeCompositionRepo) SetParent(_ context.Context, _ postgres.Tx, realmID, id uuid.UUID, parentID *uuid.UUID) error {
	f.updates = append(f.updates, &models.EquipmentDTO{ID: &id, RealmID: realmID, ParentID: parentID})
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

// TestEquipmentService_AttachComponents_WritesEvent — операция физического
// состава прикрепляет компонент к хосту (parent_id) и пишет событие
// component_added в журнал. Этап состава закрывается операцией, а не апдейтом
// единицы через PUT.
func TestEquipmentService_AttachComponents_WritesEvent(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	hostID := uuid.New()
	partID := uuid.New()

	repo := &fakeCompositionRepo{
		units: map[uuid.UUID]*models.Equipment{
			hostID: {ID: hostID, RealmID: realmID, Name: "Ноутбук", TypeKind: models.EquipmentTypeDevice},
			partID: {ID: partID, RealmID: realmID, Name: "SSD 1 ТБ", TypeKind: models.EquipmentTypeComponent},
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.AttachComponents(ctx, &models.PhysicalComponentOpDTO{
		ID: hostID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: partID}},
		Reason:     "Установка",
	})
	require.NoError(t, err)

	require.Len(t, repo.updates, 1)
	require.NotNil(t, repo.updates[0].ParentID)
	assert.Equal(t, hostID, *repo.updates[0].ParentID)

	require.Len(t, repo.events, 1)
	assert.Equal(t, models.EquipmentEventComponentAdded, repo.events[0].EventType)
	assert.Equal(t, hostID, repo.events[0].EquipmentID)
	require.NotNil(t, repo.events[0].ComponentID)
	assert.Equal(t, partID, *repo.events[0].ComponentID)
}

// TestEquipmentService_AttachComponents_RejectsPartHost — хостом физического
// состава может быть только устройство или периферия: комплектующий/расходник
// не может содержать вложения, а комплект ведёт состав логически.
func TestEquipmentService_AttachComponents_RejectsPartHost(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	hostID := uuid.New()
	partID := uuid.New()

	repo := &fakeCompositionRepo{
		units: map[uuid.UUID]*models.Equipment{
			hostID: {ID: hostID, RealmID: realmID, Name: "Память 16 ГБ", TypeKind: models.EquipmentTypeComponent},
			partID: {ID: partID, RealmID: realmID, Name: "SSD 1 ТБ", TypeKind: models.EquipmentTypeComponent},
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.AttachComponents(ctx, &models.PhysicalComponentOpDTO{
		ID: hostID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: partID}},
	})

	require.ErrorIs(t, err, models.ErrEquipmentParentCannotHost)
	assert.Empty(t, repo.events, "событие журнала не должно писаться")
}

// TestEquipmentService_DetachComponents_ClearsParent — открепление компонента
// от хоста проставляет parent_id = NULL и пишет component_removed.
func TestEquipmentService_DetachComponents_ClearsParent(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	hostID := uuid.New()
	partID := uuid.New()
	attachedTo := hostID

	repo := &fakeCompositionRepo{
		units: map[uuid.UUID]*models.Equipment{
			hostID: {ID: hostID, RealmID: realmID, Name: "Ноутбук", TypeKind: models.EquipmentTypeDevice},
			partID: {ID: partID, RealmID: realmID, Name: "SSD 1 ТБ", TypeKind: models.EquipmentTypeComponent, ParentID: &attachedTo},
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.DetachComponents(ctx, &models.PhysicalComponentOpDTO{
		ID: hostID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: partID}},
	})
	require.NoError(t, err)

	require.Len(t, repo.updates, 1)
	require.Nil(t, repo.updates[0].ParentID)

	require.Len(t, repo.events, 1)
	assert.Equal(t, models.EquipmentEventComponentRemoved, repo.events[0].EventType)
	assert.Equal(t, hostID, repo.events[0].EquipmentID)
	require.NotNil(t, repo.events[0].ComponentID)
	assert.Equal(t, partID, *repo.events[0].ComponentID)
}

// TestEquipmentService_DetachComponents_NotAttached — открепление компонента,
// который не состоит в этом хосте, отклоняется.
func TestEquipmentService_DetachComponents_NotAttached(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	hostID := uuid.New()
	partID := uuid.New()

	repo := &fakeCompositionRepo{
		units: map[uuid.UUID]*models.Equipment{
			hostID: {ID: hostID, RealmID: realmID, Name: "Ноутбук", TypeKind: models.EquipmentTypeDevice},
			partID: {ID: partID, RealmID: realmID, Name: "SSD 1 ТБ", TypeKind: models.EquipmentTypeComponent},
		},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.DetachComponents(ctx, &models.PhysicalComponentOpDTO{
		ID: hostID, RealmID: realmID,
		Components: []models.ComponentOpItem{{ComponentID: partID}},
	})

	require.ErrorIs(t, err, models.ErrEquipmentComponentNotAttached)
	assert.Empty(t, repo.events, "событие журнала не должно писаться")
}
