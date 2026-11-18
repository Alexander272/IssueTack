package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// TicketFavoritesService — сервис управления избранными заявками. Пользователь может помечать
// заявки как «избранное» (permanent) или «закреплённые» (temporary, снимаются при заморозке
// заявки, а джоб-очистка остаётся страховкой).
type TicketFavoritesService struct {
	repo         repository.TicketFavorites
	tickets      Tickets
	ticketAccess TicketAccessChecker
}

func NewTicketFavoritesService(repo repository.TicketFavorites, tickets Tickets, ticketAccess TicketAccessChecker) *TicketFavoritesService {
	return &TicketFavoritesService{
		repo:         repo,
		tickets:      tickets,
		ticketAccess: ticketAccess,
	}
}

// TicketFavorites — интерфейс управления избранными заявками.
type TicketFavorites interface {
	Add(ctx context.Context, dto *models.FavoriteDTO) error
	Remove(ctx context.Context, dto *models.FavoriteDTO) error
	IsFavorite(ctx context.Context, dto *models.IsFavoriteDTO, favoriteType models.FavoriteType) (bool, error)
	GetByUser(ctx context.Context, userID uuid.UUID, favoriteType models.FavoriteType) ([]*models.TicketFavorite, error)
	CleanupTemporary(ctx context.Context) (int, error)
}

// Add добавляет заявку в избранное, предварительно проверив read-доступ пользователя.
// «Закрепление» (temporary) заявок в «замороженных» статусах (resolved/closed/cancelled)
// запрещено; permanent-избранное разрешено в любом статусе.
func (s *TicketFavoritesService) Add(ctx context.Context, dto *models.FavoriteDTO) error {
	ticket, err := s.checkReadAccess(ctx, dto.TicketID, dto.ActorID)
	if err != nil {
		return err
	}
	if isTicketInactive(ticket.Status) && dto.Type == models.FavoriteTypeTemporary {
		return models.ErrTicketFrozen
	}
	dto.UserID = dto.ActorID
	return s.repo.Add(ctx, nil, dto)
}

// Remove убирает заявку из избранного, предварительно проверив read-доступ пользователя.
func (s *TicketFavoritesService) Remove(ctx context.Context, dto *models.FavoriteDTO) error {
	if _, err := s.checkReadAccess(ctx, dto.TicketID, dto.ActorID); err != nil {
		return err
	}
	return s.repo.Remove(ctx, nil, dto.TicketID, dto.ActorID, dto.Type)
}

// IsFavorite проверяет, есть ли у пользователя избранное заданного типа по заявке.
func (s *TicketFavoritesService) IsFavorite(ctx context.Context, dto *models.IsFavoriteDTO, favoriteType models.FavoriteType) (bool, error) {
	return s.repo.Exists(ctx, dto.TicketID, dto.UserID, favoriteType)
}

// GetByUser возвращает избранные заявки пользователя заданного типа.
func (s *TicketFavoritesService) GetByUser(ctx context.Context, userID uuid.UUID, favoriteType models.FavoriteType) ([]*models.TicketFavorite, error) {
	return s.repo.GetByUser(ctx, userID, favoriteType)
}

// checkReadAccess загружает тикет, проверяет у пользователя read-доступ и возвращает тикет.
func (s *TicketFavoritesService) checkReadAccess(ctx context.Context, ticketID, userID uuid.UUID) (*models.Ticket, error) {
	ticket, err := s.tickets.GetSummary(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if err := s.ticketAccess.CheckAccessOnTicket(ctx, ticket, userID, string(access.Read)); err != nil {
		return nil, err
	}
	return ticket, nil
}

// CleanupTemporary — страховочная очистка «закреплённых» (temporary): снимает их
// со всех заявок в «замороженных» статусах (resolved/closed/cancelled). Основная
// очистка выполняется в момент смены статуса (TicketService.Update и
// AutoCloseResolved), джоб нужен для строк, оставшихся от прежних версий или
// пропущенных запусков. Permanent-избранные не затрагиваются.
func (s *TicketFavoritesService) CleanupTemporary(ctx context.Context) (int, error) {
	n, err := s.repo.DeleteTemporaryOnInactive(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup temporary favorites: %w", err)
	}
	return int(n), nil
}
