package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
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
	getByID func(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error)
}

func (f *fakeGroupsSvc) GetByID(ctx context.Context, req *models.GetGroupDTO) (*models.Group, error) {
	return f.getByID(ctx, req)
}

func pluginMocks() (*MockMattermostRepo, *MockUserService, *MockUserRealmsService, *MockTicketsService) {
	return new(MockMattermostRepo), new(MockUserService), new(MockUserRealmsService), new(MockTicketsService)
}

// expectExistingUser готовит моки пути «пользователь уже есть по mattermost_id».
func expectExistingUser(users *MockUserService, userRealms *MockUserRealmsService, userID, realmID uuid.UUID) {
	userSite := "a1b2c3d4-0000-0000-0000-000000000001"
	users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(&models.UserData{ID: userID, Username: "u1", SiteID: &userSite}, nil)
	users.On("UpdateMMAndSite", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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

	list, err := svc.PluginListMine(context.Background(), "ch1", "mm1")
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

	_, err := svc.PluginListMine(context.Background(), "ch1", "mm1")
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

	ctx, err := svc.PluginContext(context.Background(), "ch1", "mm1")
	require.NoError(t, err)
	require.NotNil(t, ctx)
	assert.False(t, ctx.Bound)
}

func TestPluginContext_InactiveChannel(t *testing.T) {
	realmID := uuid.New()
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: false}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	ctx, err := svc.PluginContext(context.Background(), "ch1", "mm1")
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

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms,
		Realms: realms, Categories: categories, Sites: sites,
	})

	result, err := svc.PluginContext(context.Background(), "ch1", "mm1")
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

	input := &PluginCreateTicketInput{
		ChannelID:   "ch1",
		MmUserID:    "mm1",
		Title:       "Новая заявка",
		Description: "Описание",
		CategoryID:  categoryID,
		Files: []PluginFile{
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

func TestPluginCreateTicket_UnboundChannel(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginCreateTicket(context.Background(), &PluginCreateTicketInput{
		ChannelID: "ch1", MmUserID: "mm1", Title: "Заявка",
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
		ID:            ticketID,
		Title:         "Заявка 9",
		Description:   "Описание заявки",
		Status:        models.StatusOpen,
		Priority:      models.PriorityMedium,
		TicketNumber:  &num,
		CreatedAt:     time.Now(),
		DueDate:       &due,
		Creator:       models.UserShort{ID: userID, Username: "mm1"},
		Category:      &models.CategoryShort{ID: uuid.New(), Name: "Категория"},
		Site:          &models.SiteShort{ID: uuid.New(), Name: "Площадка"},
	}, nil)

	svc := NewMattermostService(&MattermostDeps{
		Repo: repo, Users: users, UserRealms: userRealms, Tickets: tickets,
	})

	detail, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", ticketID.String())
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, ticketID, detail.ID)
	assert.Equal(t, num, detail.Number)
	assert.Equal(t, "Описание заявки", detail.Description)
	assert.Equal(t, models.StatusOpen, detail.Status)
	assert.Equal(t, "Площадка", detail.Site.Name)
	assert.Equal(t, userID, detail.Creator.ID)
	assert.Empty(t, detail.Link)
}

func TestPluginGetTicket_InvalidID(t *testing.T) {
	realmID := uuid.New()
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: realmID, IsActive: true}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	_, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", "not-a-uuid")
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
					User: &models.UserShort{ID: userID, Username: "mm1", FirstName: "Иван", LastName: "Иванов"},
					Attachments: []*models.Attachment{{ID: attID, FileName: "a.txt", FileSize: 3, MimeType: "text/plain"}}},
			}, nil
		},
	}

	svc := NewMattermostService(&MattermostDeps{Repo: repo, Users: users, UserRealms: userRealms, Comments: comments})

	list, err := svc.PluginGetComments(context.Background(), "ch1", "mm1", ticketID.String())
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

	out, err := svc.PluginCreateComment(context.Background(), &PluginCreateCommentInput{
		ChannelID: "ch1",
		MmUserID:  "mm1",
		TicketID:  ticketID.String(),
		Text:      "Всем привет",
		Files: []PluginFile{
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

	_, err := svc.PluginCreateComment(context.Background(), &PluginCreateCommentInput{
		ChannelID: "ch1", MmUserID: "mm1", TicketID: uuid.New().String(),
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

	got, rc, err := svc.PluginGetAttachmentContent(context.Background(), "ch1", "mm1", attID.String())
	require.NoError(t, err)
	defer rc.Close()
	assert.Equal(t, attID, got.ID)
	assert.Equal(t, "doc.pdf", got.FileName)
	data, _ := io.ReadAll(rc)
	assert.Equal(t, "content", string(data))

	attBody, _, err := svc.PluginGetAttachmentContent(context.Background(), "ch1", "mm1", "bad-id")
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

	detail, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", ticketID.String())
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

	detail, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", ticketID.String())
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

	detail, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", ticketID.String())
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

	detail, err := svc.PluginGetTicket(context.Background(), "ch1", "mm1", ticketID.String())
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

			err := svc.PluginChangeStatus(context.Background(), "ch1", "mm1", ticketID.String(), string(status))
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

	err := svc.PluginChangeStatus(context.Background(), "ch1", "mm1", ticketID.String(), string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
}

func TestPluginChangeStatus_Invalid(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(&models.RealmMattermost{RealmID: uuid.New(), IsActive: true}, nil)

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	err := svc.PluginChangeStatus(context.Background(), "ch1", "mm1", uuid.New().String(), "bogus")
	assert.ErrorIs(t, err, models.ErrInvalidInput)

	err = svc.PluginChangeStatus(context.Background(), "ch1", "mm1", "not-a-uuid", string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrInvalidInput)
}

func TestPluginChangeStatus_UnboundChannel(t *testing.T) {
	repo, _, _, _ := pluginMocks()
	repo.On("GetByChannelID", mock.Anything, "ch1").Return(nil, errors.New("no rows"))

	svc := NewMattermostService(&MattermostDeps{Repo: repo})

	err := svc.PluginChangeStatus(context.Background(), "ch1", "mm1", uuid.New().String(), string(models.StatusClosed))
	assert.ErrorIs(t, err, models.ErrChannelNotBound)
}
