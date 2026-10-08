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
)

func statisticsFixtures() (*MockTicketsRepo, *MockGroupsRepo, *MockAccessPolicies, *StatisticsService) {
	repo := new(MockTicketsRepo)
	groups := new(MockGroupsRepo)
	policies := new(MockAccessPolicies)

	access := NewTicketAccessService(repo, groups, policies)
	svc := NewStatisticsService(repo, groups, access)
	return repo, groups, policies, svc
}

// queryMatcher сопоставляет аргумент-модель StatisticsQuery по срезу доступа, при
// необходимости — по указателю фильтра (filter != nil) и разрезу (dim != "").
// Так ожидания мока различают вызовы ByDimension по полю Dim, а не по порядку.
func queryMatcher(scope models.StatisticsScope, filter *models.StatisticsFilter, dim string) interface{} {
	return mock.MatchedBy(func(q models.StatisticsQuery) bool {
		if !assert.ObjectsAreEqual(scope, q.Scope) {
			return false
		}
		if filter != nil && q.Filter != filter {
			return false
		}
		return dim == "" || q.Dim == dim
	})
}

// breakdownMatcher, в отличие от queryMatcher, дополнительно проверяет лимит:
// разрезы «заказчики» и «исполнители» обязаны уходить с Limit = 0 (без LIMIT в SQL —
// фронт сворачивает хвост в «Прочие» и ему нужен полный), категории/группы/площадки — statisticsBucketLimit.
func breakdownMatcher(scope models.StatisticsScope, filter *models.StatisticsFilter, dim string, limit int) interface{} {
	return mock.MatchedBy(func(q models.StatisticsQuery) bool {
		if q.Limit != limit {
			return false
		}
		if !assert.ObjectsAreEqual(scope, q.Scope) {
			return false
		}
		if filter != nil && q.Filter != filter {
			return false
		}
		return dim == "" || q.Dim == dim
	})
}

// expectStatisticsRepo регистрирует базовые агрегаты статистики. Управленческие
// разрезы (byGroup/byOwner/workload) ожидаются только при managerial=true.
func expectStatisticsRepo(repo *MockTicketsRepo, scope models.StatisticsScope, managerial bool) {
	repo.On("GetStatisticsSummary", mock.Anything, queryMatcher(scope, nil, "")).Return(&models.StatisticsSummary{Total: 3, Active: 1}, nil)
	repo.On("GetStatisticsByDimension", mock.Anything, breakdownMatcher(scope, nil, "category", statisticsBucketLimit)).Return([]*models.StatisticsBucket{}, nil)
	repo.On("GetStatisticsByDimension", mock.Anything, breakdownMatcher(scope, nil, "site", statisticsBucketLimit)).Return([]*models.StatisticsBucket{}, nil)
	repo.On("GetStatisticsByStatus", mock.Anything, queryMatcher(scope, nil, "")).Return([]*models.StatusBucket{}, nil)
	repo.On("GetStatisticsTrend", mock.Anything, queryMatcher(scope, nil, "")).Return([]*models.TrendPoint{}, nil)
	if managerial {
		repo.On("GetStatisticsByDimension", mock.Anything, breakdownMatcher(scope, nil, "group", statisticsBucketLimit)).Return([]*models.StatisticsBucket{}, nil)
		repo.On("GetStatisticsByOwner", mock.Anything, breakdownMatcher(scope, nil, "", 0)).Return([]*models.StatisticsBreakdownBucket{}, nil)
		repo.On("GetStatisticsWorkload", mock.Anything, breakdownMatcher(scope, nil, "", 0)).Return([]*models.WorkloadBucket{}, nil)
	}
}

func statisticsFilter(userID, realmID uuid.UUID) *models.StatisticsFilter {
	now := time.Now()
	return &models.StatisticsFilter{
		Actor:   &models.Actor{ID: userID, Name: "test"},
		RealmID: realmID,
		From:    now.AddDate(0, 0, -30),
		To:      now,
	}
}

func TestStatisticsService_Get_Supervisor(t *testing.T) {
	repo, _, policies, svc := statisticsFixtures()

	userID := uuid.New()
	realmID := uuid.New()

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	scope := models.StatisticsScope{RealmID: realmID, AllRealm: true}
	expectStatisticsRepo(repo, scope, true)

	stats, err := svc.Get(context.Background(), statisticsFilter(userID, realmID))
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats.Summary.Total)
	assert.NotNil(t, stats.ByGroup)
	assert.NotNil(t, stats.ByOwner)
	assert.NotNil(t, stats.Workload)
	repo.AssertExpectations(t)
	policies.AssertExpectations(t)
}

