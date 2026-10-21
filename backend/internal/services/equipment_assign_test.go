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

// fakeAssignRepo подменяет только методы, нужные выдаче: саму единицу, её открытую
// выдачу и подписи справочников для снимка события.
type fakeAssignRepo struct {
	repository.Equipment

	unit     *models.Equipment
	siteName string
	// openAssignment — текущая открытая строка единицы; nil — единица на складе.
	openAssignment *models.EquipmentAssignment

	createdAssignment *models.EquipmentAssignment
	closedIDs         []uuid.UUID
	events            []*models.EquipmentEvent
	// createdAct — акт, сформированный выдачей (createActTx); nil при SkipAct.
	createdAct *models.EquipmentAct
	holder     struct {
		userID, siteID *uuid.UUID
		location       *string
	}
}

func (f *fakeAssignRepo) GetByID(_ context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	unit := *f.unit
	unit.ID, unit.RealmID = req.ID, req.RealmID
	return &unit, nil
}

func (f *fakeAssignRepo) GetChildren(_ context.Context, _, _ uuid.UUID) ([]*models.Equipment, error) {
	return nil, nil
}

func (f *fakeAssignRepo) GetUserFullName(_ context.Context, _ uuid.UUID) (string, error) {
	return "Мартынов Александр", nil
}

func (f *fakeAssignRepo) GetSiteName(_ context.Context, _ uuid.UUID) (string, error) {
	return f.siteName, nil
}

func (f *fakeAssignRepo) GetOpenAssignment(_ context.Context, _ uuid.UUID) (*models.EquipmentAssignment, error) {
	if f.openAssignment == nil {
		return nil, models.ErrNotFound
	}
	return f.openAssignment, nil
}

func (f *fakeAssignRepo) CloseAssignment(_ context.Context, _ postgres.Tx, id uuid.UUID, _ time.Time, _ string) error {
	f.closedIDs = append(f.closedIDs, id)
	return nil
}

func (f *fakeAssignRepo) CreateAssignment(_ context.Context, _ postgres.Tx, a *models.EquipmentAssignment) error {
	f.createdAssignment = a
	return nil
}

func (f *fakeAssignRepo) GetLastStoredPlace(_ context.Context, _ uuid.UUID) (*uuid.UUID, *string, error) {
	return nil, nil, models.ErrNotFound
}

func (f *fakeAssignRepo) SetHolderAndStatus(
	_ context.Context, _ postgres.Tx, _ uuid.UUID,
	userID *uuid.UUID, _ *string, siteID *uuid.UUID, location *string, _ models.EquipmentStatus,
) error {
	f.holder.userID, f.holder.siteID, f.holder.location = userID, siteID, location
	return nil
}

func (f *fakeAssignRepo) CreateEvent(_ context.Context, _ postgres.Tx, e *models.EquipmentEvent) error {
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAssignRepo) NextActSeq(_ context.Context, _ postgres.Tx, _ uuid.UUID) (int, error) {
	return 1, nil
}

func (f *fakeAssignRepo) CreateAct(_ context.Context, _ postgres.Tx, a *models.EquipmentAct) error {
	f.createdAct = a
	return nil
}

// TestEquipmentService_Assign_ActCarriesRecipientName закрепляет причину прочерка
// в акте: при выборе сотрудника из справочника holderName пуст (ФИО приходит
// отдельным полем recipientName), и раньше в акт уходило именно пустое holderName —
// в карточке и docx получатель показывался прочерком. Теперь снимок имени берётся
// из recipientName, а без него — из свободного ФИО.
func TestEquipmentService_Assign_ActCarriesRecipientName(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()
	recipient := "Мартынов Александр"

	cases := []struct {
		name     string
		dto      models.AssignEquipmentDTO
		wantName string
	}{
		{
			name: "сотрудник из справочника",
			dto:  models.AssignEquipmentDTO{UserID: &userID, RecipientName: &recipient},
			// holderName пуст — акт обязан взять ФИО из recipientName.
			wantName: recipient,
		},
		{
			name:     "свободное ФИО внешнего",
			dto:      models.AssignEquipmentDTO{HolderName: &recipient, RecipientName: &recipient},
			wantName: recipient,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAssignRepo{unit: &models.Equipment{SiteID: &siteID}}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			dto := tt.dto
			dto.ID, dto.RealmID = uuid.New(), uuid.New()

			require.NoError(t, svc.Assign(ctx, &dto))
			require.NotNil(t, repo.createdAct)
			require.NotNil(t, repo.createdAct.RecipientName)
			assert.Equal(t, tt.wantName, *repo.createdAct.RecipientName)
		})
	}
}

