package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEquipmentService_CreateWritesCreatedEvent закрепляет: создание единицы
// пишет событие «создано» с автором операции и опознавательными полями снимка —
// без него в истории не было бы строки «поступил на склад».
func TestEquipmentService_CreateWritesCreatedEvent(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, actorID := uuid.New(), uuid.New(), uuid.New()
	inventory := "INV-1003"

	repo := &fakeEquipmentRepo{
		eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	created, err := svc.Create(ctx, &models.CreateEquipmentDTO{EquipmentDTO: models.EquipmentDTO{
		RealmID: realmID, TypeID: &typeID, Name: "Принтер HP LaserJet",
		Status: models.EquipmentStatusInStock, InventoryNumber: &inventory,
		Actor: &models.Actor{ID: actorID},
	}})
	require.NoError(t, err)

	require.Len(t, repo.events, 1)
	event := repo.events[0]
	assert.Equal(t, models.EquipmentEventCreated, event.EventType)
	assert.Equal(t, created.ID, event.EquipmentID)
	require.NotNil(t, event.UserID)
	assert.Equal(t, actorID, *event.UserID)

	var snapshot map[string]any
	require.NoError(t, json.Unmarshal(event.NewValue, &snapshot))
	assert.Equal(t, "INV-1003", snapshot["inventoryNumber"])
	assert.Equal(t, "Принтер HP LaserJet", snapshot["name"])
}

// TestEquipmentService_UpdateWritesMovedEvent закрепляет: «перемещено» пишется
// только при смене площадки или локации. Форма редактирования присылает все поля
// целиком, поэтому без сравнения со состоянием до правки журнал засорялся бы
// перемещениями на каждом сохранении.
func TestEquipmentService_UpdateWritesMovedEvent(t *testing.T) {
	ctx := context.Background()
	realmID, typeID, actorID := uuid.New(), uuid.New(), uuid.New()
	oldSite, newSite := uuid.New(), uuid.New()
	oldLocationID, newOldSiteLocationID, newSiteLocationID := uuid.New(), uuid.New(), uuid.New()
	locations := newFakeSiteLocations()
	locations.add(oldSite, oldLocationID, "Главный офис, каб. 301")
	locations.add(oldSite, newOldSiteLocationID, "Сервисная зона")
	locations.add(newSite, newSiteLocationID, "Сервисная зона")

	cases := []struct {
		name         string
		siteID       *uuid.UUID
		locationID   *uuid.UUID
		locationName string
		wantEvents   int
	}{
		{name: "сменились площадка и локация", siteID: &newSite, locationID: &newSiteLocationID, locationName: "Сервисная зона", wantEvents: 1},
		{name: "сменилась только локация", siteID: &oldSite, locationID: &newOldSiteLocationID, locationName: "Сервисная зона", wantEvents: 1},
		{name: "ничего не изменилось", siteID: &oldSite, locationID: &oldLocationID, locationName: "Главный офис, каб. 301", wantEvents: 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.New()
			repo := &fakeEquipmentRepo{
				eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
				// Состояние до правки — то, с чем сравнивается перемещение.
				parent: &models.Equipment{
					ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice,
					SiteID: &oldSite, LocationID: &oldLocationID, LocationName: "Главный офис, каб. 301",
				},
				locations: locations,
			}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
			svc.SetLocations(locations)

			_, err := svc.Update(ctx, &models.EquipmentDTO{
				ID: &id, RealmID: realmID, TypeID: &typeID, Name: "Принтер HP LaserJet",
				Status:     models.EquipmentStatusInUse,
				SiteID:     tt.siteID,
				LocationID: tt.locationID,
				Actor:      &models.Actor{ID: actorID},
			})
			require.NoError(t, err)
			require.Len(t, repo.events, tt.wantEvents)
			if tt.wantEvents == 0 {
				return
			}

			event := repo.events[0]
			assert.Equal(t, models.EquipmentEventMoved, event.EventType)
			assert.Equal(t, id, event.EquipmentID)
			require.NotNil(t, event.UserID)
			assert.Equal(t, actorID, *event.UserID)

			var oldValue, newValue map[string]any
			require.NoError(t, json.Unmarshal(event.OldValue, &oldValue))
			require.NoError(t, json.Unmarshal(event.NewValue, &newValue))
			assert.Equal(t, "Главный офис, каб. 301", oldValue["location"])
			assert.Equal(t, tt.locationName, newValue["location"])
			assert.Equal(t, tt.siteID.String(), newValue["siteId"])
		})
	}
}
