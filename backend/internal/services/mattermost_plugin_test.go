package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Встраиваемые фейки для сервисных интерфейсов, у которых нет моков в
// mock_repository_test.go. Неиспользуемые методы панифить не будут — тесты
// вызывают только переопределённые.

type fakeRealmsSvc struct {
	Realms
	getByID func(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error)
}

func (f *fakeRealmsSvc) GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
	return f.getByID(ctx, req)
}

type fakeCategoriesSvc struct {
	Categories
	get     func(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error)
	getByID func(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error)
}

func (f *fakeCategoriesSvc) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	return f.get(ctx, req)
}

func (f *fakeCategoriesSvc) GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
	return f.getByID(ctx, req)
}

type fakeSitesSvc struct {
	Sites
	get func(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error)
}

func (f *fakeSitesSvc) Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error) {
	return f.get(ctx, req)
}

type fakeGroupsSvc struct {
	Groups
	get        func(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error)
	getByID    func(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
	getMember  func(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
	getManaged func(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
}

func (f *fakeGroupsSvc) Get(ctx context.Context, req *models.GetGroupsDTO) ([]*models.Group, error) {
	if f.get == nil {
		return nil, nil
	}
	return f.get(ctx, req)
}

func (f *fakeGroupsSvc) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	if f.getByID == nil {
		return nil, nil
	}
	return f.getByID(ctx, req)
}

func (f *fakeGroupsSvc) GetMemberGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error) {
	if f.getMember == nil {
		return nil, nil
	}
	return f.getMember(ctx, userID, realmID)
}

func (f *fakeGroupsSvc) GetManagedGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error) {
	if f.getManaged == nil {
		return nil, nil
	}
	return f.getManaged(ctx, userID, realmID)
}

func pluginMocks() (*MockMattermostRepo, *MockUserService, *MockUserRealmsService, *MockTicketsService) {
	return new(MockMattermostRepo), new(MockUserService), new(MockUserRealmsService), new(MockTicketsService)
}

// pluginScopeFixture — scope обычного (привязанного) канала без бота.
func pluginScopeFixture(channelID string) models.PluginScope {
	return models.PluginScope{ChannelID: channelID, MmUserID: "mm1"}
}

// expectExistingUser готовит моки пути «пользователь уже есть по mattermost_id».
// UpdateMMAndSite здесь намеренно не ожидается: fast-path не должен писать в
// users, если site_id не передан (см. TestPluginContext_KnownUser_NoWrite).
func expectExistingUser(users *MockUserService, userRealms *MockUserRealmsService, userID, realmID uuid.UUID) {
	userSite := "a1b2c3d4-0000-0000-0000-000000000001"
	users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(&models.UserData{ID: userID, Username: "u1", SiteID: &userSite}, nil)
	userRealms.On("GetByUserAndRealm", mock.Anything, userID, realmID).Return(&models.UserRealm{UserID: userID, RealmID: realmID}, nil)
}

func TestPluginListMine_HappyPath(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch1", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	num := 7
	tickets.On("Get", mock.Anything, mock.MatchedBy(func(f *models.TicketFilter) bool {
		if f.Mode == nil || *f.Mode != "created_or_owned" {
			return false
		}
		if f.RealmID == nil || *f.RealmID != realmID {
			return false
		}
		return len(f.Statuses) == 5
	})).Return([]*models.Ticket{
		{ID: uuid.New(), Title: "Заявка 1", Status: models.StatusOpen, Priority: models.PriorityHigh, TicketNumber: &num, CreatedAt: time.Now()},
	}, 1, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	list, err := svc.PluginListMine(context.Background(), pluginScopeFixture("ch1"))
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, num, list[0].Number)
	assert.Equal(t, models.StatusOpen, list[0].Status)
	assert.Equal(t, models.PriorityHigh, list[0].Priority)
	assert.Empty(t, list[0].Link)
}

func TestPluginListMine_Statuses(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{}, 0, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	_, err := svc.PluginListMine(context.Background(), pluginScopeFixture("ch1"))
	require.NoError(t, err)

	var filter *models.TicketFilter
	for _, call := range tickets.Calls {
		if call.Method == "Get" {
			filter = call.Arguments.Get(1).(*models.TicketFilter)
		}
	}
	require.NotNil(t, filter)
	// Активные статусы + resolved («ждут подтверждения» владельцем).
	want := map[models.TicketStatus]bool{
		models.StatusOpen: true, models.StatusInProgress: true,
		models.StatusPending: true, models.StatusOnHold: true, models.StatusResolved: true,
	}
	require.Len(t, filter.Statuses, 5)
	for _, s := range filter.Statuses {
		assert.True(t, want[s], "неожиданный статус: %s", s)
		assert.NotEqual(t, models.StatusClosed, s)
		assert.NotEqual(t, models.StatusCancelled, s)
	}
	// «Автор ИЛИ заказчик»: плагин запрашивает mode=created_or_owned, который
	// реальный TicketService переводит в фильтр (creator_id OR owner_id).
	if filter.Mode != nil {
		assert.Equal(t, "created_or_owned", *filter.Mode)
	}
	assert.Nil(t, filter.CreatorID)
}

