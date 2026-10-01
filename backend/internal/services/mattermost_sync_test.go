package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// syncFixture собирает сервис для тестов синка пользователей Mattermost.
// mmUsers/teamUsers задают ответы заглушки Mattermost, realmActive — состояние
// интеграции, isSupervisor — права вызывающего.
type syncFixture struct {
	svc        *MattermostService
	repo       *MockMattermostRepo
	users      *MockUserService
	userRealms *MockUserRealmsService
	realmID    uuid.UUID
	actor      *models.Actor
}

type syncFixtureOpts struct {
	mmUsers   []*model.User
	teamUsers map[string][]*model.User
	// sysUsers — системные пользователи, по которым ищется совпадение
	// (email/username/ФИО). По умолчанию пусто.
	sysUsers      []*models.UserData
	isActive      bool
	botToken      string
	isSupervisor  bool
	supervisorErr error
}

func syncService(t *testing.T, opts syncFixtureOpts) *syncFixture {
	t.Helper()

	realmID := uuid.New()
	actorID := uuid.New()
	actor := &models.Actor{ID: actorID, Name: "admin"}

	repo := new(MockMattermostRepo)
	users := new(MockUserService)
	userRealms := new(MockUserRealmsService)
	access := new(MockTicketAccessChecker)

	repo.On("GetByRealm", mock.Anything, realmID).Return(
		&models.RealmMattermost{RealmID: realmID, BotToken: opts.botToken, IsActive: opts.isActive}, nil)

	// С кем сопоставлять импортируемых пользователей.
	users.On("GetAll", mock.Anything, mock.Anything).Return(opts.sysUsers, nil)
	access.On("IsRealmSupervisor", mock.Anything, actorID, realmID.String()).
		Return(opts.isSupervisor, opts.supervisorErr)

	capture := &mmCapture{users: opts.mmUsers, teamUsers: opts.teamUsers}
	srv := mmServer(t, capture)

	svc := NewMattermostService(&MattermostDeps{
		Repo:       repo,
		Users:      users,
		UserRealms: userRealms,
		Access:     access,
		Roles:      &fakeRolesSvc{idBySlug: uuid.New()},
		EventBus:   &events.PolicyEventManager{},
		Most:       mattermost.NewMost(mattermost.MostConfig{ServerURL: srv.URL, BaseURL: testBaseURL}),
		BaseURL:    testBaseURL,
	})

	return &syncFixture{svc: svc, repo: repo, users: users, userRealms: userRealms,
		realmID: realmID, actor: actor}
}

// fakeRolesSvc отдаёт id роли «user» для ensureRealmMembership.
type fakeRolesSvc struct {
	Roles
	idBySlug uuid.UUID
}

func (f *fakeRolesSvc) GetIDBySlug(_ context.Context, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return f.idBySlug, nil
}

func TestSyncRealmUsers_NonSupervisor_Denied(t *testing.T) {
	f := syncService(t, syncFixtureOpts{isActive: true, botToken: "tok", isSupervisor: false})

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	assert.Nil(t, result, "без прав синк не выполняется и результата нет")
	f.repo.AssertNotCalled(t, "GetByRealm", mock.Anything, mock.Anything)
}

func TestSyncRealmUsers_NilActor_Denied(t *testing.T) {
	f := syncService(t, syncFixtureOpts{isActive: true, botToken: "tok", isSupervisor: true})

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	assert.Nil(t, result)
}

func TestSyncRealmUsers_InactiveIntegration_Denied(t *testing.T) {
	for _, tt := range []struct {
		name     string
		isActive bool
		botToken string
	}{
		{name: "интеграция выключена", isActive: false, botToken: "tok"},
		{name: "нет токена бота", isActive: true, botToken: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := syncService(t, syncFixtureOpts{isActive: tt.isActive, botToken: tt.botToken, isSupervisor: true})

			result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, models.ErrInvalidInput)
			assert.Nil(t, result)
		})
	}
}

