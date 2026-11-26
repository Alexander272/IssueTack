package services

import (
	"context"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUnassignRepo подменяет только методы, нужные возврату единицы на склад.
type fakeUnassignRepo struct {
	repository.Equipment

	unit *models.Equipment
	// siteName — подпись площадки: её резолвит репозиторий для снимка события.
	siteName string
	// openAssignment — текущая открытая выдача; nil — единица не выдана.
	openAssignment *models.EquipmentAssignment
	// stored — место последнего складского периода («дом»); found=false — его не было.
	stored struct {
		siteID, locationID *uuid.UUID
		found              bool
	}

	createdAssignment *models.EquipmentAssignment
	closedIDs         []uuid.UUID
	events            []*models.EquipmentEvent
	holder            struct {
		userID, siteID, locationID *uuid.UUID
		status                     models.EquipmentStatus
	}
}

func (f *fakeUnassignRepo) GetLastStoredPlace(_ context.Context, _ uuid.UUID) (*uuid.UUID, *uuid.UUID, error) {
	if !f.stored.found {
		return nil, nil, models.ErrNotFound
	}
	return f.stored.siteID, f.stored.locationID, nil
}

func (f *fakeUnassignRepo) GetByID(_ context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	unit := *f.unit
	unit.ID, unit.RealmID = req.ID, req.RealmID
	return &unit, nil
}

func (f *fakeUnassignRepo) GetChildren(_ context.Context, _, _ uuid.UUID) ([]*models.Equipment, error) {
	return nil, nil
}

func (f *fakeUnassignRepo) GetSiteName(_ context.Context, _ uuid.UUID) (string, error) {
	return f.siteName, nil
}

func (f *fakeUnassignRepo) GetOpenAssignment(_ context.Context, _ uuid.UUID) (*models.EquipmentAssignment, error) {
	if f.openAssignment == nil {
		return nil, models.ErrNotFound
	}
	return f.openAssignment, nil
}

func (f *fakeUnassignRepo) CloseAssignment(_ context.Context, _ postgres.Tx, id uuid.UUID, _ time.Time, _ string) error {
	f.closedIDs = append(f.closedIDs, id)
	return nil
}

func (f *fakeUnassignRepo) CreateAssignment(_ context.Context, _ postgres.Tx, a *models.EquipmentAssignment) error {
	f.createdAssignment = a
	return nil
}

func (f *fakeUnassignRepo) SetHolderAndStatus(
	_ context.Context, _ postgres.Tx, dto *models.EquipmentHolderDTO,
) error {
	f.holder.userID, f.holder.siteID, f.holder.locationID, f.holder.status =
		dto.UserID, dto.SiteID, dto.LocationID, dto.Status
	return nil
}

func (f *fakeUnassignRepo) CreateEvent(_ context.Context, _ postgres.Tx, e *models.EquipmentEvent) error {
	f.events = append(f.events, e)
	return nil
}

// TestEquipmentService_Unassign_KeepsCurrentPlace закрепляет: если складского
// периода у единицы не было (её завели сразу на держателя), возврат берёт её
// текущее место. Кнопка «Возврат» не открывает диалог, поэтому без этого фолбэка
// она всегда упиралась бы в EQ008.
func TestEquipmentService_Unassign_KeepsCurrentPlace(t *testing.T) {
	ctx := context.Background()
	siteID, holderID, locationID := uuid.New(), uuid.New(), uuid.New()
	location := "Кабинет 305"

	repo := &fakeUnassignRepo{
		unit:           &models.Equipment{SiteID: &siteID, LocationID: &locationID},
		siteName:       "ППРЗ",
		openAssignment: &models.EquipmentAssignment{ID: uuid.New(), UserID: &holderID},
	}
	locations := newFakeSiteLocations()
	locations.add(siteID, locationID, location)
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
	svc.SetLocations(locations)

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{ID: uuid.New(), RealmID: uuid.New()})
	require.NoError(t, err)

	// Прежняя выдача закрыта, открыта новая строка — уже без держателя.
	require.Len(t, repo.closedIDs, 1)
	require.NotNil(t, repo.createdAssignment)
	assert.Nil(t, repo.createdAssignment.UserID)

	assert.Nil(t, repo.holder.userID)
	require.NotNil(t, repo.holder.siteID)
	assert.Equal(t, siteID, *repo.holder.siteID)
	require.NotNil(t, repo.holder.locationID)
	assert.Equal(t, locationID, *repo.holder.locationID)
	assert.Equal(t, models.EquipmentStatusInStock, repo.holder.status)

	// Снимок события несёт подписи площадки и расположения: лента истории не
	// тянет справочники.
	require.Len(t, repo.events, 1)
	assert.Contains(t, string(repo.events[0].NewValue), `"siteName":"ППРЗ"`)
	assert.Contains(t, string(repo.events[0].NewValue), `"location":"Кабинет 305"`)
}

