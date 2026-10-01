package services

import (
	"context"
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

// ownerFixture — сервис для тестов выбора заказчика в диалоге создания заявки.
type ownerFixture struct {
	svc        *MattermostService
	repo       *MockMattermostRepo
	users      *MockUserService
	userRealms *MockUserRealmsService
	tickets    *MockTicketsService
	capture    *mmCapture
	realmID    uuid.UUID
	userID     uuid.UUID
}

// ownerRole описывает, какие группы видит нажавший: от этого зависит, показываем
// ли мы поле «Заказчик».
type ownerRole struct {
	isSupervisor  bool
	managedGroups []uuid.UUID
	memberGroups  []uuid.UUID
}

type ownerOpts struct {
	role ownerRole
	// customers — активные заказчики realm (вне групп).
	customers []*models.UserData
	// allCustomers — весь ответ GetByMembership, включая неактивных.
	allCustomers []*models.UserData
	mmUserID     string
}

func ownerService(t *testing.T, opts ownerOpts) *ownerFixture {
	t.Helper()

	realmID := uuid.New()
	userID := uuid.New()
	mmUserID := opts.mmUserID
	if mmUserID == "" {
		mmUserID = "mm-creator"
	}

	repo := new(MockMattermostRepo)
	users := new(MockUserService)
	userRealms := new(MockUserRealmsService)
	tickets := new(MockTicketsService)
	groups := &ownerGroupsSvc{role: opts.role}
	access := new(MockTicketAccessChecker)

	repo.On("GetByRealm", mock.Anything, realmID).Return(&models.RealmMattermost{
		RealmID: realmID, BotToken: "tok", BotUserID: "bot1", IsActive: true,
	}, nil)
	repo.On("GetByRealm", mock.Anything, mock.Anything).Return(
		&models.RealmMattermost{RealmID: realmID, BotToken: "tok", BotUserID: "bot1", IsActive: true}, nil)

	// Нажавший уже известен системе и уже состоит в realm.
	users.On("GetByMattermostID", mock.Anything, mmUserID).Return(
		&models.UserData{ID: userID, Username: "creator"}, nil)
	userRealms.On("GetByUserAndRealm", mock.Anything, userID, realmID).
		Return(&models.UserRealm{UserID: userID, RealmID: realmID}, nil)

	all := opts.allCustomers
	if all == nil {
		all = opts.customers
	}
	users.On("GetByMembership", mock.Anything, realmID, models.MembershipCustomers).Return(all, nil)

	access.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(opts.role.isSupervisor, nil)

	tickets.On("UploadAttachment", mock.Anything, mock.Anything, mock.Anything).
		Return(&models.Attachment{ID: uuid.New(), FileName: "f.txt"}, nil)

	var posts []capturedPost
	capture := &mmCapture{posts: &posts}
	srv := mmServer(t, capture)

	svc := NewMattermostService(&MattermostDeps{
		Repo:       repo,
		Users:      users,
		UserRealms: userRealms,
		Tickets:    tickets,
		Groups:     groups,
		Categories: &fakeCategoriesSvc{
			get: func(context.Context, *models.GetCategoriesDTO) ([]*models.Category, error) {
				return nil, nil
			},
		},
		Sites: &fakeSitesSvc{
			get: func(context.Context, *models.GetSitesDTO) ([]*models.Site, error) {
				return nil, nil
			},
		},
		Access:   access,
		EventBus: &events.PolicyEventManager{},
		Most:     mattermost.NewMost(mattermost.MostConfig{ServerURL: srv.URL, BaseURL: testBaseURL}),
		BaseURL:  testBaseURL,
	})

	return &ownerFixture{svc: svc, repo: repo, users: users, userRealms: userRealms,
		tickets: tickets, capture: capture, realmID: realmID, userID: userID}
}

// ownerGroupsSvc отдаёт группы по сценарию теста.
type ownerGroupsSvc struct {
	Groups
	role ownerRole
}

func (g *ownerGroupsSvc) GetManagedGroups(context.Context, uuid.UUID, *uuid.UUID) ([]uuid.UUID, error) {
	if g.role.managedGroups == nil {
		return []uuid.UUID{}, nil
	}
	return g.role.managedGroups, nil
}

func (g *ownerGroupsSvc) GetMemberGroups(context.Context, uuid.UUID, *uuid.UUID) ([]uuid.UUID, error) {
	if g.role.memberGroups == nil {
		return []uuid.UUID{}, nil
	}
	return g.role.memberGroups, nil
}

// openDialog открывает диалог создания заявки от имени нажавшего и возвращает
// его поля.
func (f *ownerFixture) openDialog(t *testing.T) []model.DialogElement {
	t.Helper()
	err := f.svc.HandleDialogOpen(context.Background(), &models.DialogOpenDTO{
		TriggerID: "tr1",
		UserID:    "mm-creator",
		ChannelID: "ch1",
		Context:   map[string]string{"realm_id": f.realmID.String()},
	})
	require.NoError(t, err)
	require.Len(t, f.capture.dialogs, 1, "диалог должен быть открыт ровно один раз")
	return f.capture.dialogs[0].Dialog.Elements
}

// dialogElement ищет поле диалога по имени.
func dialogElement(t *testing.T, elements []model.DialogElement, name string) *model.DialogElement {
	t.Helper()
	for i, el := range elements {
		if el.Name == name {
			return &elements[i]
		}
	}
	return nil
}

// submitDialog отправляет заполненный диалог и возвращает созданную заявку.
func (f *ownerFixture) submitDialog(t *testing.T, submission map[string]interface{}) *models.TicketDTO {
	t.Helper()

	var captured *models.TicketDTO
	f.tickets.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			dto := args.Get(1).(*models.TicketDTO)
			if dto.ID == nil {
				id := uuid.New()
				dto.ID = &id
			}
			captured = dto
		}).Return(nil)

	submission["title"] = "Сломалась установка"
	err := f.svc.HandleDialogSubmission(context.Background(), &model.SubmitDialogRequest{
		CallbackId: f.realmID.String(),
		UserId:     "mm-creator",
		ChannelId:  "ch1",
		Submission: submission,
	})
	require.NoError(t, err)
	require.NotNil(t, captured, "заявка должна быть создана")
	return captured
}

