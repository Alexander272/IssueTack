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

// fakePeriodicRepo подменяет только методы, нужные проверке периодических задач
// оборудования; остальные методы интерфейса недостижимы (встраивает интерфейс).
type fakePeriodicRepo struct {
	repository.Equipment

	unit *models.Equipment

	templates map[uuid.UUID]*models.EquipmentTypePeriodicTask

	pending []*models.EquipmentPeriodicTaskInstance
	// claimResult / claimCalled — поведение атомарного claim pending → created.
	claimResult  bool
	claimCalled  bool
	revertCalled bool
	// ticketSet — результат SetPeriodicTaskInstanceTicket.
	ticketSetInstance *models.EquipmentPeriodicTaskInstance
	ticketSetTicketID  uuid.UUID
	ticketSetNextAt    time.Time

	createdInstances []*models.EquipmentPeriodicTaskInstance
	// rescheduleCalls — перезаписи расписания pending (manual run с shift).
	rescheduleCalls []struct {
		instanceID uuid.UUID
		scheduledAt time.Time
	}
}

func (f *fakePeriodicRepo) GetByID(_ context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	if f.unit == nil {
		return nil, models.ErrNotFound
	}
	unit := *f.unit
	unit.ID, unit.RealmID = req.ID, req.RealmID
	return &unit, nil
}

func (f *fakePeriodicRepo) GetPeriodicTaskTemplates(_ context.Context, _ uuid.UUID, _ *uuid.UUID) ([]*models.EquipmentTypePeriodicTask, error) {
	result := []*models.EquipmentTypePeriodicTask{}
	for _, t := range f.templates {
		result = append(result, t)
	}
	return result, nil
}

func (f *fakePeriodicRepo) GetPeriodicTaskTemplateByID(_ context.Context, _ postgres.Tx, id uuid.UUID, _ uuid.UUID) (*models.EquipmentTypePeriodicTask, error) {
	t, ok := f.templates[id]
	if !ok {
		return nil, models.ErrNotFound
	}
	return t, nil
}

func (f *fakePeriodicRepo) CreatePeriodicTaskTemplate(_ context.Context, _ postgres.Tx, _ *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error) {
	return &models.EquipmentTypePeriodicTask{}, nil
}

func (f *fakePeriodicRepo) GetPendingPeriodicTasks(_ context.Context, _ time.Time, _ int) ([]*models.EquipmentPeriodicTaskInstance, error) {
	return f.pending, nil
}

func (f *fakePeriodicRepo) CreatePeriodicTaskInstance(_ context.Context, _ postgres.Tx, instance *models.EquipmentPeriodicTaskInstance) error {
	f.createdInstances = append(f.createdInstances, instance)
	return nil
}

func (f *fakePeriodicRepo) UpdatePeriodicTaskInstance(_ context.Context, _ postgres.Tx, instance *models.EquipmentPeriodicTaskInstance) error {
	f.ticketSetInstance = instance
	return nil
}

func (f *fakePeriodicRepo) ClaimPeriodicTaskInstance(_ context.Context, _ postgres.Tx, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	f.claimCalled = true
	return f.claimResult, nil
}

func (f *fakePeriodicRepo) RevertPeriodicTaskClaim(_ context.Context, _ postgres.Tx, _ uuid.UUID, _ uuid.UUID, _ time.Time, _ *uuid.UUID) error {
	f.revertCalled = true
	return nil
}

func (f *fakePeriodicRepo) SetPeriodicTaskInstanceTicket(_ context.Context, _ postgres.Tx, instance *models.EquipmentPeriodicTaskInstance, ticketID uuid.UUID, nextScheduledAt time.Time) error {
	f.ticketSetInstance = instance
	f.ticketSetTicketID = ticketID
	f.ticketSetNextAt = nextScheduledAt
	return nil
}

