package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

type TicketAccessChecker interface {
	CheckAccess(ctx context.Context, ticketID, userID uuid.UUID, action string, realm string) error
	CheckWorkAccess(ctx context.Context, ticketID, userID uuid.UUID, realm string) error
}

func (s *TicketService) CheckAccess(ctx context.Context, ticketID, userID uuid.UUID, action string, realm string) error {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceTicket), action)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return nil
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket for access check: %w", err)
	}
	if ticket.Group == nil {
		if action == string(access.Read) && (ticket.Assignee != nil && ticket.Assignee.ID == userID || ticket.Creator.ID == userID) {
			return nil
		}
		return models.ErrPermissionDenied
	}

	switch action {
	case string(access.Read):
		isMember, err := s.groups.IsMember(ctx, ticket.Group.ID, userID)
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if isMember {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
		if ticket.Assignee != nil && ticket.Assignee.ID == userID {
			return nil
		}
	case string(access.Write):
		if ticket.Creator.ID == userID {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	case string(access.Delete):
		managed, err := s.groups.GetManagedGroups(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	}
	return models.ErrPermissionDenied
}

func (s *TicketService) CheckWorkAccess(ctx context.Context, ticketID, userID uuid.UUID, realm string) error {
	if err := s.CheckAccess(ctx, ticketID, userID, string(access.Write), realm); err == nil {
		return nil
	}
	ticket, ticketErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: ticketID})
	if ticketErr != nil {
		return ticketErr
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == userID {
		return nil
	}
	return models.ErrPermissionDenied
}