func TestPluginContext_UnboundChannel(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	ctx, err := svc.PluginContext(context.Background(), pluginScopeFixture("ch1"))
	require.NoError(t, err)
	require.NotNil(t, ctx)
	assert.False(t, ctx.Bound)
}

func TestPluginContext_InactiveChannel(t *testing.T) {
	realmID := uuid.New()
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: false}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	ctx, err := svc.PluginContext(context.Background(), pluginScopeFixture("ch1"))
	require.NoError(t, err)
	require.NotNil(t, ctx)
	assert.False(t, ctx.Bound)
}

func TestPluginContext_HappyPath(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	siteID := uuid.New()
	repo, users, userRealms, _ := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch1", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	realms := &fakeRealmsSvc{getByID: func(_ context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
		assert.Equal(t, realmID, req.ID)
		return &models.Realm{ID: realmID, Name: "Риалм"}, nil
	}}
	categories := &fakeCategoriesSvc{get: func(_ context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
		assert.Equal(t, realmID, req.RealmID)
		return []*models.Category{{ID: categoryID, Name: "Категория"}}, nil
	}}
	sites := &fakeSitesSvc{get: func(_ context.Context, _ *models.GetSitesDTO) ([]*models.Site, error) {
		return []*models.Site{{ID: siteID, Name: "Площадка"}}, nil
	}}
	groups := &fakeGroupsSvc{get: func(_ context.Context, _ *models.GetGroupsDTO) ([]*models.Group, error) {
		defaultAssignee := uuid.New()
		return []*models.Group{{ID: uuid.New(), Name: "Группа", DefaultAssigneeID: &defaultAssignee}}, nil
	}}
	users.On("GetByMembership", mock.Anything, realmID, models.MembershipExecutors).Return([]*models.UserData{{ID: uuid.New(), Username: "ex1"}}, nil)
	users.On("GetByMembership", mock.Anything, realmID, models.MembershipCustomers).Return([]*models.UserData{{ID: uuid.New(), Username: "cu1"}}, nil)

	access := new(MockTicketAccessChecker)
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms,
		Realms: realms, Categories: categories, Sites: sites, Groups: groups,
		Access: access,
	})

	result, err := svc.PluginContext(context.Background(), pluginScopeFixture("ch1"))
	require.NoError(t, err)
	assert.Equal(t, true, result.Bound)
	assert.Equal(t, realmID, result.RealmID)
	assert.Equal(t, "Риалм", result.RealmName)
	assert.Equal(t, userID, result.User.ID)
	assert.Equal(t, "u1", result.User.Username)
	assert.NotNil(t, result.User.SiteID)
	assert.Equal(t, "a1b2c3d4-0000-0000-0000-000000000001", *result.User.SiteID)
	assert.Len(t, result.Categories, 1)
	assert.Len(t, result.Sites, 1)
	// Роли и списки формы.
	assert.False(t, result.IsManager)
	assert.Empty(t, result.MemberGroupIds)
	assert.Len(t, result.Groups, 1)
	assert.Equal(t, "Группа", result.Groups[0].Name)
	// defaultAssigneeId групп нужен фронту для веб-подобного автозаполнения
	// исполнителя при смене группы менеджером.
	assert.NotNil(t, result.Groups[0].DefaultAssigneeID)
	assert.Len(t, result.Executors, 1)
	assert.Len(t, result.Customers, 1)
}

