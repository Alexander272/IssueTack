package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCategoryVisibility_CanSeeHidden_Supervisor(t *testing.T) {
	access := new(MockTicketAccessChecker)
	groups := new(MockGroupsRepo)
	view := NewCategoryVisibility(access, groups)

	userID := uuid.New()
	realmID := uuid.New()
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(true, nil)

	assert.True(t, view.CanSeeHidden(context.Background(), userID, realmID))
	// Начальник области видит скрытое без обращения к группам.
	groups.AssertNotCalled(t, "GetManagedGroups")
	groups.AssertNotCalled(t, "GetMemberGroups")
}

func TestCategoryVisibility_CanSeeHidden_ManagedGroup(t *testing.T) {
	access := new(MockTicketAccessChecker)
	groups := new(MockGroupsRepo)
	view := NewCategoryVisibility(access, groups)

	userID := uuid.New()
	realmID := uuid.New()
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{uuid.New()}, nil)

	assert.True(t, view.CanSeeHidden(context.Background(), userID, realmID))
	groups.AssertNotCalled(t, "GetMemberGroups")
}

func TestCategoryVisibility_CanSeeHidden_MemberGroup(t *testing.T) {
	access := new(MockTicketAccessChecker)
	groups := new(MockGroupsRepo)
	view := NewCategoryVisibility(access, groups)

	userID := uuid.New()
	realmID := uuid.New()
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)
	groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{uuid.New()}, nil)

	assert.True(t, view.CanSeeHidden(context.Background(), userID, realmID))
}

func TestCategoryVisibility_CanSeeHidden_GrouplessCustomer(t *testing.T) {
	access := new(MockTicketAccessChecker)
	groups := new(MockGroupsRepo)
	view := NewCategoryVisibility(access, groups)

	userID := uuid.New()
	realmID := uuid.New()
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)
	groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)

	assert.False(t, view.CanSeeHidden(context.Background(), userID, realmID))
}

func TestCategoryVisibility_CanSeeHidden_Anonymous(t *testing.T) {
	access := new(MockTicketAccessChecker)
	groups := new(MockGroupsRepo)
	view := NewCategoryVisibility(access, groups)

	assert.False(t, view.CanSeeHidden(context.Background(), uuid.Nil, uuid.New()))
	access.AssertNotCalled(t, "IsRealmSupervisor")
}

// Ошибки зависимостей трактуются как «скрытое остаётся скрытым» (fail-closed).
func TestCategoryVisibility_CanSeeHidden_FailClosed(t *testing.T) {
	userID := uuid.New()
	realmID := uuid.New()

	t.Run("supervisor error", func(t *testing.T) {
		access := new(MockTicketAccessChecker)
		groups := new(MockGroupsRepo)
		access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, assert.AnError)

		assert.False(t, NewCategoryVisibility(access, groups).CanSeeHidden(context.Background(), userID, realmID))
	})

	t.Run("managed groups error", func(t *testing.T) {
		access := new(MockTicketAccessChecker)
		groups := new(MockGroupsRepo)
		access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)
		groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID(nil), assert.AnError)

		assert.False(t, NewCategoryVisibility(access, groups).CanSeeHidden(context.Background(), userID, realmID))
	})

	t.Run("member groups error", func(t *testing.T) {
		access := new(MockTicketAccessChecker)
		groups := new(MockGroupsRepo)
		access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)
		groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)
		groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID(nil), assert.AnError)

		assert.False(t, NewCategoryVisibility(access, groups).CanSeeHidden(context.Background(), userID, realmID))
	})
}

func TestFilterVisibleCategories(t *testing.T) {
	hiddenGroup := &models.CategoryGroupShort{ID: uuid.New(), Name: "Скрытый раздел", IsHidden: true}
	visibleGroup := &models.CategoryGroupShort{ID: uuid.New(), Name: "Обычный раздел"}
	noGroup := (*models.CategoryGroupShort)(nil)

	categories := []*models.Category{
		{ID: uuid.New(), Name: "обычная"},
		{ID: uuid.New(), Name: "скрытая", IsHidden: true},
		{ID: uuid.New(), Name: "в скрытом разделе", CategoryGroup: hiddenGroup},
		{ID: uuid.New(), Name: "в обычном разделе", CategoryGroup: visibleGroup},
		{ID: uuid.New(), Name: "без раздела", CategoryGroup: noGroup},
		nil,
	}

	visible := filterVisibleCategories(categories)
	assert.Len(t, visible, 3)
	names := make([]string, 0, len(visible))
	for _, c := range visible {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"обычная", "в обычном разделе", "без раздела"}, names)
}