func TestSyncRealmUsers_CreatesUnknownUsers(t *testing.T) {
	mmUsers := []*model.User{
		{Id: "mm1", Username: "ivanov", Email: "ivanov@example.com", FirstName: "Иван", LastName: "Иванов"},
		{Id: "mm2", Username: "petrova", Email: "petrova@example.com", FirstName: "Анна", LastName: "Петрова"},
	}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	var created []*models.UserDataDTO
	f.users.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = append(created, args.Get(2).([]*models.UserDataDTO)...)
		}).Return(nil)

	// Никого нет ни по mattermost_id, ни в realm — оба пользователя создаются.
	f.users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	f.userRealms.On("GetByUserAndRealm", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 0, result.Linked)
	assert.Equal(t, 0, result.Failed)

	require.Len(t, created, 2)
	usernames := []string{created[0].Username, created[1].Username}
	assert.ElementsMatch(t, []string{"ivanov", "petrova"}, usernames)

	for _, u := range created {
		assert.True(t, u.IsActive, "импортированный пользователь активен")
		require.NotNil(t, u.MattermostID)
		assert.NotEqual(t, uuid.Nil, u.ID, "MM-пользователю выдаётся свой id приложения")
		assert.Equal(t, models.UserSourceMattermost, u.Source,
			"импортированные из Mattermost пользователи защищены от Keycloak-синка")
	}
}

func TestSyncRealmUsers_LinksExistingByMattermostID(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm1", Username: "ivanov", Email: "ivanov@example.com"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	knownID := uuid.New()
	f.users.On("GetByMattermostID", mock.Anything, "mm1").Return(
		&models.UserData{ID: knownID, Username: "ivanov"}, nil)
	// Уже состоит в realm — привязка не создаётся заново.
	f.userRealms.On("GetByUserAndRealm", mock.Anything, knownID, f.realmID).
		Return(&models.UserRealm{UserID: knownID, RealmID: f.realmID}, nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Linked, "существующий пользователь засчитан как привязанный")
	assert.Equal(t, 0, result.Failed)

	f.users.AssertNotCalled(t, "CreateSeveral", mock.Anything, mock.Anything, mock.Anything)
	f.userRealms.AssertNotCalled(t, "CreateSeveral", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncRealmUsers_LinksExistingByEmail(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm-new", Username: "ivanov", Email: "ivanov@example.com"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	matchedID := uuid.New()
	// Системный пользователь без mattermost_id: сопоставляем по email.
	f = syncService(t, syncFixtureOpts{
		mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true,
		sysUsers: []*models.UserData{{ID: matchedID, Username: "ivanov", Email: "ivanov@example.com"}},
	})
	f.users.On("GetByMattermostID", mock.Anything, "mm-new").Return(nil, errors.New("not found"))
	f.users.On("UpdateMMAndSite", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, matchedID, f.realmID).
		Return(&models.UserRealm{UserID: matchedID, RealmID: f.realmID}, nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Linked)
	f.users.AssertCalled(t, "UpdateMMAndSite", mock.Anything, mock.Anything,
		mock.MatchedBy(func(d *models.UserDataDTO) bool {
			return d.ID == matchedID && d.MattermostID != nil && *d.MattermostID == "mm-new"
		}))
}

func TestSyncRealmUsers_LinksExistingByFIO(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm-new", Username: "ivanov", FirstName: "Иван", LastName: "Иванов"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	matchedID := uuid.New()
	// Ни email, ни username не совпали — сопоставляем по ФИО.
	f = syncService(t, syncFixtureOpts{
		mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true,
		sysUsers: []*models.UserData{{ID: matchedID, Username: "somebody", FirstName: "Иван", LastName: "Иванов"}},
	})
	f.users.On("GetByMattermostID", mock.Anything, "mm-new").Return(nil, errors.New("not found"))
	f.users.On("UpdateMMAndSite", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, matchedID, f.realmID).
		Return(&models.UserRealm{UserID: matchedID, RealmID: f.realmID}, nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Linked)
	f.users.AssertNotCalled(t, "CreateSeveral", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncRealmUsers_AddsMissingRealmMembership(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm1", Username: "ivanov"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	knownID := uuid.New()
	f.users.On("GetByMattermostID", mock.Anything, "mm1").Return(
		&models.UserData{ID: knownID, Username: "ivanov"}, nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, knownID, f.realmID).
		Return(nil, errors.New("not a member"))
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created, "пользователь уже был в системе — только добавлен в realm")
	assert.Equal(t, 1, result.Linked)
	f.userRealms.AssertCalled(t, "CreateSeveral", mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncRealmUsers_CountsFailures(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm1", Username: "ivanov"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	knownID := uuid.New()
	f.users.On("GetByMattermostID", mock.Anything, "mm1").Return(
		&models.UserData{ID: knownID, Username: "ivanov"}, nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, knownID, f.realmID).
		Return(nil, errors.New("not a member"))
	// Не удалось получить роль «user» — пользователь создан, но в realm не добавлен.
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("db is down"))

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err, "частичные ошибки не проваляют весь синк")

	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 0, result.Linked)
	assert.Equal(t, 1, result.Failed, "проблемный пользователь учтён в счётчике ошибок")
}

func TestSyncRealmUsers_FailedCreate_CountsFailure(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm1", Username: "ivanov"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	f.users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	f.users.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db is down"))

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Failed)
}