func TestDialogOpen_OwnerSelect_ForManager(t *testing.T) {
	customer := &models.UserData{
		ID: uuid.New(), Username: "client", FirstName: "Анна", LastName: "Петрова", IsActive: true,
	}
	f := ownerService(t, ownerOpts{
		role:      ownerRole{isSupervisor: true},
		customers: []*models.UserData{customer},
	})

	el := dialogElement(t, f.openDialog(t), "ownerId")
	require.NotNil(t, el, "администратор realm видит выбор заказчика")
	assert.Equal(t, "Заказчик", el.DisplayName)
	assert.Equal(t, "select", el.Type)
	require.Len(t, el.Options, 1)
	assert.Equal(t, "Петрова Анна (client)", el.Options[0].Text, "подпись как в веб-форме")
	assert.Equal(t, customer.ID.String(), el.Options[0].Value)
}

func TestDialogOpen_OwnerSelect_ForGroupManagerAndExecutor(t *testing.T) {
	customer := &models.UserData{ID: uuid.New(), Username: "client", LastName: "Петрова", IsActive: true}

	for _, tt := range []struct {
		name string
		role ownerRole
	}{
		{name: "менеджер группы", role: ownerRole{managedGroups: []uuid.UUID{uuid.New()}}},
		{name: "участник группы", role: ownerRole{memberGroups: []uuid.UUID{uuid.New()}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := ownerService(t, ownerOpts{role: tt.role, customers: []*models.UserData{customer}})
			assert.NotNil(t, dialogElement(t, f.openDialog(t), "ownerId"),
				"менеджеры и исполнители видят выбор заказчика, как в веб-форме")
		})
	}
}

func TestDialogOpen_NoOwnerSelect_ForPlainRequester(t *testing.T) {
	f := ownerService(t, ownerOpts{
		role:      ownerRole{}, // ни администратор, ни групп
		customers: []*models.UserData{{ID: uuid.New(), Username: "client", IsActive: true}},
	})

	assert.Nil(t, dialogElement(t, f.openDialog(t), "ownerId"),
		"чистому заявителю поле не нужно — заявка создаётся на него")
}

func TestDialogOpen_NoOwnerSelect_WhenNoCustomers(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts ownerOpts
	}{
		{name: "заказчиков нет", opts: ownerOpts{role: ownerRole{isSupervisor: true}}},
		{name: "все заказчики неактивны", opts: ownerOpts{
			role: ownerRole{isSupervisor: true},
			allCustomers: []*models.UserData{
				{ID: uuid.New(), Username: "gone", IsActive: false},
			},
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := ownerService(t, tt.opts)
			assert.Nil(t, dialogElement(t, f.openDialog(t), "ownerId"),
				"неактивных заказчиков в список не берём и пустой select не показываем")
		})
	}
}

func TestDialogOpen_OwnerSelect_SkipsInactive(t *testing.T) {
	active := &models.UserData{ID: uuid.New(), Username: "active", LastName: "Актив", IsActive: true}
	inactive := &models.UserData{ID: uuid.New(), Username: "inactive", LastName: "Ушедший", IsActive: false}
	f := ownerService(t, ownerOpts{
		role:         ownerRole{isSupervisor: true},
		allCustomers: []*models.UserData{active, inactive},
	})

	el := dialogElement(t, f.openDialog(t), "ownerId")
	require.NotNil(t, el)
	require.Len(t, el.Options, 1)
	assert.Equal(t, active.ID.String(), el.Options[0].Value)
}