// TestEquipmentService_Unassign_ReturnsHome: единица возвращается туда, где лежала
// до выдачи, а не в кабинет держателя. Место берётся из последнего складского
// периода, даже если у самой единицы место другое (выдали сразу на объект) — это и
// есть «возврат домой».
func TestEquipmentService_Unassign_ReturnsHome(t *testing.T) {
	ctx := context.Background()
	holderSiteID, homeSiteID := uuid.New(), uuid.New()
	holderID, holderLocationID, homeLocationID := uuid.New(), uuid.New(), uuid.New()
	homeLocation := "Склад, полка A"

	repo := &fakeUnassignRepo{
		unit:           &models.Equipment{SiteID: &holderSiteID, LocationID: &holderLocationID},
		openAssignment: &models.EquipmentAssignment{ID: uuid.New(), UserID: &holderID},
	}
	repo.stored.siteID, repo.stored.locationID, repo.stored.found = &homeSiteID, &homeLocationID, true
	locations := newFakeSiteLocations()
	locations.add(homeSiteID, homeLocationID, homeLocation)

	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
	svc.SetLocations(locations)

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{ID: uuid.New(), RealmID: uuid.New()})
	require.NoError(t, err)

	require.NotNil(t, repo.holder.siteID)
	assert.Equal(t, homeSiteID, *repo.holder.siteID)
	require.NotNil(t, repo.holder.locationID)
	assert.Equal(t, homeLocationID, *repo.holder.locationID)

	// В складскую строку и в снимок события уходит то же место.
	require.NotNil(t, repo.createdAssignment)
	require.NotNil(t, repo.createdAssignment.SiteID)
	assert.Equal(t, homeSiteID, *repo.createdAssignment.SiteID)
	assert.Equal(t, homeLocationID, *repo.createdAssignment.LocationID)
	assert.Contains(t, string(repo.events[0].NewValue), `"location":"Склад, полка A"`)
}

// TestEquipmentService_Unassign_ExplicitPlaceWins: явное место из тела запроса
// важнее и текущего, и «дома» — единицу могут вернуть сразу на другую площадку.
func TestEquipmentService_Unassign_ExplicitPlaceWins(t *testing.T) {
	ctx := context.Background()
	currentSiteID, newSiteID, homeSiteID := uuid.New(), uuid.New(), uuid.New()

	repo := &fakeUnassignRepo{unit: &models.Equipment{SiteID: &currentSiteID}}
	repo.stored.siteID, repo.stored.found = &homeSiteID, true
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), SiteID: &newSiteID,
	})
	require.NoError(t, err)
	require.NotNil(t, repo.holder.siteID)
	assert.Equal(t, newSiteID, *repo.holder.siteID)
}

// TestEquipmentService_Unassign_RequiresPlaceWhenUnitHasNone: инвариант «держатель
// или место» держит CHECK в equipment_assignments, поэтому единицу без места (за
// ней был только держатель) вернуть нельзя — клиент обязан указать площадку.
func TestEquipmentService_Unassign_RequiresPlaceWhenUnitHasNone(t *testing.T) {
	ctx := context.Background()

	repo := &fakeUnassignRepo{unit: &models.Equipment{}}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{ID: uuid.New(), RealmID: uuid.New()})
	assert.ErrorIs(t, err, models.ErrEquipmentNoHoldersOrLocation)
	assert.Empty(t, repo.createdAssignment)
	assert.Empty(t, repo.events)
}