func TestStatisticsService_Get_Manager(t *testing.T) {
	repo, groups, policies, svc := statisticsFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{groupID}, nil)

	scope := models.StatisticsScope{RealmID: realmID, GroupIDs: []uuid.UUID{groupID}, AssigneeID: &userID}
	expectStatisticsRepo(repo, scope, true)

	stats, err := svc.Get(context.Background(), statisticsFilter(userID, realmID))
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.ByGroup)
	groups.AssertNotCalled(t, "GetMemberGroups", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestStatisticsService_Get_Executor(t *testing.T) {
	repo, groups, policies, svc := statisticsFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)
	groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{groupID}, nil)

	scope := models.StatisticsScope{RealmID: realmID, AssigneeID: &userID}
	expectStatisticsRepo(repo, scope, false)

	stats, err := svc.Get(context.Background(), statisticsFilter(userID, realmID))
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	// Управленческие разрезы пусты, но не nil: фронт считает их массивами.
	assert.Empty(t, stats.ByGroup)
	assert.Empty(t, stats.ByOwner)
	assert.Empty(t, stats.Workload)
	assert.NotNil(t, stats.ByGroup)
	assert.NotNil(t, stats.ByOwner)
	assert.NotNil(t, stats.Workload)
	assert.NotNil(t, stats.ByStatus)
	repo.AssertExpectations(t)
}

func TestStatisticsService_Get_Requester(t *testing.T) {
	repo, groups, policies, svc := statisticsFixtures()

	userID := uuid.New()
	realmID := uuid.New()

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)
	groups.On("GetMemberGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{}, nil)

	scope := models.StatisticsScope{RealmID: realmID, RequesterID: &userID}
	expectStatisticsRepo(repo, scope, false)

	stats, err := svc.Get(context.Background(), statisticsFilter(userID, realmID))
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	repo.AssertExpectations(t)
}

// TestStatisticsService_Get_PassesRefinements проверяет, что уточнения фильтра
// (исполнитель, категория) доходят до репозитория в составе StatisticsQuery и при
// этом срез доступа не расширяется: Scope остаётся прежним, фильтр — тот же указатель.
func TestStatisticsService_Get_PassesRefinements(t *testing.T) {
	repo, groups, policies, svc := statisticsFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	categoryID := uuid.New()

	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	policies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	groups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{groupID}, nil)

	filter := statisticsFilter(userID, realmID)
	filter.AssigneeIDs = []uuid.UUID{assigneeID}
	filter.CategoryIDs = []uuid.UUID{categoryID}

	scope := models.StatisticsScope{RealmID: realmID, GroupIDs: []uuid.UUID{groupID}, AssigneeID: &userID}
	repo.On("GetStatisticsSummary", mock.Anything, queryMatcher(scope, filter, "")).
		Return(&models.StatisticsSummary{Total: 1}, nil)
	repo.On("GetStatisticsByDimension", mock.Anything, queryMatcher(scope, filter, "category")).Return([]*models.StatisticsBucket{}, nil)
	repo.On("GetStatisticsByDimension", mock.Anything, queryMatcher(scope, filter, "site")).Return([]*models.StatisticsBucket{}, nil)
	repo.On("GetStatisticsByStatus", mock.Anything, queryMatcher(scope, filter, "")).Return([]*models.StatusBucket{}, nil)
	repo.On("GetStatisticsTrend", mock.Anything, queryMatcher(scope, filter, "")).Return([]*models.TrendPoint{}, nil)
	repo.On("GetStatisticsByDimension", mock.Anything, queryMatcher(scope, filter, "group")).Return([]*models.StatisticsBucket{}, nil)
	repo.On("GetStatisticsByOwner", mock.Anything, breakdownMatcher(scope, filter, "", 0)).Return([]*models.StatisticsBreakdownBucket{}, nil)
	repo.On("GetStatisticsWorkload", mock.Anything, breakdownMatcher(scope, filter, "", 0)).Return([]*models.WorkloadBucket{}, nil)

	stats, err := svc.Get(context.Background(), filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Summary.Total)
	repo.AssertExpectations(t)
}

func TestStatisticsService_Get_NoActor(t *testing.T) {
	_, _, _, svc := statisticsFixtures()

	_, err := svc.Get(context.Background(), &models.StatisticsFilter{RealmID: uuid.New()})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestAutoGranularity(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "day", autoGranularity(base, base.AddDate(0, 0, 7)))
	assert.Equal(t, "day", autoGranularity(base, base.AddDate(0, 0, 31)))
	assert.Equal(t, "week", autoGranularity(base, base.AddDate(0, 0, 32)))
	assert.Equal(t, "week", autoGranularity(base, base.AddDate(0, 0, 180)))
	assert.Equal(t, "month", autoGranularity(base, base.AddDate(0, 0, 181)))
}