func (f *fakePeriodicRepo) ReschedulePendingInstance(_ context.Context, _ postgres.Tx, instance *models.EquipmentPeriodicTaskInstance, scheduledAt time.Time) error {
	f.rescheduleCalls = append(f.rescheduleCalls, struct {
		instanceID  uuid.UUID
		scheduledAt time.Time
	}{instance.ID, scheduledAt})
	return nil
}

func (f *fakePeriodicRepo) HasPeriodicTaskInstances(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (f *fakePeriodicRepo) GetPendingPeriodicTaskInstance(_ context.Context, templateID, equipmentID, realmID uuid.UUID) (*models.EquipmentPeriodicTaskInstance, error) {
	for _, i := range f.pending {
		if i.TemplateID == templateID && i.EquipmentID == equipmentID && i.RealmID == realmID && i.Status == models.TaskInstancePending {
			return i, nil
		}
	}
	return nil, nil
}

func (f *fakePeriodicRepo) GetLastAssignmentForEquipment(_ context.Context, _ uuid.UUID) (*models.EquipmentAssignment, error) {
	return nil, nil
}

// TestEquipmentPeriodicTaskService_CreateTemplate_InvalidPeriod: период должен быть
// задан хотя бы одним ненулевым компонентом (годы/месяцы/дни).
func TestEquipmentPeriodicTaskService_CreateTemplate_InvalidPeriod(t *testing.T) {
	svc := NewEquipmentPeriodicTaskService(&fakePeriodicRepo{}, &mockTransactionManager{})

	_, err := svc.CreatePeriodicTaskTemplate(context.Background(), &models.EquipmentTypePeriodicTaskDTO{
		RealmID:         uuid.New(),
		EquipmentTypeID: uuid.New(),
		Title:           "Проверка оборудования",
		CategoryID:      uuid.New(),
		PeriodYears:     0,
		PeriodMonths:    0,
		PeriodDays:      0,
	})
	require.Error(t, err)

	_, err = svc.CreatePeriodicTaskTemplate(context.Background(), &models.EquipmentTypePeriodicTaskDTO{
		RealmID:         uuid.New(),
		EquipmentTypeID: uuid.New(),
		Title:           "Проверка оборудования",
		CategoryID:      uuid.New(),
		PeriodYears:     0,
		PeriodMonths:    3,
		PeriodDays:      0,
	})
	require.NoError(t, err)
}

// TestEquipmentPeriodicTaskService_ProcessPendingTask_CreatesTicket: pending → claim →
// заявка (создателем = автор шаблона) → следующий pending на расписании + период.
func TestEquipmentPeriodicTaskService_ProcessPendingTask_CreatesTicket(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	templateID := uuid.New()
	categoryID := uuid.New()
	equipmentID := uuid.New()
	assignmentID := uuid.New()

	template := &models.EquipmentTypePeriodicTask{
		ID:                 templateID,
		RealmID:            realmID,
		EquipmentTypeID:    uuid.New(),
		Title:              "Проверка оборудования",
		Description:        "Проверить исправность",
		PeriodYears:        2,
		CategoryID:         categoryID,
		Priority:           "medium",
		AutoStartOnFirstAssign: false,
		Enabled:            true,
		CreatedBy:          &authorID,
	}
	eq := &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", InventoryNumber: strPtr("000001")}
	scheduledAt := time.Now().UTC().AddDate(-2, 0, 0)

	repo := &fakePeriodicRepo{
		unit:       eq,
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{templateID: template},
		pending: []*models.EquipmentPeriodicTaskInstance{{
			ID:               uuid.New(),
			RealmID:          realmID,
			TemplateID:       templateID,
			EquipmentID:      equipmentID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      scheduledAt,
			LastAssignmentID: &assignmentID,
			CreatedBy:        &authorID,
		}},
		claimResult: true,
	}
	creator := &fakeTicketCreator{}
	cats := &fakeCategoriesSvc{}
	cats.getByID = func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		return &models.Category{ID: req.ID, Name: "IT", GroupID: uuid.New()}, nil
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)
	svc.SetCategories(cats)

	require.NoError(t, svc.ProcessPendingTasks(ctx))

	require.Len(t, creator.created, 1)
	ticket := creator.created[0]
	require.NotNil(t, ticket.ID)
	assert.Equal(t, authorID, ticket.CreatorID)
	require.NotNil(t, ticket.OwnerID)
	assert.Equal(t, authorID, *ticket.OwnerID)
	require.NotNil(t, ticket.GroupID)
	assert.NotEqual(t, uuid.Nil, *ticket.GroupID)
	require.NotNil(t, ticket.EquipmentID)
	assert.Equal(t, equipmentID, *ticket.EquipmentID)
	assert.Equal(t, authorID, ticket.Actor.ID)
	assert.True(t, ticket.Status == models.StatusOpen || ticket.Status == "")
	assert.Equal(t, models.Priority("medium"), ticket.Priority)

	// Заявка привязана к созданному экземпляру, следующий старт — от расписания + период.
	require.NotNil(t, repo.ticketSetInstance)
	assert.Equal(t, *ticket.ID, repo.ticketSetTicketID)
	nextAt := scheduledAt.AddDate(2, 0, 0)
	assert.True(t, repo.ticketSetNextAt.Equal(nextAt), "next scheduled at %v, want %v", repo.ticketSetNextAt, nextAt)
	require.Len(t, repo.createdInstances, 1)
	assert.True(t, repo.createdInstances[0].ScheduledAt.Equal(nextAt))
}

