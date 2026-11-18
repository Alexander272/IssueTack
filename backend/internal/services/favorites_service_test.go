package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
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
	mockAccess.On("CheckAccessOnTicket", mock.Anything, &models.Ticket{ID: ticketID}, userID, string(access.Read)).Return(nil)
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
	mockAccess.On("CheckAccessOnTicket", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(models.ErrPermissionDenied)

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
	mockAccess.On("CheckAccessOnTicket", mock.Anything, &models.Ticket{ID: ticketID, RealmID: &realmID}, userID, string(access.Read)).Return(nil)
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

func TestFavoriteService_CleanupTemporary_DeletesOnInactive(t *testing.T) {
	mockRepo, _, _, svc := favoriteServiceFixtures()

	mockRepo.On("DeleteTemporaryOnInactive", mock.Anything).Return(int64(3), nil)

	n, err := svc.CleanupTemporary(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	mockRepo.AssertExpectations(t)
}

func TestFavoriteService_CleanupTemporary_Error(t *testing.T) {
	mockRepo, _, _, svc := favoriteServiceFixtures()

	mockRepo.On("DeleteTemporaryOnInactive", mock.Anything).Return(int64(0), errors.New("db is down"))

	n, err := svc.CleanupTemporary(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 0, n)
	mockRepo.AssertExpectations(t)
}