// TestPluginContext_KnownUser_NoWrite фиксирует отсутствие записи в users на
// fast-path: mattermost_id уже совпадает, site_id не передан, поэтому UPDATE
// менял бы только updated_at. /plugin/context вызывается при каждом переключении
// канала, и лишняя запись давала бы мёртвые кортежи без изменения данных.
func TestPluginContext_KnownUser_NoWrite(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	siteID := uuid.New()
	repo, users, userRealms, _ := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch1", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	realms := &fakeRealmsSvc{getByID: func(_ context.Context, _ *models.GetRealmByIdDTO) (*models.Realm, error) {
		return &models.Realm{ID: realmID, Name: "Риалм"}, nil
	}}
	categories := &fakeCategoriesSvc{get: func(_ context.Context, _ *models.GetCategoriesDTO) ([]*models.Category, error) {
		return nil, nil
	}}
	sites := &fakeSitesSvc{get: func(_ context.Context, _ *models.GetSitesDTO) ([]*models.Site, error) {
		return []*models.Site{{ID: siteID}}, nil
	}}
	groups := &fakeGroupsSvc{get: func(_ context.Context, _ *models.GetGroupsDTO) ([]*models.Group, error) {
		return nil, nil
	}}
	users.On("GetByMembership", mock.Anything, realmID, mock.Anything).Return([]*models.UserData{}, nil)

	access := new(MockTicketAccessChecker)
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms,
		Realms: realms, Categories: categories, Sites: sites, Groups: groups,
		Access: access,
	})

	// Два вызова подряд — как два переключения канала одним пользователем.
	for i := 0; i < 2; i++ {
		result, err := svc.PluginContext(context.Background(), pluginScopeFixture("ch1"))
		require.NoError(t, err)
		require.True(t, result.Bound)
		require.Equal(t, userID, result.User.ID)
	}

	users.AssertNotCalled(t, "UpdateMMAndSite", mock.Anything, mock.Anything, mock.Anything)
}

func TestPluginCreateTicket_HappyPath(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	groupID := uuid.New()
	assigneeID := uuid.New()
	managerID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch1", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	categories := &fakeCategoriesSvc{getByID: func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		assert.Equal(t, categoryID, req.ID)
		return &models.Category{ID: categoryID, GroupID: groupID, Priority: models.PriorityHigh}, nil
	}}
	groups := &fakeGroupsSvc{getByID: func(_ context.Context, req *models.GetGroupDTO) (*models.Group, error) {
		assert.Equal(t, groupID, req.ID)
		return &models.Group{ID: groupID, DefaultAssigneeID: &assigneeID, ManagerID: &managerID}, nil
	}}

	tickets.On("Create", mock.Anything, mock.MatchedBy(func(dto *models.TicketDTO) bool {
		return dto.Title == "Новая заявка" &&
			dto.CategoryID == categoryID &&
			dto.GroupID != nil && *dto.GroupID == groupID &&
			dto.AssigneeID != nil && *dto.AssigneeID == assigneeID &&
			dto.ManagerID != nil && *dto.ManagerID == managerID &&
			dto.Priority == models.PriorityHigh &&
			dto.Status == models.StatusOpen &&
			dto.CreatorID == userID
	})).Run(func(args mock.Arguments) {
		dto := args.Get(1).(*models.TicketDTO)
		id := uuid.New()
		dto.ID = &id
		number := 42
		dto.TicketNumber = number
		if dto.RealmID == nil {
			dto.RealmID = &realmID
		}
	}).Return(nil)

	tickets.On("UploadAttachment", mock.Anything, mock.Anything, mock.Anything).Return(
		&models.Attachment{ID: uuid.New(), FileName: "file.txt"}, nil,
	)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms,
		Tickets: tickets, Categories: categories, Groups: groups,
	})

	input := &models.PluginCreateTicketInput{
		PluginScope: pluginScopeFixture("ch1"),
		Title:       "Новая заявка",
		Description: "Описание",
		CategoryID:  categoryID,
		Files: []models.PluginFile{
			{FileName: "a.txt", FileSize: 3, MimeType: "text/plain", Content: bytes.NewReader([]byte("abc"))},
			{FileName: "b.txt", FileSize: 3, MimeType: "text/plain", Content: bytes.NewReader([]byte("def"))},
		},
	}

	result, err := svc.PluginCreateTicket(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Новая заявка", result.Title)
	assert.NotEmpty(t, result.ID)
	tickets.AssertExpectations(t)
	tickets.AssertNumberOfCalls(t, "UploadAttachment", 2)
}