// TestEquipmentService_Assign_KeepsPlaceWhenNotProvided закрепляет: пустые
// «Площадка»/«Локация» в диалоге означают «единица не переезжает». Раньше
// SetHolderAndStatus записывал по ним nil, и выдача без заполненной площадки
// стирала место единицы — вместе с подписью площадки в снимке события и в акте.
func TestEquipmentService_Assign_KeepsPlaceWhenNotProvided(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()
	location := "Склад, полка A"

	repo := &fakeAssignRepo{
		unit:     &models.Equipment{SiteID: &siteID, Location: &location},
		siteName: "ППРЗ",
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SkipAct: true,
	})
	require.NoError(t, err)

	// Место осталось прежним и в самой единице, и в строке выдачи.
	require.NotNil(t, repo.holder.userID)
	assert.Equal(t, userID, *repo.holder.userID)
	require.NotNil(t, repo.holder.siteID)
	assert.Equal(t, siteID, *repo.holder.siteID)
	require.NotNil(t, repo.holder.location)
	assert.Equal(t, location, *repo.holder.location)

	require.NotNil(t, repo.createdAssignment)
	require.NotNil(t, repo.createdAssignment.SiteID)
	assert.Equal(t, siteID, *repo.createdAssignment.SiteID)

	// Снимок события тоже несёт площадку: лента истории не тянет справочники.
	require.Len(t, repo.events, 1)
	assert.Contains(t, string(repo.events[0].NewValue), `"siteName":"ППРЗ"`)
}

// TestEquipmentService_Assign_ExplicitPlaceWins: указанное место важнее текущего —
// единицу могут выдать сразу на другой объект.
func TestEquipmentService_Assign_ExplicitPlaceWins(t *testing.T) {
	ctx := context.Background()
	currentSiteID, newSiteID, userID := uuid.New(), uuid.New(), uuid.New()

	repo := &fakeAssignRepo{unit: &models.Equipment{SiteID: &currentSiteID}}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SiteID: &newSiteID, SkipAct: true,
	})
	require.NoError(t, err)
	require.NotNil(t, repo.holder.siteID)
	assert.Equal(t, newSiteID, *repo.holder.siteID)
}

// TestEquipmentService_Assign_StoredRowDoesNotBlock: после возврата открытая строка
// остаётся без держателя, и выдача со склада не должна считаться «выдано другому».
// Без оговорки конфликта по чужому держателю повторная выдача требовала бы force.
func TestEquipmentService_Assign_StoredRowDoesNotBlock(t *testing.T) {
	ctx := context.Background()
	userID, openID := uuid.New(), uuid.New()

	repo := &fakeAssignRepo{
		unit:           &models.Equipment{},
		openAssignment: &models.EquipmentAssignment{ID: openID},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SkipAct: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{openID}, repo.closedIDs)
}

// TestEquipmentService_Assign_Conflicts: выдача блокируется только чужим держателем.
// Тот же держатель с исправленным местом конфликтом не считается (место — реквизит
// единицы), а force снимает конфликт.
func TestEquipmentService_Assign_Conflicts(t *testing.T) {
	ctx := context.Background()
	userID, otherID := uuid.New(), uuid.New()
	otherName := "Петров Пётр"

	cases := []struct {
		name    string
		open    *models.EquipmentAssignment
		dtoUser *uuid.UUID
		force   bool
		wantErr bool
	}{
		{
			name:    "другой держатель",
			open:    &models.EquipmentAssignment{ID: uuid.New(), UserID: &otherID, HolderName: &otherName},
			dtoUser: &userID,
			wantErr: true,
		},
		{
			name:    "свободный текст вместо сотрудника",
			open:    &models.EquipmentAssignment{ID: uuid.New(), HolderName: &otherName},
			dtoUser: &userID,
			wantErr: true,
		},
		{
			name:    "тот же держатель",
			open:    &models.EquipmentAssignment{ID: uuid.New(), UserID: &userID},
			dtoUser: &userID,
		},
		{
			name:    "другой держатель, force",
			open:    &models.EquipmentAssignment{ID: uuid.New(), UserID: &otherID},
			dtoUser: &userID,
			force:   true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			siteID := uuid.New()
			repo := &fakeAssignRepo{
				unit:           &models.Equipment{SiteID: &siteID},
				openAssignment: tt.open,
			}
			svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

			err := svc.Assign(ctx, &models.AssignEquipmentDTO{
				ID: uuid.New(), RealmID: uuid.New(), UserID: tt.dtoUser, Force: tt.force, SkipAct: true,
			})
			if tt.wantErr {
				assert.ErrorIs(t, err, models.ErrEquipmentAssigned)
				assert.Empty(t, repo.createdAssignment)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, repo.createdAssignment)
		})
	}
}

