package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

func (s *userService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserDataDTO) error {
	if len(dto) == 0 {
		return nil
	}
	if err := s.repo.CreateSeveral(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create few users. error: %w", err)
	}
	return nil
}

func (s *userService) UpdateAccount(ctx context.Context, dto *models.UpdateAccountDTO) error {
	candidate, err := s.GetByID(ctx, dto.ID)
	if err != nil {
		return err
	}

	var oldRealms []*models.UserRealm
	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		var err error
		oldRealms, err = s.userRealm.GetByUserID(ctx, dto.ID)
		if err != nil {
			return fmt.Errorf("failed to get user realms: %w", err)
		}

		if err := s.repo.UpdateAccount(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update user account: %w", err)
		}

		if len(dto.Realms) > 0 {
			created := []*models.UserRealmDTO{}
			updated := []*models.UserRealmDTO{}
			deleted := []*models.UserRealmDTO{}
			for _, r := range dto.Realms {
				if r.CreatedAt == "" {
					created = append(created, r)
				} else if r.RoleID != nil {
					updated = append(updated, r)
				} else {
					deleted = append(deleted, r)
				}
			}

			if err := s.userRealm.CreateSeveral(ctx, tx, created); err != nil {
				return fmt.Errorf("failed to create user realms: %w", err)
			}
			if err := s.userRealm.UpdateSeveral(ctx, tx, updated); err != nil {
				return fmt.Errorf("failed to update user realms: %w", err)
			}
			if err := s.userRealm.DeleteSeveral(ctx, tx, deleted); err != nil {
				return fmt.Errorf("failed to delete user realms: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	event := events.PolicyEvent{
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		Action:        "update_user",
		EntityType:    "users",
		Entity:        &candidate.Username,
		EntityID:      &dto.ID,
	}

	if err := event.SetOldValues(map[string]any{
		"isActive":       candidate.IsActive,
		"mattermostId":   candidate.MattermostID,
		"internalNumber": candidate.InternalNumber,
		"realms":         oldRealms,
	}); err != nil {
		return fmt.Errorf("failed to set old values: %w", err)
	}

	if err := event.SetNewValues(map[string]any{
		"isActive":       dto.IsActive,
		"mattermostId":   dto.MattermostID,
		"internalNumber": dto.InternalNumber,
		"realms":         dto.Realms,
	}); err != nil {
		return fmt.Errorf("failed to set new values: %w", err)
	}

	s.eventBus.Notify(event)

	return nil
}

func (s *userService) UpdateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.UserDataDTO) error {
	if len(dto) == 0 {
		return nil
	}
	if err := s.repo.UpdateSeveral(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to update few users. error: %w", err)
	}
	return nil
}

func (s *userService) DeleteSeveral(ctx context.Context, tx postgres.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.repo.DeleteSeveral(ctx, tx, ids); err != nil {
		return fmt.Errorf("failed to delete few users. error: %w", err)
	}
	return nil
}