// TestPluginCreateTicket_ManagerFields: явные поля менеджера (приоритет,
// группа, исполнитель, заказчик, срок) перебивают значения категории и дефолты
// группы. Дальше их роль ещё раз подтверждает TicketService.Create как coarse-
// правами, так и applyExecutorCreateRestrictions — плагин только прокидывает.
func TestPluginCreateTicket_ManagerFields(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	catGroupID := uuid.New()
	chosenGroupID := uuid.New()
	assigneeID := uuid.New()
	ownerID := uuid.New()
	due := time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch1", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	categories := &fakeCategoriesSvc{getByID: func(_ context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
		assert.Equal(t, categoryID, req.ID)
		// Категория по умолчанию ведёт в catGroupID с приоритетом high —
		// менеджер выбирает другую группу и low, и его выбор должен победить.
		return &models.Category{ID: categoryID, GroupID: catGroupID, Priority: models.PriorityHigh}, nil
	}}
	groups := &fakeGroupsSvc{getByID: func(_ context.Context, req *models.GetGroupDTO) (*models.Group, error) {
		assert.Equal(t, chosenGroupID, req.ID)
		return &models.Group{ID: chosenGroupID, DefaultAssigneeID: nil, ManagerID: nil}, nil
	}}

	tickets.On("Create", mock.Anything, mock.MatchedBy(func(dto *models.TicketDTO) bool {
		return dto.CategoryID == categoryID &&
			dto.GroupID != nil && *dto.GroupID == chosenGroupID &&
			dto.Priority == models.PriorityLow &&
			dto.AssigneeID != nil && *dto.AssigneeID == assigneeID &&
			dto.OwnerID != nil && *dto.OwnerID == ownerID &&
			dto.DueDate != nil && dto.DueDate.Equal(due)
	})).Run(func(args mock.Arguments) {
		dto := args.Get(1).(*models.TicketDTO)
		id := uuid.New()
		dto.ID = &id
		if dto.RealmID == nil {
			dto.RealmID = &realmID
		}
	}).Return(nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms,
		Tickets: tickets, Categories: categories, Groups: groups,
	})

	input := &models.PluginCreateTicketInput{
		PluginScope: pluginScopeFixture("ch1"),
		Title:       "Заявка менеджера",
		CategoryID:  categoryID,
		Priority:    "low",
		GroupID:     chosenGroupID,
		AssigneeID:  assigneeID,
		OwnerID:     ownerID,
		DueDate:     &due,
	}

	_, err := svc.PluginCreateTicket(context.Background(), input)
	require.NoError(t, err)
	tickets.AssertExpectations(t)
}

func TestPluginCreateTicket_UnboundChannel(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginCreateTicket(context.Background(), &models.PluginCreateTicketInput{
		PluginScope: pluginScopeFixture("ch1"), Title: "Заявка",
	})
	assert.ErrorIs(t, err, models.ErrChannelNotBound)
}

func TestPluginGetTicket_HappyPath(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	num := 9
	due := time.Now().Add(24 * time.Hour)
	tickets.On("GetByID", mock.Anything, mock.MatchedBy(func(d *models.GetTicketByIdDTO) bool {
		return d.ID == ticketID && d.Actor != nil && d.Actor.ID == userID && d.RealmID == realmID.String()
	})).Return(&models.Ticket{
		ID:           ticketID,
		Title:        "Заявка 9",
		Description:  "Описание заявки",
		Status:       models.StatusOpen,
		Priority:     models.PriorityMedium,
		TicketNumber: &num,
		CreatedAt:    time.Now(),
		DueDate:      &due,
		Creator:      models.UserShort{ID: userID, Username: "mm1"},
		Category:     &models.CategoryShort{ID: uuid.New(), Name: "Категория"},
		Site:         &models.SiteShort{ID: uuid.New(), Name: "Площадка"},
	}, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets,
	})

	detail, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, ticketID, detail.ID)
	assert.Equal(t, num, detail.Number)
	assert.Equal(t, "Описание заявки", detail.Description)
	assert.Equal(t, models.StatusOpen, detail.Status)
	assert.Equal(t, "Площадка", detail.Site.Name)
	assert.Equal(t, userID, detail.Creator.ID)
	assert.Empty(t, detail.Link)

	// DeepLink site-relative и заполняется всегда, в отличие от Link,
	// который зависит от настроенного baseURL.
	assert.Equal(t, "/plug/issuetrack/ticket/"+ticketID.String(), detail.DeepLink)
}