// TestEquipmentService_Assign_RequiresHolderOrPlace: выдача без держателя и без
// места не имеет смысла и отклоняется — как и раньше. Пустое место само по себе
// ошибкой не является: оно значит «единица не переезжает».
func TestEquipmentService_Assign_RequiresHolderOrPlace(t *testing.T) {
	ctx := context.Background()
	siteID := uuid.New()

	repo := &fakeAssignRepo{unit: &models.Equipment{SiteID: &siteID}}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{ID: uuid.New(), RealmID: uuid.New(), SkipAct: true})
	assert.ErrorIs(t, err, models.ErrEquipmentNoHoldersOrLocation)
	assert.Empty(t, repo.createdAssignment)
	assert.Empty(t, repo.events)
}

// TestEquipmentService_Assign_EventLinksRow: событие выдачи называет строку выдачи
// явно (assignment_id) и прежнюю открытую строку, которую эта же операция закрыла
// (closed_assignment_id). Лента истории соединяет строку и журнал по этим
// ссылкам: момент выдачи приходит из формы (её оформляют задним числом), а
// момент записи в журнал — другое, и по времени их не отличить.
func TestEquipmentService_Assign_EventLinksRow(t *testing.T) {
	ctx := context.Background()
	siteID, userID, openID := uuid.New(), uuid.New(), uuid.New()

	repo := &fakeAssignRepo{
		unit:           &models.Equipment{SiteID: &siteID},
		openAssignment: &models.EquipmentAssignment{ID: openID},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SkipAct: true,
	})
	require.NoError(t, err)

	require.Len(t, repo.events, 1)
	ev := repo.events[0]
	require.NotNil(t, repo.createdAssignment)

	// Событие открыло новую строку выдачи.
	require.NotNil(t, ev.AssignmentID)
	assert.Equal(t, repo.createdAssignment.ID, *ev.AssignmentID)
	// И закрыло прежнюю открытую.
	require.NotNil(t, ev.ClosedAssignmentID)
	assert.Equal(t, openID, *ev.ClosedAssignmentID)
}

// TestEquipmentService_Assign_EventWithoutPreviousRow: первой выдаче нечего
// закрывать — ссылка на закрытую строку пуста, а не указывает на случайную.
func TestEquipmentService_Assign_EventWithoutPreviousRow(t *testing.T) {
	ctx := context.Background()
	siteID, userID := uuid.New(), uuid.New()

	repo := &fakeAssignRepo{unit: &models.Equipment{SiteID: &siteID}}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Assign(ctx, &models.AssignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), UserID: &userID, SkipAct: true,
	})
	require.NoError(t, err)

	require.Len(t, repo.events, 1)
	ev := repo.events[0]
	require.NotNil(t, ev.AssignmentID)
	require.NotNil(t, repo.createdAssignment)
	assert.Equal(t, repo.createdAssignment.ID, *ev.AssignmentID)
	assert.Nil(t, ev.ClosedAssignmentID)
}

// TestEquipmentService_Unassign_EventLinksRows: возврат меняет две строки —
// закрывает выдачу и открывает строку «на складе», — и событие называет обе:
// прежняя выдача в closed_assignment_id, складская строка в assignment_id.
func TestEquipmentService_Unassign_EventLinksRows(t *testing.T) {
	ctx := context.Background()
	siteID, openID := uuid.New(), uuid.New()

	repo := &fakeAssignRepo{
		unit:           &models.Equipment{SiteID: &siteID},
		openAssignment: &models.EquipmentAssignment{ID: openID},
	}
	svc := NewEquipmentService(repo, &mockTransactionManager{}, testEquipmentConf(t))

	err := svc.Unassign(ctx, &models.UnassignEquipmentDTO{
		ID: uuid.New(), RealmID: uuid.New(), SiteID: &siteID,
	})
	require.NoError(t, err)

	require.Len(t, repo.events, 1)
	ev := repo.events[0]
	require.NotNil(t, repo.createdAssignment)

	require.NotNil(t, ev.AssignmentID)
	assert.Equal(t, repo.createdAssignment.ID, *ev.AssignmentID)
	require.NotNil(t, ev.ClosedAssignmentID)
	assert.Equal(t, openID, *ev.ClosedAssignmentID)
}
