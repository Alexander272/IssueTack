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

// fakeOperationTaskRepo хранит правила в памяти. Встроенный repository.Equipment
// заглушает остальные методы интерфейса — тесты трогают только правила и чтение
// единицы после операции.
type fakeOperationTaskRepo struct {
	repository.Equipment
	tasks   []*models.EquipmentOperationTask
	err     error
	unit    *models.Equipment
	deleted *models.DelEquipmentOperationTaskDTO
}

func (f *fakeOperationTaskRepo) GetOperationTasks(_ context.Context, _ uuid.UUID, operation *models.EquipmentOperation) ([]*models.EquipmentOperationTask, error) {
	if f.err != nil {
		return nil, f.err
	}
	if operation == nil {
		return append([]*models.EquipmentOperationTask{}, f.tasks...), nil
	}
	tasks := []*models.EquipmentOperationTask{}
	for _, task := range f.tasks {
		if task.Operation == *operation {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (f *fakeOperationTaskRepo) GetByID(_ context.Context, _ *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	return f.unit, nil
}

func (f *fakeOperationTaskRepo) CreateOperationTask(_ context.Context, _ postgres.Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error) {
	id := uuid.New()
	return &models.EquipmentOperationTask{ID: id, RealmID: dto.RealmID, Name: dto.Name, Operation: dto.Operation,
		Action: dto.Action, CategoryID: dto.CategoryID, GroupID: dto.GroupID, Priority: dto.Priority, Enabled: dto.Enabled}, nil
}

func (f *fakeOperationTaskRepo) UpdateOperationTask(_ context.Context, _ postgres.Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error) {
	return &models.EquipmentOperationTask{ID: *dto.ID, RealmID: dto.RealmID, Name: dto.Name, Operation: dto.Operation,
		Action: dto.Action, CategoryID: dto.CategoryID, GroupID: dto.GroupID, Priority: dto.Priority, Enabled: dto.Enabled}, nil
}

func (f *fakeOperationTaskRepo) DeleteOperationTask(_ context.Context, dto *models.DelEquipmentOperationTaskDTO) error {
	f.deleted = dto
	return nil
}

func operationTaskFixture(op models.EquipmentOperation, enabled bool) *models.EquipmentOperationTask {
	return &models.EquipmentOperationTask{
		ID: uuid.New(), RealmID: uuid.New(), Name: "Передача в ERP", Operation: op,
		Action: models.EquipmentActionCreateTicket, Title: "Своя задача", Description: "Своё описание",
		CategoryID: uuid.New(), GroupID: uuid.New(), Priority: string(models.PriorityHigh), Enabled: enabled,
	}
}

// TestOperationTask_RunForOperation_UsesRulesAndEquipmentSite: правила в БД
// полностью заменяют конфиг; площадка заявки берётся у единицы, а не из правила
// или заказчика.
func TestOperationTask_RunForOperation_UsesRulesAndEquipmentSite(t *testing.T) {
	ctx := context.Background()
	realmID, equipmentID := uuid.New(), uuid.New()
	equipmentSite := uuid.New()
	actor := &models.Actor{ID: uuid.New()}
	task := operationTaskFixture(models.EquipmentOperationAssign, true)
	task.RealmID = realmID

	repo := &fakeOperationTaskRepo{
		tasks: []*models.EquipmentOperationTask{task, operationTaskFixture(models.EquipmentOperationAssign, false)},
		unit:  &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", SiteID: &equipmentSite},
	}
	creator := &fakeTicketCreator{}
	svc := NewEquipmentOperationTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)

	handled := svc.RunForOperation(ctx, models.EquipmentOperationAssign, equipmentID, realmID, actor)
	assert.True(t, handled)
	require.Len(t, creator.created, 1)

	ticket := creator.created[0]
	assert.Equal(t, "Своя задача", ticket.Title)
	assert.Equal(t, "Своё описание", ticket.Description)
	assert.Equal(t, equipmentSite, ticket.SiteID)
	assert.Equal(t, task.CategoryID, ticket.CategoryID)
	require.NotNil(t, ticket.GroupID)
	assert.Equal(t, task.GroupID, *ticket.GroupID)
	assert.Equal(t, models.PriorityHigh, ticket.Priority)
	assert.Equal(t, actor.ID, ticket.CreatorID)
	require.NotNil(t, ticket.OwnerID)
	assert.Equal(t, actor.ID, *ticket.OwnerID)
	require.NotNil(t, ticket.AssigneeID)
	assert.Equal(t, actor.ID, *ticket.AssigneeID)
	require.NotNil(t, ticket.EquipmentID)
	assert.Equal(t, equipmentID, *ticket.EquipmentID)
}

// TestOperationTask_RunForOperation_NoRulesFallsBack: без правил реалм живёт по
// старому конфигу — EquipmentService сам вызовет createERPTicket.
func TestOperationTask_RunForOperation_NoRulesFallsBack(t *testing.T) {
	ctx := context.Background()
	realmID, equipmentID := uuid.New(), uuid.New()
	repo := &fakeOperationTaskRepo{}
	svc := NewEquipmentOperationTaskService(repo, &mockTransactionManager{})

	assert.False(t, svc.RunForOperation(ctx, models.EquipmentOperationAssign, equipmentID, realmID, &models.Actor{ID: uuid.New()}))
}

// TestOperationTask_RunForOperation_DisabledRulesWin: выключенные правила — это
// осознанный отказ от задач, а не повод вернуться к конфигу.
func TestOperationTask_RunForOperation_DisabledRulesWin(t *testing.T) {
	ctx := context.Background()
	realmID, equipmentID := uuid.New(), uuid.New()
	repo := &fakeOperationTaskRepo{tasks: []*models.EquipmentOperationTask{operationTaskFixture(models.EquipmentOperationAssign, false)}}
	creator := &fakeTicketCreator{}
	svc := NewEquipmentOperationTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)

	assert.True(t, svc.RunForOperation(ctx, models.EquipmentOperationAssign, equipmentID, realmID, &models.Actor{ID: uuid.New()}))
	assert.Empty(t, creator.created)
}

// TestEquipmentService_SkipOperationTasks: импорт вызывает Assign с флагом
// пропуска, поэтому даже configured fallback не создаёт заявку в 1C:ERP.
func TestEquipmentService_SkipOperationTasks(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()
	actor := &models.Actor{ID: uuid.New()}
	creator := &fakeTicketCreator{}
	svc := NewEquipmentService(&fakeAssignRepo{unit: &models.Equipment{Name: "Ноутбук", SiteID: &siteID}},
		&mockTransactionManager{}, testEquipmentConf(t))
	svc.SetERPConfig(erpTestConfig())
	svc.SetTicketCreator(creator)

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SiteID: &siteID,
		SkipAct: true, SkipOperationTasks: true, Actor: actor,
	})
	require.NoError(t, err)
	assert.Empty(t, creator.created)
}