func TestPluginGetTicketLinkContext_ResolvesChannelByRealm(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	// Канал клиент не передаёт: реалм определяется по тикету, канал берётся
	// из настроек реалма.
	tickets.On("GetRealmIDByTicketID", mock.Anything, ticketID).Return(realmID, nil)
	repo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch9", IsActive: true}, nil)
	// PluginGetTicket пере-resolve'ит настройки уже по найденному каналу.
	repo.On("GetByChannelID", mock.Anything, "ch9").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch9", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	num := 3
	tickets.On("GetByID", mock.Anything, mock.MatchedBy(func(d *models.GetTicketByIdDTO) bool {
		return d.ID == ticketID && d.Actor != nil && d.Actor.ID == userID && d.RealmID == realmID.String()
	})).Return(&models.Ticket{
		ID:           ticketID,
		Title:        "Заявка 3",
		Status:       models.StatusOpen,
		Priority:     models.PriorityMedium,
		TicketNumber: &num,
		CreatedAt:    time.Now(),
		Creator:      models.UserShort{ID: userID, Username: "mm1"},
	}, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets,
	})

	linkContext, err := svc.PluginGetTicketLinkContext(context.Background(), "mm1", ticketID.String())
	require.NoError(t, err)
	require.NotNil(t, linkContext)
	assert.Equal(t, "ch9", linkContext.ChannelID)
	require.NotNil(t, linkContext.Detail)
	assert.Equal(t, ticketID, linkContext.Detail.ID)
	assert.Equal(t, "/plug/issuetrack/ticket/"+ticketID.String(), linkContext.Detail.DeepLink)

	// Канал приходит только из настроек реалма, клиентский параметр не участвует.
	repo.AssertNotCalled(t, "GetByChannelID", mock.Anything, "ch1")
}

func TestPluginGetTicketLinkContext_InvalidID(t *testing.T) {
	repo, _, _, _ := pluginMocks()

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginGetTicketLinkContext(context.Background(), "mm1", "not-a-uuid")
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestPluginGetTicketLinkContext_InactiveRealm(t *testing.T) {
	realmID := uuid.New()
	ticketID := uuid.New()
	repo, _, _, tickets := pluginMocks()

	tickets.On("GetRealmIDByTicketID", mock.Anything, ticketID).Return(realmID, nil)
	repo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch9", IsActive: false}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Tickets: tickets})

	_, err := svc.PluginGetTicketLinkContext(context.Background(), "mm1", ticketID.String())
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrChannelNotBound)
}

// Права на заявку проверяет PluginGetTicket внутри; link-context не должен
// обходить её, поэтому ошибка доступа пробрасывается как есть.
func TestPluginGetTicketLinkContext_PermissionDeniedPropagated(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	tickets.On("GetRealmIDByTicketID", mock.Anything, ticketID).Return(realmID, nil)
	repo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch9", IsActive: true}, nil)
	repo.On("GetByChannelID", mock.Anything, "ch9").Return(&models.RealmMattermost{RealmID: realmID, ChannelID: "ch9", IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("GetByID", mock.Anything, mock.Anything).Return(nil, models.ErrPermissionDenied)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets,
	})

	_, err := svc.PluginGetTicketLinkContext(context.Background(), "mm1", ticketID.String())
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestPluginGetTicket_InvalidID(t *testing.T) {
	realmID := uuid.New()
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), "not-a-uuid")
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

type fakeCommentsSvc struct {
	Comments
	getByTicket func(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, realm string) ([]*models.Comment, error)
	create      func(ctx context.Context, tx postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error)
}

func (f *fakeCommentsSvc) GetByTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, realm string) ([]*models.Comment, error) {
	return f.getByTicket(ctx, ticketID, userID, realm)
}

func (f *fakeCommentsSvc) Create(ctx context.Context, tx postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error) {
	return f.create(ctx, tx, dto)
}

func TestPluginGetComments_FiltersInternal(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	attID := uuid.New()
	repo, users, userRealms, _ := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	comments := &fakeCommentsSvc{
		getByTicket: func(ctx context.Context, tid uuid.UUID, uid uuid.UUID, realm string) ([]*models.Comment, error) {
			assert.Equal(t, ticketID, tid)
			assert.Equal(t, userID, uid)
			assert.Equal(t, realmID.String(), realm)
			return []*models.Comment{
				{ID: uuid.New(), Text: "внутренний", IsInternal: true, CreatedAt: time.Now(),
					User: &models.UserShort{ID: userID, Username: "mm1"}},
				{ID: uuid.New(), Text: "публичный", IsInternal: false, CreatedAt: time.Now().Add(time.Minute),
					User:        &models.UserShort{ID: userID, Username: "mm1", FirstName: "Иван", LastName: "Иванов"},
					Attachments: []*models.Attachment{{ID: attID, FileName: "a.txt", FileSize: 3, MimeType: "text/plain"}}},
			}, nil
		},
	}

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Comments: comments})

	list, err := svc.PluginGetComments(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "публичный", list[0].Text)
	assert.Equal(t, "Иван Иванов", list[0].User.Name)
	require.Len(t, list[0].Attachments, 1)
	assert.Equal(t, "a.txt", list[0].Attachments[0].FileName)
}