// TestEquipmentPeriodicTaskService_ProcessPendingTask_ClaimFail_Skips: если claim не
// удался (уже обработан конкурентным тиком) — заявка не создаётся.
func TestEquipmentPeriodicTaskService_ProcessPendingTask_ClaimFail_Skips(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	templateID := uuid.New()
	categoryID := uuid.New()
	equipmentID := uuid.New()
	assignmentID := uuid.New()

	template := &models.EquipmentTypePeriodicTask{
		ID:                 templateID,
		RealmID:            realmID,
		EquipmentTypeID:    uuid.New(),
		Title:              "Проверка оборудования",
		Description:        "Проверить исправность",
		PeriodYears:        2,
		CategoryID:         categoryID,
		Priority:           "medium",
		Enabled:            true,
		CreatedBy:          &authorID,
	}
	eq := &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", InventoryNumber: strPtr("000001")}
	scheduledAt := time.Now().UTC().AddDate(-2, 0, 0)

	repo := &fakePeriodicRepo{
		unit:       eq,
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{templateID: template},
		pending: []*models.EquipmentPeriodicTaskInstance{{
			ID:               uuid.New(),
			RealmID:          realmID,
			TemplateID:       templateID,
			EquipmentID:      equipmentID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      scheduledAt,
			LastAssignmentID: &assignmentID,
			CreatedBy:        &authorID,
		}},
		claimResult: false,
	}
	creator := &fakeTicketCreator{}
	cats := &fakeCategoriesSvc{}
	cats.getByID = func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		return &models.Category{ID: req.ID, Name: "IT", GroupID: uuid.New()}, nil
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)
	svc.SetCategories(cats)

	require.NoError(t, svc.ProcessPendingTasks(ctx))
	assert.Empty(t, creator.created)
	assert.False(t, repo.revertCalled)
}

