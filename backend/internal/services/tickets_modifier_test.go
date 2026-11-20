package services

import (
	"context"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// modifierFixture — минимальный набор зависимостей для правил подкатегории:
// сам справочник, категории, проверка роли начальника области и группы.
func modifierFixture() (*TicketService, *MockCategoryModifiers, *MockCategoriesRepo, *MockAccessPolicies) {
	modifiers := new(MockCategoryModifiers)
	categories := new(MockCategoriesRepo)
	policies := new(MockAccessPolicies)
	groups := new(MockGroupsRepo)
	svc := NewTicketService(&TicketDeps{
		Modifiers:  modifiers,
		Categories: categories,
		Groups:     groups,
		Access:     NewTicketAccessService(new(MockTicketsRepo), groups, policies),
	})
	return svc, modifiers, categories, policies
}

// expectSupervisor настраивает ответ IsRealmSupervisor (category:write → site:write).
func expectSupervisor(policies *MockAccessPolicies, userID uuid.UUID, realm string, yes bool) {
	policies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(yes, nil)
	if !yes {
		policies.On("Enforce", userID.String(), realm, string(access.ResourceSite), string(access.Write)).Return(false, nil)
	}
}

// TestTicketModifier_ApplyValues_ExplicitPriorityKept закрепляет право
// начальника области, менеджера группы и исполнителя-создателя переопределять
// и срок, и приоритет после выбора подкатегории. Раньше приоритет затирался
// всегда — вопреки правилу «может изменить срок и приоритет».
func TestTicketModifier_ApplyValues_ExplicitPriorityKept(t *testing.T) {
	days := 3
	modifier := &models.CategoryModifier{
		Priority:       ptrPriority(models.PriorityUrgent),
		DefaultDueDays: &days,
	}
	dto := &models.TicketDTO{Priority: models.PriorityLow}
	dto.MarkProvided("priority")
	explicitDue := time.Now().UTC().Add(72 * time.Hour)
	dto.DueDate = &explicitDue
	dto.MarkProvided("dueDate")

	require.NoError(t, applyModifierValues(&modifierValuesInput{
		DTO: dto, Modifier: modifier, MayOverride: true, Base: time.Now().UTC(),
	}))
	assert.Equal(t, models.PriorityLow, dto.Priority, "явный приоритет должен сохраниться")
	assert.Equal(t, &explicitDue, dto.DueDate, "явный срок должен сохраниться")
}

// TestTicketModifier_ApplyValues_InheritCategory проверяет, что NULL у
// подкатегории наследуется от категории: и приоритет, и срок.
func TestTicketModifier_ApplyValues_InheritCategory(t *testing.T) {
	days := 5
	modifier := &models.CategoryModifier{} // оба параметра не заданы
	category := &models.Category{Priority: models.PriorityHigh, DefaultDueDays: &days}
	dto := &models.TicketDTO{}

	base := time.Now().UTC()
	require.NoError(t, applyModifierValues(&modifierValuesInput{
		DTO: dto, Modifier: modifier, Category: category, Base: base,
	}))
	assert.Equal(t, models.PriorityHigh, dto.Priority)
	require.NotNil(t, dto.DueDate)
	assert.Equal(t, addWorkingDaysMonFri(base, days, time.Local), *dto.DueDate)
	assert.True(t, dto.HasField("priority"))
	assert.True(t, dto.HasField("dueDate"))
}

// TestTicketModifier_DueDateFromCreatedAt закрепляет, что при смене и при
// добавлении подкатегории срок считается от created_at заявки, а не от момента
// правки. Иначе поздний выбор подкатегории сдвигал бы срок вперёд.
func TestTicketModifier_DueDateFromCreatedAt(t *testing.T) {
	actor := uuid.New()
	realm := uuid.New()
	categoryID := uuid.New()
	oldModifier := uuid.New()
	newModifier := uuid.New()
	created := time.Now().UTC().AddDate(0, 0, -10)
	days := 3

	cases := []struct {
		name        string
		oldModifier *uuid.UUID
	}{
		{name: "change", oldModifier: &oldModifier},
		{name: "add", oldModifier: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, modifiers, categories, policies := modifierFixture()
			old := &models.Ticket{
				ID:               uuid.New(),
				RealmID:          &realm,
				CreatedAt:        created,
				Status:           models.StatusOpen,
				Creator:          models.UserShort{ID: actor},
				Assignee:         &models.UserShort{ID: actor},
				Category:         &models.CategoryShort{ID: categoryID},
				CategoryModifier: &models.CategoryModifierShort{ID: oldModifier},
			}
			if tc.oldModifier == nil {
				old.CategoryModifier = nil
			}
			dto := &models.TicketDTO{ID: &old.ID, Actor: &models.Actor{ID: actor}, CategoryModifierID: &newModifier}
			dto.MarkProvided("categoryModifierId")
			expectSupervisor(policies, actor, realm.String(), true)

			modifiers.On("GetByID", mock.Anything, &models.GetCategoryModifierByIDDTO{ID: newModifier, RealmID: realm}).
				Return(&models.CategoryModifier{ID: newModifier, CategoryID: categoryID, IsActive: true,
					Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days}, nil)
			categories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: realm}).
				Return(&models.Category{ID: categoryID, Priority: models.PriorityMedium}, nil)

			result, err := svc.prepareUpdateModifier(context.Background(), &updateModifierInput{DTO: dto, Old: old, ActorID: actor})
			require.NoError(t, err)
			assert.True(t, result.Applied)
			require.NotNil(t, dto.DueDate)
			assert.Equal(t, addWorkingDaysMonFri(created, days, time.Local), *dto.DueDate,
				"срок считается от даты создания заявки (рабочие дни, конец дня)")
			assert.True(t, dto.DueDate.Before(time.Now().UTC()),
				"срок не должен считаться от момента смены подкатегории")
		})
	}
}