func TestPluginCreateComment_HappyPath(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, _ := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	created := &models.Comment{ID: uuid.New(), Text: "Всем привет", TicketID: ticketID, UserID: userID, CreatedAt: time.Now()}
	comments := &fakeCommentsSvc{
		create: func(ctx context.Context, _ postgres.Tx, dto *models.CreateCommentDTO) (*models.Comment, error) {
			assert.Equal(t, ticketID, dto.TicketID)
			assert.Equal(t, "Всем привет", dto.Text)
			assert.Equal(t, userID, dto.UserID)
			assert.Equal(t, realmID.String(), dto.Realm)
			assert.False(t, dto.IsInternal)
			require.Len(t, dto.Files, 1)
			assert.Equal(t, "f.txt", dto.Files[0].FileName)
			return created, nil
		},
	}

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Comments: comments})

	out, err := svc.PluginCreateComment(context.Background(), &models.PluginCreateCommentInput{
		PluginScope: pluginScopeFixture("ch1"),
		TicketID:    ticketID.String(),
		Text:        "Всем привет",
		Files: []models.PluginFile{
			{FileName: "f.txt", FileSize: 3, MimeType: "text/plain", Content: bytes.NewReader([]byte("abc"))},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, created.ID, out.ID)
	assert.Equal(t, "Всем привет", out.Text)
	assert.Equal(t, userID, out.User.ID)
}

func TestPluginCreateComment_EmptyFails(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: uuid.New(), IsActive: true}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginCreateComment(context.Background(), &models.PluginCreateCommentInput{
		PluginScope: pluginScopeFixture("ch1"), TicketID: uuid.New().String(),
	})
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

type fakeAttachmentsSvc struct {
	Attachments
	getContent func(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error)
}

func (f *fakeAttachmentsSvc) GetContent(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error) {
	return f.getContent(ctx, id, actorID, realm)
}

func TestPluginGetAttachmentContent(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	attID := uuid.New()
	repo, users, userRealms, _ := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)

	att := &models.Attachment{ID: attID, FileName: "doc.pdf", FileSize: 10, MimeType: "application/pdf"}
	attachments := &fakeAttachmentsSvc{
		getContent: func(ctx context.Context, id uuid.UUID, actorID uuid.UUID, realm string) (*models.Attachment, io.ReadCloser, error) {
			assert.Equal(t, attID, id)
			assert.Equal(t, userID, actorID)
			assert.Equal(t, realmID.String(), realm)
			return att, io.NopCloser(strings.NewReader("content")), nil
		},
	}

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Attachments: attachments})

	got, rc, err := svc.PluginGetAttachmentContent(context.Background(), pluginScopeFixture("ch1"), attID.String())
	require.NoError(t, err)
	defer rc.Close()
	assert.Equal(t, attID, got.ID)
	assert.Equal(t, "doc.pdf", got.FileName)
	data, _ := io.ReadAll(rc)
	assert.Equal(t, "content", string(data))

	attBody, _, err := svc.PluginGetAttachmentContent(context.Background(), pluginScopeFixture("ch1"), "bad-id")
	assert.ErrorIs(t, err, models.ErrInvalidInput)
	assert.Nil(t, attBody)
}

func pluginGetTicketWithTicket(repo *MockMattermostRepo, users *MockUserService, userRealms *MockUserRealmsService, tickets *MockTicketsService, realmID, userID, ticketID uuid.UUID, tck *models.Ticket) {
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("GetByID", mock.Anything, mock.MatchedBy(func(d *models.GetTicketByIdDTO) bool {
		return d.ID == ticketID && d.Actor != nil && d.Actor.ID == userID && d.RealmID == realmID.String()
	})).Return(tck, nil)
}

func TestPluginGetTicket_OwnerFlags(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	pluginGetTicketWithTicket(repo, users, userRealms, tickets, realmID, userID, ticketID, &models.Ticket{
		ID:     ticketID,
		Status: models.StatusResolved,
		Owner:  &models.UserShort{ID: userID, Username: "u1"},
	})

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	detail, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	assert.True(t, detail.CanConfirm)
	assert.True(t, detail.CanReopen)
	assert.False(t, detail.CanCancel)
}

func TestPluginGetTicket_OwnerOpenCanCancel(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	pluginGetTicketWithTicket(repo, users, userRealms, tickets, realmID, userID, ticketID, &models.Ticket{
		ID:     ticketID,
		Status: models.StatusOpen,
		Owner:  &models.UserShort{ID: userID, Username: "u1"},
	})

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	detail, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	assert.False(t, detail.CanConfirm)
	assert.False(t, detail.CanReopen)
	assert.True(t, detail.CanCancel)
}