// TestEquipmentPeriodicTaskService_ProcessPendingTask_TicketFail_RevertsClaim: сбой
// создания заявки откатывает claim — задача не потеряется, повторится на следующем тике.
func TestEquipmentPeriodicTaskService_ProcessPendingTask_TicketFail_RevertsClaim(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	templateID := uuid.New()
	categoryID := uuid.New()
	equipmentID := uuid.New()
	assignmentID := uuid.New()

	template := &models.EquipmentTypePeriodicTask{
		ID:                 templateID,
		RealmID:            realmID,
		EquipmentTypeID:    uuid.New(),
		Title:              "Проверка оборудования",
		Description:        "Проверить исправность",
		PeriodYears:        2,
		CategoryID:         categoryID,
		Priority:           "medium",
		Enabled:            true,
		CreatedBy:          &authorID,
	}
	eq := &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", InventoryNumber: strPtr("000001")}
	scheduledAt := time.Now().UTC().AddDate(-2, 0, 0)

	repo := &fakePeriodicRepo{
		unit:       eq,
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{templateID: template},
		pending: []*models.EquipmentPeriodicTaskInstance{{
			ID:               uuid.New(),
			RealmID:          realmID,
			TemplateID:       templateID,
			EquipmentID:      equipmentID,
			Status:           models.TaskInstancePending,
			ScheduledAt:      scheduledAt,
			LastAssignmentID: &assignmentID,
			CreatedBy:        &authorID,
		}},
		claimResult: true,
	}
	creator := &fakeTicketCreator{err: assert.AnError}
	cats := &fakeCategoriesSvc{}
	cats.getByID = func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		return &models.Category{ID: req.ID, Name: "IT", GroupID: uuid.New()}, nil
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)
	svc.SetCategories(cats)

	assert.NoError(t, svc.ProcessPendingTasks(ctx))
	assert.True(t, repo.revertCalled)
	assert.Nil(t, repo.ticketSetInstance)
}

// TestEquipmentPeriodicTaskService_ManualRun_ShiftReschedulesChain: ручной запуск с
// shift=true создаёт заявку сейчас и перепланирует следующий старт на now + период.
func TestEquipmentPeriodicTaskService_ManualRun_ShiftReschedulesChain(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	templateID := uuid.New()
	categoryID := uuid.New()
	equipmentID := uuid.New()
	assignmentID := uuid.New()
	actorID := uuid.New()

	template := &models.EquipmentTypePeriodicTask{
		ID:                 templateID,
		RealmID:            realmID,
		EquipmentTypeID:    uuid.New(),
		Title:              "Проверка оборудования",
		Description:        "Проверить исправность",
		PeriodYears:        0,
		PeriodMonths:       0,
		PeriodDays:         7,
		CategoryID:         categoryID,
		Priority:           "medium",
		Enabled:            true,
		CreatedBy:          &authorID,
	}
	eq := &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", InventoryNumber: strPtr("000001")}
	pending := &models.EquipmentPeriodicTaskInstance{
		ID:               uuid.New(),
		RealmID:          realmID,
		TemplateID:       templateID,
		EquipmentID:      equipmentID,
		Status:           models.TaskInstancePending,
		ScheduledAt:      time.Now().UTC().AddDate(0, 0, -3),
		LastAssignmentID: &assignmentID,
		CreatedBy:        &authorID,
	}

	repo := &fakePeriodicRepo{
		unit:       eq,
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{templateID: template},
		pending:    []*models.EquipmentPeriodicTaskInstance{pending},
	}
	creator := &fakeTicketCreator{}
	cats := &fakeCategoriesSvc{}
	cats.getByID = func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		return &models.Category{ID: req.ID, Name: "IT", GroupID: uuid.New()}, nil
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)
	svc.SetCategories(cats)

	err := svc.ManualRunTask(ctx, &models.ManualPeriodicTaskRunDTO{
		RealmID:     realmID,
		TemplateID:  templateID,
		EquipmentID: equipmentID,
		ShiftNext:   true,
	}, &models.Actor{ID: actorID, Name: "Ручной запуск"})
	require.NoError(t, err)
	require.Len(t, creator.created, 1)

	// Расписание заменено на «сейчас»: ручной запуск заменяет ближайший плановый.
	require.Len(t, repo.rescheduleCalls, 1)
	assert.WithinDuration(t, time.Now().UTC(), repo.rescheduleCalls[0].scheduledAt, time.Minute)
	assert.Equal(t, pending.ID, repo.rescheduleCalls[0].instanceID)

	// Заявка создана от имени инициатора ручного запуска.
	ticket := creator.created[0]
	assert.Equal(t, actorID, ticket.CreatorID)

	// Следующий автоматический старт — ровно через период (now + 7 дней), а не через два.
	require.Len(t, repo.createdInstances, 1)
	wantNext := time.Now().UTC().AddDate(0, 0, 7)
	assert.WithinDuration(t, wantNext, repo.createdInstances[0].ScheduledAt, time.Minute)
}

