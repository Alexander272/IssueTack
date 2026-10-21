package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRealmSupervisor — подмена проверки «начальник области» для списания.
type fakeRealmSupervisor struct {
	supervisor bool
	err        error
	calls      int
}

func (f *fakeRealmSupervisor) IsRealmSupervisor(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	f.calls++
	return f.supervisor, f.err
}

// TestEquipmentService_AddDefect закрепляет правила постановки дефекта: дефект
// заводят только на активной единице (склад/эксплуатация) и не на комплекте,
// статус становится «дефектное», а в журнал пишется событие с описанием и датой
// обнаружения.
func TestEquipmentService_AddDefect(t *testing.T) {
	ctx := context.Background()
	realmID, typeID := uuid.New(), uuid.New()

	cases := []struct {
		name    string
		unit    *models.Equipment
		wantErr error
	}{
		{
			name: "склад — дефект можно",
			unit: &models.Equipment{ID: uuid.New(), RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusInStock},
		},
		{
			name: "эксплуатация — дефект можно",
			unit: &models.Equipment{ID: uuid.New(), RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusInUse},
		},
		{
			name:    "уже дефектная — отказ",
			unit:    &models.Equipment{ID: uuid.New(), RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusDefective},
			wantErr: models.ErrEquipmentDefectNotAllowed,
		},
		{
			name:    "списанная — отказ",
			unit:    &models.Equipment{ID: uuid.New(), RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusWrittenOff},
			wantErr: models.ErrEquipmentDefectNotAllowed,
		},
		{
			name:    "ремонт — отказ",
			unit:    &models.Equipment{ID: uuid.New(), RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusRepair},
			wantErr: models.ErrEquipmentDefectNotAllowed,
		},
		{
			name:    "комплект — отказ (у него нет своего состояния)",
			unit:    &models.Equipment{ID: uuid.New(), RealmID: realmID, IsKit: true, Status: models.EquipmentStatusInStock},
			wantErr: models.ErrEquipmentDefectNotAllowed,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeEquipmentRepo{eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice}, parent: tt.unit}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			_, err := svc.AddDefect(ctx, &models.AddDefectDTO{
				ID: tt.unit.ID, RealmID: realmID, Description: "Не включается",
				Actor: &models.Actor{ID: uuid.New()},
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, repo.statuses, "статус не должен меняться при отказе")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, []models.EquipmentStatus{models.EquipmentStatusDefective}, repo.statuses)
			require.Len(t, repo.events, 1)
			assert.Equal(t, models.EquipmentEventDefective, repo.events[0].EventType)
			// Описание и дата обнаружения лежат в снимке события — ради них тип и
			// отдельный: лента печатает описание, а не переход статуса.
			assert.Contains(t, string(repo.events[0].NewValue), "Не включается")
			// Дата хранится только днём (YYYY-MM-DD): время обнаружения никто не
			// фиксирует, а момент записи даёт createdAt самого события.
			assert.Regexp(t, `"foundAt":"\d{4}-\d{2}-\d{2}"`, string(repo.events[0].NewValue))
		})
	}
}

// TestEquipmentService_AddDefect_FoundAt проверяет границы даты обнаружения:
// будущая дата отклоняется (дефект — факт, а не план), а дата сегодня с любым
// временем суток допустима; в снимок попадает только день.
func TestEquipmentService_AddDefect_FoundAt(t *testing.T) {
	ctx := context.Background()
	realmID, typeID := uuid.New(), uuid.New()

	newSvc := func(id uuid.UUID) (*EquipmentService, *fakeEquipmentRepo) {
		repo := &fakeEquipmentRepo{
			eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
			parent: &models.Equipment{ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: models.EquipmentStatusInStock},
		}
		return NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t)), repo
	}

	t.Run("будущая дата — отказ", func(t *testing.T) {
		id := uuid.New()
		svc, repo := newSvc(id)
		future := time.Now().Add(48 * time.Hour)
		_, err := svc.AddDefect(ctx, &models.AddDefectDTO{
			ID: id, RealmID: realmID, Description: "Что-то", FoundAt: &future,
			Actor: &models.Actor{ID: uuid.New()},
		})
		require.ErrorIs(t, err, models.ErrEquipmentDefectDateInFuture)
		assert.Empty(t, repo.statuses)
	})

	t.Run("сегодня, позднее время — можно (сравнение по дню)", func(t *testing.T) {
		id := uuid.New()
		svc, repo := newSvc(id)
		// Конец сегодняшних суток: время в будущем, но день — сегодняшний.
		today := time.Now().UTC()
		late := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 0, 0, time.UTC)
		_, err := svc.AddDefect(ctx, &models.AddDefectDTO{
			ID: id, RealmID: realmID, Description: "Что-то", FoundAt: &late,
			Actor: &models.Actor{ID: uuid.New()},
		})
		require.NoError(t, err)
		require.Len(t, repo.events, 1)
		// Время суток отбрасывается: в снимке только дата.
		assert.Contains(t, string(repo.events[0].NewValue), fmt.Sprintf(`"foundAt":"%s"`, today.Format("2006-01-02")))
	})
}

// TestEquipmentService_WriteOff закрепляет правила списания: списать можно только
// дефектную единицу, и только начальнику области. Оба условия — отдельные отказы,
// чтобы ошибка объясняла, чего именно не хватает.
func TestEquipmentService_WriteOff(t *testing.T) {
	ctx := context.Background()
	realmID, typeID := uuid.New(), uuid.New()

	cases := []struct {
		name       string
		status     models.EquipmentStatus
		supervisor bool
		access     bool
		wantErr    error
	}{
		{name: "дефектная + начальник области — списание", status: models.EquipmentStatusDefective, supervisor: true, access: true},
		{name: "не дефектная — отказ", status: models.EquipmentStatusInStock, supervisor: true, access: true, wantErr: models.ErrEquipmentWriteOffNotDefective},
		{name: "дефектная, но не начальник — отказ", status: models.EquipmentStatusDefective, supervisor: false, access: true, wantErr: models.ErrEquipmentWriteOffForbidden},
		{name: "дефектная, проверки прав нет — отказ", status: models.EquipmentStatusDefective, access: false, wantErr: models.ErrEquipmentWriteOffForbidden},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.New()
			repo := &fakeEquipmentRepo{
				eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: models.EquipmentTypeDevice},
				parent: &models.Equipment{ID: id, RealmID: realmID, TypeKind: models.EquipmentTypeDevice, Status: tt.status},
			}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
			if tt.access {
				svc.SetAccessChecker(&fakeRealmSupervisor{supervisor: tt.supervisor})
			}

			_, err := svc.WriteOff(ctx, &models.WriteOffEquipmentDTO{
				ID: id, RealmID: realmID, Comment: "Не подлежит ремонту",
				Actor: &models.Actor{ID: uuid.New()},
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, repo.statuses, "статус не должен меняться при отказе")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, []models.EquipmentStatus{models.EquipmentStatusWrittenOff}, repo.statuses)
			require.Len(t, repo.events, 1)
			assert.Equal(t, models.EquipmentEventWrittenOff, repo.events[0].EventType)
			assert.Equal(t, "Не подлежит ремонту", repo.events[0].Comment)
		})
	}
}
