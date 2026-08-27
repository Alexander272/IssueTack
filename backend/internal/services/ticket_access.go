package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

type TicketAccessChecker interface {
	CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error
	CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string, realm string) error
}

type TicketAccessService struct {
	repo     repository.Tickets
	groups   Groups
	policies AccessPolicies
}

func NewTicketAccessService(repo repository.Tickets, groups Groups, policies AccessPolicies) *TicketAccessService {
	return &TicketAccessService{
		repo:     repo,
		groups:   groups,
		policies: policies,
	}
}

// CheckAccessOnTicket performs an access check against an already loaded ticket.
// It is the shared core used by CheckAccess and by ticket status/capability logic.
func (s *TicketAccessService) CheckAccessOnTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string, realm string) error {
	ok, err := s.policies.Enforce(userID.String(), realm, string(access.ResourceTicket), action)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if ok {
		return nil
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
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
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
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err != nil {
			return fmt.Errorf("failed to check managed groups: %w", err)
		}
		for _, gid := range managed {
			if gid == ticket.Group.ID {
				return nil
			}
		}
	case string(access.Delete):
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
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

func (s *TicketAccessService) CheckAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
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
	return s.CheckAccessOnTicket(ctx, ticket, dto.UserID, dto.Action, dto.Realm)
}

func (s *TicketAccessService) CheckWorkAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	if err := s.CheckAccess(ctx, &models.AccessCheckDTO{
		TicketID: dto.TicketID,
		UserID:   dto.UserID,
		Action:   string(access.Write),
		Realm:    dto.Realm,
	}); err == nil {
		return nil
	}
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}
	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}
	return models.ErrPermissionDenied
}

func (s *TicketAccessService) CheckInternalAssigneeAccess(ctx context.Context, dto *models.AccessCheckDTO) error {
	ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.TicketID})
	if err != nil {
		return fmt.Errorf("failed to load ticket: %w", err)
	}

	if ticket.Assignee != nil && ticket.Assignee.ID == dto.UserID {
		return nil
	}

	if ticket.Group != nil {
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