// TestEquipmentPeriodicTaskService_ManualRun_NoPending_CreatesChainStart: при отсутствии
// pending ручной запуск создаёт старт цепочки (now) и заявку сейчас.
func TestEquipmentPeriodicTaskService_ManualRun_NoPending_CreatesChainStart(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	templateID := uuid.New()
	categoryID := uuid.New()
	equipmentID := uuid.New()
	actorID := uuid.New()

	template := &models.EquipmentTypePeriodicTask{
		ID:                 templateID,
		RealmID:            realmID,
		EquipmentTypeID:    uuid.New(),
		Title:              "Проверка оборудования",
		Description:        "Проверить испправность",
		PeriodYears:        0,
		PeriodMonths:       0,
		PeriodDays:         7,
		CategoryID:         categoryID,
		Priority:           "medium",
		Enabled:            true,
		CreatedBy:          &authorID,
	}
	eq := &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", InventoryNumber: strPtr("000001")}

	repo := &fakePeriodicRepo{
		unit:       eq,
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{templateID: template},
	}
	creator := &fakeTicketCreator{}
	cats := &fakeCategoriesSvc{}
	cats.getByID = func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		return &models.Category{ID: req.ID, Name: "IT", GroupID: uuid.New()}, nil
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})
	svc.SetTicketCreator(creator)
	svc.SetCategories(cats)

	err := svc.ManualRunTask(ctx, &models.ManualPeriodicTaskRunDTO{
		RealmID:     realmID,
		TemplateID:  templateID,
		EquipmentID: equipmentID,
		ShiftNext:   false,
	}, &models.Actor{ID: actorID, Name: "Ручной запуск"})
	require.NoError(t, err)
	require.Len(t, creator.created, 1)

	// Старт цепочки создан на now, следующий — через период: initial + next.
	require.Len(t, repo.createdInstances, 2)
	wantNext := time.Now().UTC().AddDate(0, 0, 7)
	assert.WithinDuration(t, wantNext, repo.createdInstances[len(repo.createdInstances)-1].ScheduledAt, time.Minute)
}

// TestEquipmentPeriodicTaskService_StartPeriodicTasksForAssignment: при выдаче единицы
// автостарт создаёт pending на дату выдачи + период, если цепочка ещё не заводилась.
func TestEquipmentPeriodicTaskService_StartPeriodicTasksForAssignment(t *testing.T) {
	ctx := context.Background()
	authorID := uuid.New()
	realmID := uuid.New()
	equipmentID := uuid.New()
	assignedAt := time.Now().UTC().AddDate(-1, 0, 0)

	repo := &fakePeriodicRepo{
		unit: &models.Equipment{ID: equipmentID, RealmID: realmID, Name: "Ноутбук", TypeID: ptrUUID(uuid.New())},
		templates: map[uuid.UUID]*models.EquipmentTypePeriodicTask{uuid.New(): {
			ID:                 uuid.New(),
			RealmID:            realmID,
			EquipmentTypeID:    uuid.New(),
			Title:              "Проверка оборудования",
			Description:        "Проверить исправность",
			PeriodYears:        1,
			CategoryID:         uuid.New(),
			Priority:           "medium",
			AutoStartOnFirstAssign: true,
			Enabled:            true,
			CreatedBy:          &authorID,
		}},
	}
	svc := NewEquipmentPeriodicTaskService(repo, &mockTransactionManager{})

	require.NoError(t, svc.StartPeriodicTasksForAssignment(ctx, equipmentID, realmID, assignedAt))
	require.Len(t, repo.createdInstances, 1)
	wantStart := assignedAt.AddDate(1, 0, 0)
	assert.True(t, repo.createdInstances[0].ScheduledAt.Equal(wantStart))
	assert.Equal(t, models.TaskInstancePending, repo.createdInstances[0].Status)
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

func strPtr(s string) *string { return &s }
