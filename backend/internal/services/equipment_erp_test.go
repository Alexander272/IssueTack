package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTicketCreator запоминает созданные заявки: оборудованию важен сам факт
// создания и реквизиты (категория/группа/площадка, ссылка на единицу, автор).
type fakeTicketCreator struct {
	created []*models.TicketDTO
	err     error
}

func (f *fakeTicketCreator) Create(_ context.Context, dto *models.TicketDTO) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, dto)
	return nil
}

// erpTestConfig — валидные реквизиты ERP-заявки: все три id заданы.
func erpTestConfig() config.EquipmentConfig {
	return config.EquipmentConfig{
		ERPCategoryID: uuid.New().String(),
		ERPGroupID:    uuid.New().String(),
		ERPSiteID:     uuid.New().String(),
	}
}

// TestEquipmentService_Assign_CreatesERPTicket: выдача заводит заявку «провести
// передачу техники в 1C:ERP» — обычную заявку, привязанную к единице, с автором и
// заказчиком, равными тому, кто выдал.
func TestEquipmentService_Assign_CreatesERPTicket(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "Кладовщик"}
	erp := erpTestConfig()

	repo := &fakeAssignRepo{unit: &models.Equipment{Name: "Ноутбук", SiteID: &siteID}}
	creator := &fakeTicketCreator{}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
	svc.SetERPConfig(erp)
	svc.SetTicketCreator(creator)

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SiteID: &siteID, SkipAct: true, Actor: actor,
	})
	require.NoError(t, err)
	require.Len(t, creator.created, 1)

	ticket := creator.created[0]
	assert.Contains(t, ticket.Title, "1C:ERP")
	assert.Contains(t, ticket.Title, "Ноутбук")
	assert.Equal(t, actor.ID, ticket.CreatorID)
	require.NotNil(t, ticket.OwnerID)
	assert.Equal(t, actor.ID, *ticket.OwnerID)
	// Исполнитель — тоже проводивший операцию, а не ответственный по умолчанию
	// группы: иначе autoAssign отдал бы заявку постороннему.
	require.NotNil(t, ticket.AssigneeID)
	assert.Equal(t, actor.ID, *ticket.AssigneeID)
	require.NotNil(t, ticket.EquipmentID)
	require.NotNil(t, ticket.GroupID)
	assert.Equal(t, uuid.MustParse(erp.ERPGroupID), *ticket.GroupID)
	assert.Equal(t, uuid.MustParse(erp.ERPCategoryID), ticket.CategoryID)
	assert.Equal(t, uuid.MustParse(erp.ERPSiteID), ticket.SiteID)
	assert.Contains(t, ticket.Description, "Провести выдачу")
}

// TestEquipmentService_Unassign_CreatesERPTicket: возврат заводит свою заявку
// (приёмка), а не переиспользует текст выдачи.
func TestEquipmentService_Unassign_CreatesERPTicket(t *testing.T) {
	ctx := context.Background()
	siteID := uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "Кладовщик"}

	repo := &fakeUnassignRepo{unit: &models.Equipment{Name: "Ноутбук", SiteID: &siteID}}
	creator := &fakeTicketCreator{}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
	svc.SetERPConfig(erpTestConfig())
	svc.SetTicketCreator(creator)

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{ID: uuid.New(), RealmID: uuid.New(), Actor: actor})
	require.NoError(t, err)
	require.Len(t, creator.created, 1)

	ticket := creator.created[0]
	assert.Contains(t, ticket.Title, "приёмку")
	assert.Contains(t, ticket.Description, "Провести приёмку")
}

// TestEquipmentService_ERP_Skipped: без настроенных реквизитов или без создателя
// заявок ERP-заявка не создаётся, но выдача/возврат проходят успешно — ERP-задача
// не должна ломать основную операцию.
func TestEquipmentService_ERP_Skipped(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()
	actor := &models.Actor{ID: uuid.New(), Name: "Кладовщик"}

	cases := []struct {
		name       string
		erp        config.EquipmentConfig
		withCreate bool
	}{
		{name: "реквизиты не заданы", erp: config.EquipmentConfig{}, withCreate: true},
		{name: "частичные реквизиты", erp: config.EquipmentConfig{ERPCategoryID: uuid.New().String()}, withCreate: true},
		{name: "нет создателя заявок", erp: erpTestConfig(), withCreate: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAssignRepo{unit: &models.Equipment{Name: "Ноутбук", SiteID: &siteID}}
			creator := &fakeTicketCreator{}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))
			svc.SetERPConfig(tt.erp)
			if tt.withCreate {
				svc.SetTicketCreator(creator)
			}

			err := svc.Assign(ctx, &models.AssignEquipmentDTO{
				ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SiteID: &siteID, SkipAct: true, Actor: actor,
			})
			require.NoError(t, err)
			assert.Empty(t, creator.created)
		})
	}
}
