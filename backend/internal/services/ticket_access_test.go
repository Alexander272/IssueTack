package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func accessServiceFixtures() (*MockGroupsRepo, *MockAccessPolicies, *TicketAccessService) {
	mockRepo := new(MockTicketsRepo)
	mockGroups := new(MockGroupsRepo)
	mockPolicies := new(MockAccessPolicies)
	svc := NewTicketAccessService(mockRepo, mockGroups, mockPolicies)
	return mockGroups, mockPolicies, svc
}

func TestTicketAccessService_IsRealmSupervisor_CategoryWrite(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, ok)
	mockPolicies.AssertNotCalled(t, "Enforce", mock.Anything, mock.Anything, string(access.ResourceSite), mock.Anything)
}

func TestTicketAccessService_IsRealmSupervisor_SiteWrite(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceSite), string(access.Write)).Return(true, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestTicketAccessService_IsRealmSupervisor_NoRights(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceSite), string(access.Write)).Return(false, nil)

	ok, err := svc.IsRealmSupervisor(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.False(t, ok)
	mockPolicies.AssertExpectations(t)
}

func TestTicketAccessService_CanManage_Supervisor(t *testing.T) {
	mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.True(t, allowed)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketAccessService_CanManage_GroupManager(t *testing.T) {
	mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{groupID}, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestTicketAccessService_CanManage_OtherGroupManager(t *testing.T) {
	mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	groupID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Group:   &models.GroupShort{ID: groupID},
	}
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)
	mockGroups.On("GetManagedGroups", mock.Anything, userID, &realmID).Return([]uuid.UUID{uuid.New()}, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestTicketAccessService_CanManage_GrouplessTicket(t *testing.T) {
	mockGroups, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: nil,
		Group:   nil,
	}
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), "", string(access.ResourceSite), string(access.Write)).Return(false, nil)

	allowed, err := svc.CanManage(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.False(t, allowed)
	mockGroups.AssertNotCalled(t, "GetManagedGroups", mock.Anything, mock.Anything, mock.Anything)
}

func TestTicketAccessService_CanCreateTicket(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceTicket), string(access.Write)).Return(true, nil)

	allowed, err := svc.CanCreateTicket(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.True(t, allowed)
	mockPolicies.AssertExpectations(t)
}

func TestTicketAccessService_CanCreateTicket_Denied(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realm := uuid.New().String()
	mockPolicies.On("Enforce", userID.String(), realm, string(access.ResourceTicket), string(access.Write)).Return(false, nil)

	allowed, err := svc.CanCreateTicket(context.Background(), userID, realm)
	assert.NoError(t, err)
	assert.False(t, allowed)
	mockPolicies.AssertExpectations(t)
}

// TestTicketAccessService_CheckAccessOnTicket_CrossRealmSupervisor — регрессия на IDOR:
// supervisor реалма A не должен получать доступ к тикету реалма B. Реалм для Casbin-домена
// берётся из ticket.RealmID, а не из клиентского realm-заголовка, поэтому подмена
// заголовка не помогает. Права supervisor'а выдаём ровно в его собственном реалме —
// домен тикета в Enforce не должен появляться вовсе.
func TestTicketAccessService_CheckAccessOnTicket_CrossRealmSupervisor(t *testing.T) {
	for _, action := range []access.ActionCode{access.Read, access.Write, access.Delete} {
		t.Run(string(action), func(t *testing.T) {
			_, mockPolicies, svc := accessServiceFixtures()

			userID := uuid.New()
			ownRealm := uuid.New().String()
			foreignRealmID := uuid.New()
			ticket := &models.Ticket{
				ID:      uuid.New(),
				RealmID: &foreignRealmID,
				Creator: models.UserShort{ID: uuid.New()},
			}

			// На домене тикета пользователь — не supervisor. Права supervisor'а в его
			// собственном реалме намеренно НЕ мокаем: если бы код подставил реалм
			// вызывающего, тест упал бы на «неожиданный вызов Enforce» с ownRealm.
			mockPolicies.On("Enforce", userID.String(), foreignRealmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
			mockPolicies.On("Enforce", userID.String(), foreignRealmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)

			err := svc.CheckAccessOnTicket(context.Background(), ticket, userID, string(action))
			assert.ErrorIs(t, err, models.ErrPermissionDenied)

			mockPolicies.AssertExpectations(t)
			for _, call := range mockPolicies.Calls {
				if call.Method == "Enforce" {
					assert.NotEqual(t, ownRealm, call.Arguments[1], "реалм тикета не должен подменяться реалмом вызывающего")
				}
			}
		})
	}
}

// TestTicketAccessService_CheckAccessOnTicket_SupervisorOwnRealm — контроль кейса выше:
// supervisor своего реалма доступ к тикету своего реалма получает (обход атрибутов).
func TestTicketAccessService_CheckAccessOnTicket_SupervisorOwnRealm(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	ticket := &models.Ticket{
		ID:      uuid.New(),
		RealmID: &realmID,
		Creator: models.UserShort{ID: uuid.New()},
	}

	// IsRealmSupervisor короткозамкнут на category:write, поэтому site не проверяется.
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)

	err := svc.CheckAccessOnTicket(context.Background(), ticket, userID, string(access.Write))
	assert.NoError(t, err)
	mockPolicies.AssertExpectations(t)
}

// TestTicketAccessService_CanAdministerTickets_UsesTicketRealm — регрессия:
// CanAdministerTickets проверяет supervisor по реалму ТИКЕТА, поэтому начальник
// чужого реалма не администрирует заявку (раньше домен приходил из аргумента).
func TestTicketAccessService_CanAdministerTickets_UsesTicketRealm(t *testing.T) {
	_, mockPolicies, svc := accessServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	ticket := &models.Ticket{ID: uuid.New(), RealmID: &realmID, Creator: models.UserShort{ID: uuid.New()}}

	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)

	ok, err := svc.CanAdministerTickets(context.Background(), userID, ticket)
	assert.NoError(t, err)
	assert.False(t, ok)
	mockPolicies.AssertExpectations(t)
}