func TestPluginGetTicket_OwnerInProgressNoCancel(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	pluginGetTicketWithTicket(repo, users, userRealms, tickets, realmID, userID, ticketID, &models.Ticket{
		ID:     ticketID,
		Status: models.StatusInProgress,
		Owner:  &models.UserShort{ID: userID, Username: "u1"},
	})

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	detail, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	assert.False(t, detail.CanConfirm)
	assert.False(t, detail.CanReopen)
	assert.False(t, detail.CanCancel)
}

func TestPluginGetTicket_NotOwnerNoFlags(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	pluginGetTicketWithTicket(repo, users, userRealms, tickets, realmID, userID, ticketID, &models.Ticket{
		ID:     ticketID,
		Status: models.StatusResolved,
		Owner:  &models.UserShort{ID: otherID, Username: "u2"},
	})

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	detail, err := svc.PluginGetTicket(context.Background(), pluginScopeFixture("ch1"), ticketID.String())
	require.NoError(t, err)
	assert.False(t, detail.CanConfirm)
	assert.False(t, detail.CanReopen)
	assert.False(t, detail.CanCancel)
}

func TestPluginChangeStatus_HappyPaths(t *testing.T) {
	for _, status := range []models.TicketStatus{models.StatusClosed, models.StatusInProgress, models.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			realmID := uuid.New()
			userID := uuid.New()
			ticketID := uuid.New()
			repo, users, userRealms, tickets := pluginMocks()

			repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
			expectExistingUser(users, userRealms, userID, realmID)

			tickets.On("Update", mock.Anything, mock.MatchedBy(func(d *models.TicketDTO) bool {
				if d.ID == nil || *d.ID != ticketID {
					return false
				}
				if d.Actor == nil || d.Actor.ID != userID {
					return false
				}
				if d.RealmID == nil || *d.RealmID != realmID {
					return false
				}
				if d.Status != status || !d.HasField("status") {
					return false
				}
				return true
			})).Return(nil)

			svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

			err := svc.PluginChangeStatus(context.Background(), pluginScopeFixture("ch1"), ticketID.String(), string(status))
			require.NoError(t, err)
		})
	}
}

func TestPluginChangeStatus_PermissionDeniedPropagated(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo, users, userRealms, tickets := pluginMocks()

	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)
	expectExistingUser(users, userRealms, userID, realmID)
	tickets.On("Update", mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets})

	err := svc.PluginChangeStatus(context.Background(), pluginScopeFixture("ch1"), ticketID.String(), string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestPluginChangeStatus_Invalid(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: uuid.New(), IsActive: true}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	err := svc.PluginChangeStatus(context.Background(), pluginScopeFixture("ch1"), uuid.New().String(), "bogus")
	assert.ErrorIs(t, err, models.ErrInvalidInput)

	err = svc.PluginChangeStatus(context.Background(), pluginScopeFixture("ch1"), "not-a-uuid", string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestPluginChangeStatus_UnboundChannel(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	err := svc.PluginChangeStatus(context.Background(), pluginScopeFixture("ch1"), uuid.New().String(), string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrChannelNotBound)
}

// dmScopeServer отдаёт состав участников канала для проверки scope личного
// диалога. Endpoint members/ids — POST с телом-списком id: он ничего не
// меняет, только возвращает участников.
func dmScopeServer(t *testing.T, members ...string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/channels/dm1/members/ids")

		var requested []string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&requested))
		assert.ElementsMatch(t, []string{"bot1", "mm1"}, requested)

		out := make([]map[string]any, 0, len(members))
		for _, id := range members {
			out = append(out, map[string]any{"user_id": id})
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(out))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// dmScopeFixture — scope личного диалога: канал не привязан к реалму, реалм
// определяется по боту, участники подтверждаются запросом members.
func dmScopeFixture() models.PluginScope {
	return models.PluginScope{ChannelID: "dm1", MmUserID: "mm1", BotUserID: "bot1"}
}

func dmSettings(realmID uuid.UUID) *models.RealmMattermost {
	return &models.RealmMattermost{
		RealmID: realmID, BotUserID: "bot1", BotToken: "tok", IsActive: true,
	}
}

func dmMocks(t *testing.T, members ...string) (*MockMattermostRepo, *MockUserService, *MockUserRealmsService, *MockTicketsService, *mattermost.Most) {
	t.Helper()
	repo, users, userRealms, tickets := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "dm1").Return(nil, errors.New("no rows"))
	most := &mattermost.Most{Client: mattermost.NewClient(dmScopeServer(t, members...).URL)}
	return repo, users, userRealms, tickets, most
}