// TestSyncRealmUsers_CreateThenMembershipFails_CountsOnlyFailure фиксирует, что
// счётчики Created/Failed взаимоисключающие: если пользователь создан, но в realm
// не добавлен, это одна неудача, а не «создан и провален» одновременно.
func TestSyncRealmUsers_CreateThenMembershipFails_CountsOnlyFailure(t *testing.T) {
	mmUsers := []*model.User{{Id: "mm1", Username: "ivanov"}}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	f.users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	f.users.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not a member"))
	// Роль «user» не нашлась — пользователь создан, но членство не выдано.
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("role not found"))

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Created, "недобавленный в realm пользователь не считается созданным")
	assert.Equal(t, 1, result.Failed, "неудача учтена ровно один раз")
}

func TestSyncRealmUsers_SkipsBotsDuplicatesAndDeactivated(t *testing.T) {
	mmUsers := []*model.User{
		{Id: "mm1", Username: "ivanov"},
		{Id: "mm1", Username: "ivanov"},
		{Id: "bot1", Username: "realmbot", IsBot: true},
		{Id: "mm-deleted", Username: "former", DeleteAt: 1712345678000},
	}
	f := syncService(t, syncFixtureOpts{mmUsers: mmUsers, isActive: true, botToken: "tok", isSupervisor: true})

	var created []*models.UserDataDTO
	f.users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	f.users.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = append(created, args.Get(2).([]*models.UserDataDTO)...)
		}).Return(nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not a member"))
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Created, "дубликат mm_id, бот и деактивированный пользователь не импортируются")
	require.Len(t, created, 1)
	assert.Equal(t, "ivanov", created[0].Username)
}

func TestSyncRealmUsers_SyncsByTeamNames(t *testing.T) {
	f := syncService(t, syncFixtureOpts{
		teamUsers: map[string][]*model.User{
			"support": {{Id: "mm1", Username: "ivanov"}},
			// Пользователь другой команды не должен попасть в синк этой команды.
			"other": {{Id: "mm2", Username: "petrova"}},
		},
		isActive:     true,
		botToken:     "tok",
		isSupervisor: true,
	})

	f.users.On("GetByMattermostID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
	f.users.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.userRealms.On("GetByUserAndRealm", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not a member"))
	f.userRealms.On("CreateSeveral", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := f.svc.SyncRealmUsers(context.Background(), f.realmID, f.actor, []string{"support"})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Created, "синк ограничен указанными командами")
}