// TestTicketModifier_PrepareCreate проверяет выбор подкатегории при создании:
// исполнитель (участник группы) или начальник области. Право не зависит от того,
// кто выбран исполнителем новой заявки — отдельный кейс executor_other_assignee
// закрепляет, что исполнитель видит поле и при чужом assignee. Дополнительно
// проверяются разовая подстановка параметров, отказ на неактивной и чужой
// подкатегории.
func TestTicketModifier_PrepareCreate(t *testing.T) {
	actor := uuid.New()
	realm := uuid.New()
	categoryID := uuid.New()
	modifierID := uuid.New()
	otherAssignee := uuid.New()
	explicitDue := time.Now().UTC().Add(96 * time.Hour)
	days := 2

	cases := []struct {
		name         string
		supervisor   bool
		isExecutor   bool
		assignee     *uuid.UUID
		explicit     bool
		modifier     *models.CategoryModifier
		wantErr      error
		wantPriority models.Priority
		wantDue      *time.Time
		wantLookup   bool
	}{
		{
			name:         "supervisor_explicit_values_kept",
			supervisor:   true,
			assignee:     &otherAssignee,
			explicit:     true,
			modifier:     &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days},
			wantPriority: models.PriorityLow,
			wantDue:      &explicitDue,
			wantLookup:   true,
		},
		{
			name:         "supervisor_modifier_values_applied",
			supervisor:   true,
			modifier:     &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days},
			wantPriority: models.PriorityUrgent,
			wantLookup:   true,
		},
		{
			name:         "executor_self_assigned_explicit_values_kept",
			isExecutor:   true,
			assignee:     &actor,
			explicit:     true,
			modifier:     &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days},
			wantPriority: models.PriorityLow,
			wantDue:      &explicitDue,
			wantLookup:   true,
		},
		{
			name:         "executor_self_assigned_modifier_values_applied",
			isExecutor:   true,
			assignee:     &actor,
			modifier:     &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days},
			wantPriority: models.PriorityUrgent,
			wantLookup:   true,
		},
		{
			// Ключевой кейс уточнения: исполнитель выбирает подкатегорию, даже если
			// исполнителем новой заявки будет другой пользователь.
			name:         "executor_other_assignee_allowed",
			isExecutor:   true,
			assignee:     &otherAssignee,
			modifier:     &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days},
			wantPriority: models.PriorityUrgent,
			wantLookup:   true,
		},
		{
			name:    "customer_denied",
			wantErr: models.ErrPermissionDenied,
		},
		{
			name:       "inactive_denied",
			isExecutor: true,
			modifier:   &models.CategoryModifier{ID: modifierID, CategoryID: categoryID, IsActive: false},
			wantErr:    models.ErrInvalidInput,
			wantLookup: true,
		},
		{
			name:       "foreign_category_denied",
			isExecutor: true,
			modifier:   &models.CategoryModifier{ID: modifierID, CategoryID: uuid.New(), IsActive: true},
			wantErr:    models.ErrInvalidInput,
			wantLookup: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, modifiers, categories, _ := modifierFixture()
			dto := &models.TicketDTO{
				RealmID:            &realm,
				Actor:              &models.Actor{ID: actor},
				CreatorID:          actor,
				AssigneeID:         tc.assignee,
				CategoryID:         categoryID,
				CategoryModifierID: &modifierID,
				Priority:           models.PriorityLow,
			}
			if tc.explicit {
				dto.MarkProvided("priority")
				dto.DueDate = &explicitDue
				dto.MarkProvided("dueDate")
			}
			if tc.wantLookup {
				modifiers.On("GetByID", mock.Anything, &models.GetCategoryModifierByIDDTO{ID: modifierID, RealmID: realm}).Return(tc.modifier, nil)
			}
			if tc.wantErr == nil && tc.wantLookup {
				categories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: realm}).
					Return(&models.Category{ID: categoryID, Priority: models.PriorityMedium, DefaultDueDays: &days}, nil)
			}

			before := time.Now()
			err := svc.prepareCreateModifier(context.Background(), &createModifierInput{
				DTO: dto, Supervisor: tc.supervisor, IsExecutor: tc.isExecutor,
			})
			after := time.Now()
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				if !tc.wantLookup {
					modifiers.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantPriority, dto.Priority)
			if tc.wantDue == nil {
				require.NotNil(t, dto.DueDate)
				expectedA := addWorkingDaysMonFri(before, days, time.Local)
				expectedB := addWorkingDaysMonFri(after, days, time.Local)
				assert.True(t, dto.DueDate.Equal(expectedA) || dto.DueDate.Equal(expectedB),
					"due=%v expected %v or %v", *dto.DueDate, expectedA, expectedB)
			} else {
				assert.Equal(t, tc.wantDue, dto.DueDate)
			}
		})
	}
}