// TestOperationTask_CRUDValidate: обязательные поля правил проверяются на
// границе сервиса, а не только CHECK'ом в базе.
func TestOperationTask_CRUDValidate(t *testing.T) {
	ctx := context.Background()
	realmID := uuid.New()
	svc := NewEquipmentOperationTaskService(&fakeOperationTaskRepo{}, &mockTransactionManager{})

	base := func() *models.EquipmentOperationTaskDTO {
		return &models.EquipmentOperationTaskDTO{
			RealmID: realmID, Name: "Передача", Operation: models.EquipmentOperationAssign,
			CategoryID: uuid.New(), GroupID: uuid.New(),
		}
	}

	_, err := svc.CreateOperationTask(ctx, &models.EquipmentOperationTaskDTO{RealmID: realmID})
	assert.ErrorIs(t, err, models.ErrNotValid)

	created, err := svc.CreateOperationTask(ctx, base())
	require.NoError(t, err)
	assert.Equal(t, models.EquipmentActionCreateTicket, created.Action)
	assert.NotEqual(t, uuid.Nil, created.ID)

	_, err = svc.UpdateOperationTask(ctx, base())
	assert.ErrorIs(t, err, models.ErrNotValid)

	id := uuid.New()
	dto := base()
	dto.ID = &id
	_, err = svc.UpdateOperationTask(ctx, dto)
	require.NoError(t, err)

	err = svc.DeleteOperationTask(ctx, &models.DelEquipmentOperationTaskDTO{ID: id, RealmID: realmID})
	require.NoError(t, err)
}
