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

func favoriteServiceFixtures() (*MockTicketFavoritesRepo, *MockTicketsService, *MockTicketAccessChecker, *TicketFavoritesService) {
	mockRepo := new(MockTicketFavoritesRepo)
	mockTickets := new(MockTicketsService)
	mockAccess := new(MockTicketAccessChecker)

	svc := NewTicketFavoritesService(mockRepo, mockTickets, mockAccess)
	return mockRepo, mockTickets, mockAccess, svc
}

func TestFavoriteService_Add_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, &models.Ticket{ID: ticketID}, userID, string(access.Read), "").Return(nil)
	mockRepo.On("Add", mock.Anything, nil, mock.Anything).Return(nil)

	err := svc.Add(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypePermanent,
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_Add_Denied(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(
		&models.Ticket{ID: ticketID}, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

	err := svc.Add(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypePermanent,
	})
	assert.ErrorIs(t, err, models.ErrPermissionDenied)
	mockRepo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
}

func TestFavoriteService_Remove_Success(t *testing.T) {
	mockRepo, mockTickets, mockAccess, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()
	realmID := uuid.New()

	mockTickets.On("GetSummary", mock.Anything, ticketID).Return(
		&models.Ticket{ID: ticketID, RealmID: &realmID}, nil)
	mockAccess.On("CheckAccessOnTicket", mock.Anything, &models.Ticket{ID: ticketID, RealmID: &realmID}, userID, string(access.Read), realmID.String()).Return(nil)
	mockRepo.On("Remove", mock.Anything, nil, ticketID, userID, models.FavoriteTypeTemporary).Return(nil)

	err := svc.Remove(context.Background(), &models.FavoriteDTO{
		TicketID: ticketID, ActorID: userID, Type: models.FavoriteTypeTemporary,
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_IsFavorite(t *testing.T) {
	mockRepo, _, _, svc := favoriteServiceFixtures()

	ticketID := uuid.New()
	userID := uuid.New()

	mockRepo.On("Exists", mock.Anything, ticketID, userID, models.FavoriteTypePermanent).Return(true, nil)

	ok, err := svc.IsFavorite(context.Background(), &models.IsFavoriteDTO{TicketID: ticketID, UserID: userID}, models.FavoriteTypePermanent)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestFavoriteService_CleanupTemporary_ResolvedAssignee(t *testing.T) {
	mockRepo, _, _, svc := favoriteServiceFixtures()

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
	mockRepo, _, mockAccess, svc := favoriteServiceFixtures()

	supervisorID := uuid.New()
	realmID := uuid.New()
	favID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: favID, UserID: supervisorID, TicketID: uuid.New(),
		Status: models.StatusClosed, RealmID: &realmID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)
	mockAccess.On("IsRealmSupervisor", mock.Anything, supervisorID, realmID.String()).Return(true, nil)
	mockRepo.On("DeleteByIDs", mock.Anything, nil, []uuid.UUID{favID}).Return(nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_ClosedNotSupervisor(t *testing.T) {
	mockRepo, _, mockAccess, svc := favoriteServiceFixtures()

	userID := uuid.New()
	realmID := uuid.New()
	v := &postgres.TempFavoriteView{
		ID: uuid.New(), UserID: userID, TicketID: uuid.New(),
		Status: models.StatusClosed, RealmID: &realmID,
	}
	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{v}, nil)
	mockAccess.On("IsRealmSupervisor", mock.Anything, userID, realmID.String()).Return(false, nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertNotCalled(t, "DeleteByIDs", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_ResolvedNotAssignee(t *testing.T) {
	mockRepo, _, _, svc := favoriteServiceFixtures()

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
	mockRepo, _, _, svc := favoriteServiceFixtures()

	mockRepo.On("GetTemporaryExpired", mock.Anything).Return([]*postgres.TempFavoriteView{}, nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertNotCalled(t, "DeleteByIDs", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}