// TestTicketModifier_PrepareUpdate_ResetsOnCategoryChange закрепляет «Да, сброс
// при смене»: сервер игнорирует присланный modifier id и снимает подкатегорию,
// если в том же обновлении поменялась категория. Эхо того же id сменой не считается.
func TestTicketModifier_PrepareUpdate_ResetsOnCategoryChange(t *testing.T) {
	svc, modifiers, _, policies := modifierFixture()
	actor := uuid.New()
	realm := uuid.New()
	oldCategory := uuid.New()
	newCategory := uuid.New()
	oldModifier := uuid.New()
	staleModifier := uuid.New()

	old := &models.Ticket{
		ID:               uuid.New(),
		RealmID:          &realm,
		Status:           models.StatusOpen,
		Creator:          models.UserShort{ID: uuid.New()},
		Category:         &models.CategoryShort{ID: oldCategory},
		CategoryModifier: &models.CategoryModifierShort{ID: oldModifier},
	}
	dto := &models.TicketDTO{
		ID:                 &old.ID,
		Actor:              &models.Actor{ID: actor},
		CategoryID:         newCategory,
		CategoryModifierID: &staleModifier,
	}
	dto.MarkProvided("categoryId")
	dto.MarkProvided("categoryModifierId")
	expectSupervisor(policies, actor, realm.String(), false)

	result, err := svc.prepareUpdateModifier(context.Background(), &updateModifierInput{DTO: dto, Old: old, ActorID: actor})
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.False(t, result.Applied)
	assert.Nil(t, dto.CategoryModifierID, "подкатегория сбрасывается при смене категории")
	modifiers.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

// TestTicketModifier_PrepareUpdate_EchoNotChange: форма правки шлёт все поля
// разом, поэтому тот же самый modifier id не должен ни сбрасывать подкатегорию,
// ни упираться в замороженный статус.
func TestTicketModifier_PrepareUpdate_EchoNotChange(t *testing.T) {
	svc, _, _, policies := modifierFixture()
	actor := uuid.New()
	realm := uuid.New()
	modifierID := uuid.New()

	old := &models.Ticket{
		ID:               uuid.New(),
		RealmID:          &realm,
		Status:           models.StatusClosed,
		Creator:          models.UserShort{ID: uuid.New()},
		CategoryModifier: &models.CategoryModifierShort{ID: modifierID},
	}
	dto := &models.TicketDTO{ID: &old.ID, Actor: &models.Actor{ID: actor}, CategoryModifierID: &modifierID}
	dto.MarkProvided("categoryModifierId")
	expectSupervisor(policies, actor, realm.String(), false)

	result, err := svc.prepareUpdateModifier(context.Background(), &updateModifierInput{DTO: dto, Old: old, ActorID: actor})
	require.NoError(t, err)
	assert.False(t, result.Changed)
	assert.False(t, result.Applied)
}

// TestTicketModifier_PrepareUpdate_ChangeRules проверяет, кто и когда может
// сменить подкатегорию: исполнитель или начальник области, только в open.
func TestTicketModifier_PrepareUpdate_ChangeRules(t *testing.T) {
	actor := uuid.New()
	realm := uuid.New()
	other := uuid.New()
	categoryID := uuid.New()
	oldModifier := uuid.New()
	newModifier := uuid.New()
	days := 4

	cases := []struct {
		name       string
		status     models.TicketStatus
		supervisor bool
		assignee   *uuid.UUID
		wantErr    error
	}{
		{name: "assignee_open_allowed", status: models.StatusOpen, assignee: &actor},
		{name: "supervisor_open_allowed", status: models.StatusOpen, supervisor: true, assignee: &other},
		{name: "non_assignee_denied", status: models.StatusOpen, assignee: &other, wantErr: models.ErrPermissionDenied},
		{name: "assignee_frozen_denied", status: models.StatusResolved, assignee: &actor, wantErr: models.ErrTicketFrozen},
		{name: "supervisor_frozen_denied", status: models.StatusClosed, supervisor: true, assignee: &other, wantErr: models.ErrTicketFrozen},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, modifiers, categories, policies := modifierFixture()
			old := &models.Ticket{
				ID:               uuid.New(),
				RealmID:          &realm,
				Status:           tc.status,
				Creator:          models.UserShort{ID: other},
				Assignee:         &models.UserShort{ID: *tc.assignee},
				Category:         &models.CategoryShort{ID: categoryID},
				CategoryModifier: &models.CategoryModifierShort{ID: oldModifier},
			}
			dto := &models.TicketDTO{
				ID:                 &old.ID,
				Actor:              &models.Actor{ID: actor},
				CategoryModifierID: &newModifier,
				Priority:           models.PriorityLow,
			}
			dto.MarkProvided("categoryModifierId")
			expectSupervisor(policies, actor, realm.String(), tc.supervisor)

			if tc.wantErr == nil {
				modifiers.On("GetByID", mock.Anything, &models.GetCategoryModifierByIDDTO{ID: newModifier, RealmID: realm}).
					Return(&models.CategoryModifier{ID: newModifier, CategoryID: categoryID, IsActive: true, Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days}, nil)
				categories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: categoryID, RealmID: realm}).
					Return(&models.Category{ID: categoryID, Priority: models.PriorityMedium}, nil)
			}

			before := time.Now()
			result, err := svc.prepareUpdateModifier(context.Background(), &updateModifierInput{DTO: dto, Old: old, ActorID: actor})
			after := time.Now()
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, result.Changed)
			assert.True(t, result.Applied)
			assert.Equal(t, tc.supervisor, result.MayOverride, "начальник области сохраняет право ручной правки полей")
			assert.Equal(t, models.PriorityUrgent, dto.Priority)
			require.NotNil(t, dto.DueDate)
			expectedA := addWorkingDaysMonFri(before, days, time.Local)
			expectedB := addWorkingDaysMonFri(after, days, time.Local)
			assert.True(t, dto.DueDate.Equal(expectedA) || dto.DueDate.Equal(expectedB),
				"due=%v expected %v or %v", *dto.DueDate, expectedA, expectedB)
		})
	}
}

