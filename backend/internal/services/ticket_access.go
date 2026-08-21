package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
)

type TicketAccessChecker interface {
	CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error
}

func (s *TicketService) CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ok, err := s.policies.Enforce(dto.UserID.String(), dto.Realm, string(access.ResourceTicket), dto.Action)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return nil
	}

	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket for access check: %w", err)
	}
	if ticket.Group == nil {
		if dto.Action == string(access.Read) && (ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID || ticket.Creator.ID == dto.UserID) {
			return nil
		}
		return models.ErrPermissionDenied
	}

	switch dto.Action {
	case string(access.Read):
		isMember, err := s.groups.IsMember(ctx, ticket.Group.ID, dto.UserID)
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}
		if isMember {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, dto.UserID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
		if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
			return nil
		}
	case string(access.Write):
		if ticket.Creator.ID == dto.UserID {
			return nil
		}
		managed, err := s.groups.GetManagedGroups(ctx, dto.UserID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	case string(access.Delete):
		managed, err := s.groups.GetManagedGroups(ctx, dto.UserID, nil)
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

func (s *TicketService) CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	if err := s.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: dto.TicketID,
		UserID:   dto.UserID,
		Action:   string(access.Write),
		Realm:    dto.Realm,
	}); err == nil {
		return nil
	}
	ticket, ticketErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if ticketErr != nil {
		return ticketErr
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}
	return models.ErrPermissionDenied
}
