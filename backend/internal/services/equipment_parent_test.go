package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEquipmentRepo подменяет только методы, нужные проверке «в физический
// состав можно положить только комплектующее или расходник». Остальные методы
// интерфейса не вызываются.
type fakeEquipmentRepo struct {
	repository.Equipment
	eqType  *models.EquipmentType
	created *models.EquipmentDTO
	updated *models.EquipmentDTO
}

func (f *fakeEquipmentRepo) GetTypeByID(_ context.Context, _ *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error) {
	return f.eqType, nil
}

func (f *fakeEquipmentRepo) GetAncestors(_ context.Context, _, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeEquipmentRepo) Create(_ context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	f.created = dto
	return &models.Equipment{ID: uuid.New(), TypeID: dto.TypeID, ParentID: dto.ParentID}, nil
}

func (f *fakeEquipmentRepo) Update(_ context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	f.updated = dto
	return &models.Equipment{TypeID: dto.TypeID, ParentID: dto.ParentID}, nil
}

// TestEquipmentService_ParentRequiresPart закрепляет правило: родитель
// (физический состав, parent_id) допустим только у комплектующего или расходника.
func TestEquipmentService_ParentRequiresPart(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	parentID := uuid.New()

	cases := []struct {
		name       string
		kind       models.EquipmentTypeKind
		withParent bool
		allowed    bool
	}{
		{name: "комплектующее с родителем", kind: models.EquipmentTypeComponent, withParent: true, allowed: true},
		{name: "расходник с родителем", kind: models.EquipmentTypeConsumable, withParent: true, allowed: true},
		{name: "устройство с родителем", kind: models.EquipmentTypeDevice, withParent: true, allowed: false},
		{name: "периферия с родителем", kind: models.EquipmentTypePeriphery, withParent: true, allowed: false},
		{name: "устройство без родителя", kind: models.EquipmentTypeDevice, withParent: false, allowed: true},
		{name: "периферия без родителя", kind: models.EquipmentTypePeriphery, withParent: false, allowed: true},
	}

	for _, tt := range cases {
		t.Run("create/"+tt.name, func(t *testing.T) {
			typeID := uuid.New()
			repo := &fakeEquipmentRepo{
				eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: tt.kind},
			}
			svc := NewEquipmentService(repo, nil)

			dto := &models.EquipmentDTO{
				RealmID: realmID,
				TypeID:  typeID,
				Name:    "Единица",
				Status:  models.EquipmentStatusInStock,
			}
			if tt.withParent {
				dto.ParentID = &parentID
			}

			_, err := svc.Create(ctx, dto)

			if tt.allowed {
				require.NoError(t, err)
				assert.NotNil(t, repo.created)
				return
			}
			require.ErrorIs(t, err, models.ErrEquipmentParentNotPart)
			assert.Nil(t, repo.created)
		})

		t.Run("update/"+tt.name, func(t *testing.T) {
			typeID := uuid.New()
			repo := &fakeEquipmentRepo{
				eqType: &models.EquipmentType{ID: typeID, RealmID: realmID, Kind: tt.kind},
			}
			svc := NewEquipmentService(repo, nil)

			id := uuid.New()
			dto := &models.EquipmentDTO{
				ID:      &id,
				RealmID: realmID,
				TypeID:  typeID,
				Name:    "Единица",
				Status:  models.EquipmentStatusInStock,
			}
			if tt.withParent {
				dto.ParentID = &parentID
			}

			_, err := svc.Update(ctx, dto)

			if tt.allowed {
				require.NoError(t, err)
				assert.NotNil(t, repo.updated)
				return
			}
			require.ErrorIs(t, err, models.ErrEquipmentParentNotPart)
			assert.Nil(t, repo.updated)
		})
	}
}