// TestTicketModifier_PrepareUpdate_MayOverride закрепляет «Начальник + менеджер
// группы» и исключение «исполнитель-создатель, пока open и он исполнитель».
func TestTicketModifier_PrepareUpdate_MayOverride(t *testing.T) {
	actor := uuid.New()
	realm := uuid.New()
	other := uuid.New()
	groupID := uuid.New()
	modifierID := uuid.New()

	cases := []struct {
		name       string
		supervisor bool
		creator    uuid.UUID
		assignee   *uuid.UUID
		group      *models.GroupShort
		managed    []uuid.UUID
		want       bool
	}{
		{name: "supervisor", supervisor: true, creator: other, want: true},
		{name: "group_manager", creator: other, group: &models.GroupShort{ID: groupID}, managed: []uuid.UUID{groupID}, want: true},
		{name: "executor_creator", creator: actor, assignee: &actor, want: true},
		{name: "executor_not_creator", creator: other, assignee: &actor, want: false},
		{name: "creator_not_executor", creator: actor, assignee: &other, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, policies := modifierFixture()
			groups := svc.groups.(*MockGroupsRepo)
			old := &models.Ticket{
				ID:               uuid.New(),
				RealmID:          &realm,
				Status:           models.StatusOpen,
				Creator:          models.UserShort{ID: tc.creator},
				Assignee:         userShortOrNil(tc.assignee),
				Group:            tc.group,
				CategoryModifier: &models.CategoryModifierShort{ID: modifierID},
			}
			// Эхо текущей подкатегории: смены нет, но MayOverride всё равно считается.
			dto := &models.TicketDTO{ID: &old.ID, Actor: &models.Actor{ID: actor}, CategoryModifierID: &modifierID}
			dto.MarkProvided("categoryModifierId")
			expectSupervisor(policies, actor, realm.String(), tc.supervisor)
			if tc.group != nil {
				groups.On("GetManagedGroups", mock.Anything, actor, (*uuid.UUID)(nil)).Return(tc.managed, nil)
			}

			result, err := svc.prepareUpdateModifier(context.Background(), &updateModifierInput{DTO: dto, Old: old, ActorID: actor})
			require.NoError(t, err)
			assert.False(t, result.Changed)
			assert.Equal(t, tc.want, result.MayOverride)
		})
	}
}