func TestDialogOpen_OwnerSelect_PlacedBeforeCategory(t *testing.T) {
	f := ownerService(t, ownerOpts{
		role:      ownerRole{isSupervisor: true},
		customers: []*models.UserData{{ID: uuid.New(), Username: "client", IsActive: true}},
	})

	elements := f.openDialog(t)
	require.Len(t, elements, 3)
	assert.Equal(t, []string{"title", "description", "ownerId"},
		[]string{elements[0].Name, elements[1].Name, elements[2].Name})
}

func TestDialogSubmit_SetsOwnerFromSelection(t *testing.T) {
	customer := &models.UserData{ID: uuid.New(), Username: "client", LastName: "Петрова", IsActive: true}
	f := ownerService(t, ownerOpts{
		role:      ownerRole{isSupervisor: true},
		customers: []*models.UserData{customer},
	})

	dto := f.submitDialog(t, map[string]interface{}{"ownerId": customer.ID.String()})
	require.NotNil(t, dto.OwnerID)
	assert.Equal(t, customer.ID, *dto.OwnerID, "заказчик из диалога сохраняется в заявке")
	assert.Equal(t, f.userID, dto.CreatorID, "создатель остаётся автором")
}

func TestDialogSubmit_IgnoresOwnerOutsideRealmCustomers(t *testing.T) {
	customer := &models.UserData{ID: uuid.New(), Username: "client", IsActive: true}
	f := ownerService(t, ownerOpts{
		role:      ownerRole{isSupervisor: true},
		customers: []*models.UserData{customer},
	})

	// Чужой пользователь (например, из другого realm) в список заказчиков не входит.
	dto := f.submitDialog(t, map[string]interface{}{"ownerId": uuid.New().String()})
	assert.Nil(t, dto.OwnerID, "подменённый заказчик отбрасывается, заявка создаётся на создателя")
	assert.Equal(t, f.userID, dto.CreatorID)
}

func TestDialogSubmit_IgnoresInactiveOwner(t *testing.T) {
	active := &models.UserData{ID: uuid.New(), Username: "active", IsActive: true}
	inactive := &models.UserData{ID: uuid.New(), Username: "inactive", IsActive: false}
	f := ownerService(t, ownerOpts{
		role:         ownerRole{isSupervisor: true},
		allCustomers: []*models.UserData{active, inactive},
	})

	dto := f.submitDialog(t, map[string]interface{}{"ownerId": inactive.ID.String()})
	assert.Nil(t, dto.OwnerID, "неактивного заказчика выбрать нельзя")
}

func TestDialogSubmit_IgnoresOwnerFromSubmitterWithoutPicker(t *testing.T) {
	customer := &models.UserData{ID: uuid.New(), Username: "client", IsActive: true}
	// Заявитель без групп: у него в диалоге нет поля «Заказчик», но подделав
	// ownerId в теле сабмита он не должен получить право назначить заказчика.
	f := ownerService(t, ownerOpts{
		role:      ownerRole{},
		customers: []*models.UserData{customer},
	})

	dto := f.submitDialog(t, map[string]interface{}{"ownerId": customer.ID.String()})
	assert.Nil(t, dto.OwnerID, "подделанный ownerId от заявителя без селекта отбрасывается")
	assert.Equal(t, f.userID, dto.CreatorID, "заявка уходит на самого заявителя")
}

func TestDialogSubmit_OwnerDefaultsToCreator(t *testing.T) {
	for _, tt := range []struct {
		name       string
		submission map[string]interface{}
	}{
		{name: "поле не прислано", submission: map[string]interface{}{}},
		{name: "пустое значение", submission: map[string]interface{}{"ownerId": ""}},
		{name: "не uuid", submission: map[string]interface{}{"ownerId": "not-a-uuid"}},
		{name: "чужой тип", submission: map[string]interface{}{"ownerId": 42}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := ownerService(t, ownerOpts{
				role:      ownerRole{isSupervisor: true},
				customers: []*models.UserData{{ID: uuid.New(), Username: "client", IsActive: true}},
			})

			dto := f.submitDialog(t, tt.submission)
			assert.Nil(t, dto.OwnerID,
				"без корректного выбора OwnerID остаётся пустым — сервис подставит создателя")
		})
	}
}

func TestCustomerLabel(t *testing.T) {
	for _, tt := range []struct {
		name string
		user *models.UserData
		want string
	}{
		{name: "полные данные", user: &models.UserData{LastName: "Петрова", FirstName: "Анна", Username: "client"},
			want: "Петрова Анна (client)"},
		{name: "без имени", user: &models.UserData{LastName: "Петрова", Username: "client"},
			want: "Петрова (client)"},
		{name: "только логин", user: &models.UserData{Username: "client"}, want: "client"},
		{name: "без логина", user: &models.UserData{LastName: "Петрова"}, want: "Петрова"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, customerLabel(tt.user))
		})
	}
}
