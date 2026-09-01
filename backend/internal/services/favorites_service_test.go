package services

import (
	"context"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func favoriteServiceFixtures() (*MockTicketFavoritesRepo, *MockTicketsRepo, *MockTicketAccessChecker, *MockAccessPolicies, *TicketFavoritesService) {
	mockRepo := new(MockTicketFavoritesRepo)
	mockTickets := new(MockTicketsRepo)
	mockAccess := new(MockTicketAccessChecker)
	mockPolicies := new(MockAccessPolicies)

	svc := NewTicketFavoritesService(mockRepo, mockTickets, mockAccess, mockPolicies)
	return mockRepo, mockTickets, mockAccess, mockPolicies, svc
}

func TestFavoriteService_Add_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, _, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: "",
	}).Return(nil)
	mockRepo.On("Add", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Add(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypePermanent,
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_Add_Denied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, _, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

	err := svc.Add(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypePermanent,
	})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Remove_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, _, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetByID", mock.Anything, &models.GetTicketByIdDTO{ID: ticketID}).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccess", mock.Anything, &models.AccessCheckDTO{
		TicketID: ticketID, UserID: userID, Action: string(access.Read), Realm: realmID.String(),
	}).Return(nil)
	mockRepo.On("Remove", mock.Anything, nil, ticketID, userID, models.FavoriteTypeTemporary).Return(nil)

	err := svc.Remove(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypeTemporary,
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_IsFavorite(t *testing.T) {
	mockRepo, _, _, _, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockRepo.On("Exists", mock.Anything, ticketID, userID, models.FavoriteTypePermanent).Return(true, nil)

	ok, err := svc.IsFavorite(context.Background(), &models.IsFavoriteDTO{TicketID: ticketID, UserID: userID}, models.FavoriteTypePermanent)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestFavoriteService_CleanupTemporary_ResolvedAssignee(t *testing.T) {
	mockRepo, _, _, _, svc := favoriteServiceFixtures()

	assigneeID := uuid.New()
	favID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: favID, UserID: assigneeID, TicketID: uuid.New(),
		Status: models.StatusResolved, AssigneeID: &assigneeID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)
	mockRepo.On("DeleteByIDs", mock.Anything, nil, []uuid.UUID{favID}).Return(nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_ClosedSupervisor(t *testing.T) {
	mockRepo, _, _, mockPolicies, svc := favoriteServiceFixtures()

	supervisorID := uuid.New()
	realmID := uuid.New()
	favID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: favID, UserID: supervisorID, TicketID: uuid.New(),
		Status: models.StatusClosed, RealmID: &realmID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)
	mockPolicies.On("Enforce", supervisorID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(true, nil)
	mockRepo.On("DeleteByIDs", mock.Anything, nil, []uuid.UUID{favID}).Return(nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_ClosedNotSupervisor(t *testing.T) {
	mockRepo, _, _, mockPolicies, svc := favoriteServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: uuid.New(), UserID: userID, TicketID: uuid.New(),
		Status: models.StatusClosed, RealmID: &realmID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceCategory), string(access.Write)).Return(false, nil)
	mockPolicies.On("Enforce", userID.String(), realmID.String(), string(access.ResourceSite), string(access.Write)).Return(false, nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertNotCalled(t, "DeleteByIDs", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_ResolvedNotAssignee(t *testing.T) {
	mockRepo, _, _, _, svc := favoriteServiceFixtures()

	assigneeID := uuid.New()
	otherID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: uuid.New(), UserID: otherID, TicketID: uuid.New(),
		Status: models.StatusResolved, AssigneeID: &assigneeID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertNotCalled(t, "DeleteByIDs", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_None(t *testing.T) {
	mockRepo, _, _, _, svc := favoriteServiceFixtures()

	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{}, nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertNotCalled(t, "DeleteByIDs", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}