// TestTicketService_Update_ModifierByNonCreatorAssignee — регресс первой фазы:
// исполнитель, который не создавал заявку, выбирает подкатегорию через отдельный
// поповер в InfoBar. Сервер разово подставляет срок и приоритет, и эти два change
// не должны отбрасываться правом «assignedOnly» (иначе выбор подкатегории
// завершался 403 даже при корректном праве на саму подкатегорию).
func TestTicketService_Update_ModifierByNonCreatorAssignee(t *testing.T) {
	repo, logs, _, _, notifications, groups, policies, svc := ticketServiceFixtures()
	modifiers := new(MockCategoryModifiers)
	categories := new(MockCategoriesRepo)
	svc.modifiers = modifiers
	svc.categories = categories

	actor := uuid.New()
	creator := uuid.New()
	realm := uuid.New()
	ticketID := uuid.New()
	oldCategoryID := uuid.New()
	oldModifierID := uuid.New()
	newModifierID := uuid.New()
	days := 3
	oldDue := time.Now().UTC().Add(24 * time.Hour)

	old := &models.Ticket{
		ID:               ticketID,
		RealmID:          &realm,
		Status:           models.StatusOpen,
		Creator:          models.UserShort{ID: creator},
		Owner:            &models.UserShort{ID: creator},
		Assignee:         &models.UserShort{ID: actor},
		Category:         &models.CategoryShort{ID: oldCategoryID},
		CategoryModifier: &models.CategoryModifierShort{ID: oldModifierID},
		Priority:         models.PriorityMedium,
		DueDate:          &oldDue,
		Title:            "Чужая заявка",
	}
	dto := &models.TicketDTO{
		ID:                 &ticketID,
		Actor:              &models.Actor{ID: actor, Name: "executor"},
		CategoryModifierID: &newModifierID,
	}
	dto.MarkProvided("categoryModifierId")

	// Исполнитель не является создателем и не управляет группой — Write по
	// атрибутам у него есть как у assignee, но canEditFields(false) переводит
	// обновление в режим assignedOnly.
	policies.On("Enforce", actor.String(), realm.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	policies.On("Enforce", actor.String(), realm.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	markUserAsEmployee(groups, actor)

	modifiers.On("GetByID", mock.Anything, &models.GetCategoryModifierByIDDTO{ID: newModifierID, RealmID: realm}).
		Return(&models.CategoryModifier{ID: newModifierID, CategoryID: oldCategoryID, IsActive: true,
			Priority: ptrPriority(models.PriorityUrgent), DefaultDueDays: &days}, nil)
	categories.On("GetByID", mock.Anything, &models.GetCategoryByIdDTO{ID: oldCategoryID, RealmID: realm}).
		Return(&models.Category{ID: oldCategoryID, Priority: models.PriorityMedium}, nil)

	repo.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(old, nil)
	repo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	logs.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	notifications.On("TicketUpdated", mock.Anything, mock.Anything, actor, mock.Anything).Return(nil)

	before := time.Now()
	require.NoError(t, svc.Update(context.Background(), dto))
	after := time.Now()
	assert.Equal(t, models.PriorityUrgent, dto.Priority, "приоритет подкатегории применён разово")
	require.NotNil(t, dto.DueDate)
	expectedA := addWorkingDaysMonFri(before, days, time.Local)
	expectedB := addWorkingDaysMonFri(after, days, time.Local)
	assert.True(t, dto.DueDate.Equal(expectedA) || dto.DueDate.Equal(expectedB),
		"due=%v expected %v or %v", *dto.DueDate, expectedA, expectedB)
}

func ptrPriority(p models.Priority) *models.Priority {
	return &p
}

func userShortOrNil(id *uuid.UUID) *models.UserShort {
	if id == nil {
		return nil
	}
	return &models.UserShort{ID: *id}
}
