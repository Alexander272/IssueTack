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
// заявки как «избранное» (permanent) или «закреплённые» (temporary, авто-очищаются джобом).
type TicketFavoritesService struct {
	repo         repository.TicketFavorites
	ticketRepo   repository.Tickets
	ticketAccess TicketAccessChecker
	policies     AccessPolicies
}

func NewTicketFavoritesService(repo repository.TicketFavorites, ticketRepo repository.Tickets, ticketAccess TicketAccessChecker, policies AccessPolicies) *TicketFavoritesService {
	return &TicketFavoritesService{
		repo:         repo,
		ticketRepo:   ticketRepo,
		ticketAccess: ticketAccess,
		policies:     policies,
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
	ticket, err := s.ticketRepo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return nil, err
	}

	realm := ""
	if ticket.RealmID != nil {
		realm = ticket.RealmID.String()
	}

	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: ticketID,
		UserID:   userID,
		Action:   string(access.Read),
		Realm:    realm,
	}); err != nil {
		return nil, err
	}
	return ticket, nil
}

// CleanupTemporary удаляет автоматически истёкшие «закреплённые» (temporary) избранные:
//   - для заявки в статусе resolved — у её исполнителя (assignee);
//   - для заявки в статусе closed — у «менеджеров и выше» (realm supervisor).
//
// Permanent-избранные не затрагиваются. Возвращает число удалённых записей.
func (s *TicketFavoritesService) CleanupTemporary(ctx context.Context) (int, error) {
	views, err := s.repo.GetTemporaryExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get temporary favorites: %w", err)
	}

	if len(views) == 0 {
		return 0, nil
	}

	// Мемо-кэш проверок «начальник области» за один проход джоба.
	supervisorCache := map[string]bool{}

	var toDelete []uuid.UUID
	for _, v := range views {
		switch v.Status {
		case models.StatusResolved:
			if v.AssigneeID != nil && *v.AssigneeID == v.UserID {
				toDelete = append(toDelete, v.ID)
			}
		case models.StatusClosed:
			supervisor, err := s.isSupervisorCached(ctx, v.UserID, v.RealmID, supervisorCache)
			if err != nil {
				return 0, fmt.Errorf("supervisor check failed: %w", err)
			}
			if supervisor {
				toDelete = append(toDelete, v.ID)
			}
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	if err := s.repo.DeleteByIDs(ctx, nil, toDelete); err != nil {
		return 0, fmt.Errorf("failed to delete temporary favorites: %w", err)
	}
	return len(toDelete), nil
}

// isSupervisorCached определяет, является ли пользователь начальником области в реалме, кэшируя результат.
func (s *TicketFavoritesService) isSupervisorCached(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID, cache map[string]bool) (bool, error) {
	realm := ""
	if realmID != nil {
		realm = realmID.String()
	}

	key := userID.String() + "|" + realm
	if v, ok := cache[key]; ok {
		return v, nil
	}

	ok, err := isRealmSupervisor(s.policies, userID, realm)
	if err != nil {
		return false, err
	}
	cache[key] = ok
	return ok, nil
}