func TestPluginContext_DMWithBot(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	siteID := uuid.New()
	repo, users, userRealms, _, most := dmMocks(t, "bot1", "mm1")

	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(realmID), nil)
	expectExistingUser(users, userRealms, userID, realmID)

	realms := &fakeRealmsSvc{getByID: func(_ context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error) {
		assert.Equal(t, realmID, req.ID)
		return &models.Realm{ID: realmID, Name: "Риалм"}, nil
	}}
	categories := &fakeCategoriesSvc{get: func(_ context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
		assert.Equal(t, realmID, req.RealmID)
		return []*models.Category{{ID: categoryID, Name: "Категория"}}, nil
	}}
	sites := &fakeSitesSvc{get: func(_ context.Context, _ *models.GetSitesDTO) ([]*models.Site, error) {
		return []*models.Site{{ID: siteID, Name: "Площадка"}}, nil
	}}
	groups := &fakeGroupsSvc{get: func(_ context.Context, _ *models.GetGroupsDTO) ([]*models.Group, error) {
		return nil, nil
	}}
	users.On("GetByMembership", mock.Anything, realmID, mock.Anything).Return([]*models.UserData{}, nil)

	access := new(MockTicketAccessChecker)
	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms, Most: most,
		Realms: realms, Categories: categories, Sites: sites, Groups: groups,
		Access: access,
	})

	result, err := svc.PluginContext(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	assert.True(t, result.Bound)
	assert.Equal(t, realmID, result.RealmID)
	assert.Equal(t, userID, result.User.ID)
	assert.Len(t, result.Categories, 1)
}

func TestPluginContext_DMWithoutBotInChannel(t *testing.T) {
	// В диалоге только бот и сам пользователь не состоят: канал не его диалог.
	repo, _, _, _, most := dmMocks(t, "bot1")
	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(uuid.New()), nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Most: most})

	result, err := svc.PluginContext(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	assert.False(t, result.Bound)
}

func TestPluginContext_DMExtraMember(t *testing.T) {
	// Три участника — это групповой канал, а не личный диалог.
	repo, _, _, _, most := dmMocks(t, "bot1", "mm1", "third")
	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(uuid.New()), nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Most: most})

	result, err := svc.PluginContext(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	assert.False(t, result.Bound)
}

func TestPluginContext_DMUnknownBot(t *testing.T) {
	repo, _, _, _, most := dmMocks(t, "bot1", "mm1")
	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Most: most})

	result, err := svc.PluginContext(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	assert.False(t, result.Bound)
}

func TestPluginContext_DMWithoutMostClient(t *testing.T) {
	// Без клиента Mattermost подтвердить состав участников нечем.
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "dm1").Return(nil, errors.New("no rows"))
	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(uuid.New()), nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	result, err := svc.PluginContext(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	assert.False(t, result.Bound)
}

func TestPluginListMine_DMWithBot(t *testing.T) {
	realmID := uuid.New()
	userID := uuid.New()
	repo, users, userRealms, tickets, most := dmMocks(t, "bot1", "mm1")

	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(realmID), nil)
	expectExistingUser(users, userRealms, userID, realmID)

	num := 11
	tickets.On("Get", mock.Anything, mock.Anything).Return([]*models.Ticket{
		{ID: uuid.New(), Title: "Заявка из DM", Status: models.StatusOpen, Priority: models.PriorityHigh, TicketNumber: &num, CreatedAt: time.Now()},
	}, 1, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets, Most: most})

	list, err := svc.PluginListMine(context.Background(), dmScopeFixture())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, num, list[0].Number)

	var filter *models.TicketFilter
	for _, call := range tickets.Calls {
		if call.Method == "Get" {
			filter = call.Arguments.Get(1).(*models.TicketFilter)
		}
	}
	require.NotNil(t, filter)
	require.NotNil(t, filter.RealmID)
	assert.Equal(t, realmID, *filter.RealmID)
}

func TestPluginListMine_DMNotADirectDialog(t *testing.T) {
	repo, users, userRealms, tickets, most := dmMocks(t, "bot1", "mm1", "third")
	repo.On("GetByBotUserID", mock.Anything, "bot1").Return(dmSettings(uuid.New()), nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets, Most: most})

	_, err := svc.PluginListMine(context.Background(), dmScopeFixture())
	assert.ErrorIs(t, err, models.ErrChannelNotBound)
}
