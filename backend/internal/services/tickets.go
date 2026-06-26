package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type TicketService struct {
	repo          repository.Tickets
	tx            TransactionManager
	logs          ActivityLog
	subtasks      Subtasks
	attachments   Attachments
	notifications Notifications
	groups        Groups
	policies      AccessPolices
}

func NewTicketService(repo repository.Tickets, tx TransactionManager, logs ActivityLog, subtasks Subtasks, attachments Attachments, notifications Notifications, groups Groups, policies AccessPolices) *TicketService {
	return &TicketService{
		repo:          repo,
		tx:            tx,
		logs:          logs,
		subtasks:      subtasks,
		attachments:   attachments,
		notifications: notifications,
		groups:        groups,
		policies:      policies,
	}
}

type Tickets interface {
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error)
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	Create(ctx context.Context, dto *models.TicketDTO) error
	Update(ctx context.Context, dto *models.TicketDTO) error
	Delete(ctx context.Context, dto *models.DeleteTicketDTO) error
}

type TicketAccessChecker interface {
	CheckAccess(ctx context.Context, ticketID, userID uuid.UUID, action string, realm ...string) error
}

func (s *TicketService) Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error) {
	realmStr := ""
	if req.RealmID != nil {
		realmStr = req.RealmID.String()
	}

	elevated, err := s.policies.Enforce(req.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return nil, 0, fmt.Errorf("policy check failed: %w", err)
	}
	if !elevated {
		elevated, err = s.policies.Enforce(req.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Delete))
		if err != nil {
			return nil, 0, fmt.Errorf("policy check failed: %w", err)
		}
	}

	if !elevated {
		managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
		}
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get member groups: %w", err)
		}

		seen := make(map[uuid.UUID]struct{})
		var all []uuid.UUID
		for _, gid := range managed {
			if _, ok := seen[gid]; !ok {
				seen[gid] = struct{}{}
				all = append(all, gid)
			}
		}
		for _, gid := range member {
			if _, ok := seen[gid]; !ok {
				seen[gid] = struct{}{}
				all = append(all, gid)
			}
		}
		if len(all) == 0 {
			return nil, 0, models.ErrPermissionDenied
		}
		req.GroupIDs = all
	}

	// Handle mode filtering
	if req.Mode != nil {
		switch *req.Mode {
		case "created":
			req.CreatorID = &req.Actor.ID
		case "assigned":
			req.AssigneeID = &req.Actor.ID
		}
	}

	data, total, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get tickets. error: %w", err)
	}
	return data, total, nil
}

func (s *TicketService) autoAssign(ctx context.Context, dto *models.TicketDTO) error {
	group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	if group.DefaultAssigneeID != nil {
		dto.AssigneeID = group.DefaultAssigneeID
		return nil
	}

	count, err := s.groups.GetMemberCount(ctx, *dto.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get member count: %w", err)
	}
	if count == 1 {
		members, err := s.groups.GetMembers(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
		if err != nil {
			return fmt.Errorf("failed to get members: %w", err)
		}
		if len(members) > 0 {
			dto.AssigneeID = &members[0].ID
		}
	}
	return nil
}

func (s *TicketService) CheckAccess(ctx context.Context, ticketID, userID uuid.UUID, action string, realm ...string) error {
	realmStr := ""
	if len(realm) > 0 {
		realmStr = realm[0]
	}

	ok, err := s.policies.Enforce(userID.String(), realmStr, string(access.ResourceTicket), action)
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
	case string(access.Write), string(access.Delete):
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

func (s *TicketService) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	if err := s.CheckAccess(ctx, req.ID, req.Actor.ID, string(access.Read), req.RealmID); err != nil {
		return nil, err
	}

	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	subtasks, err := s.subtasks.GetByTicketID(ctx, data.ID, req.Actor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	data.Subtasks = subtasks

	attachments, err := s.attachments.GetByEntity(ctx, string(access.ResourceTicket), data.ID, req.Actor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	data.Attachments = attachments

	return data, nil
}

func (s *TicketService) Create(ctx context.Context, dto *models.TicketDTO) error {
	realmStr := ""
	if dto.RealmID != nil {
		realmStr = dto.RealmID.String()
	}

	ok, err := s.policies.Enforce(dto.Actor.ID.String(), realmStr, string(access.ResourceTicket), string(access.Write))
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}
	if !ok {
		if dto.GroupID == nil {
			managed, err := s.groups.GetManagedGroups(ctx, dto.Actor.ID)
			if err != nil {
				return fmt.Errorf("failed to get managed groups: %w", err)
			}
			if len(managed) == 0 {
				return models.ErrPermissionDenied
			}
			if len(managed) > 1 {
				return fmt.Errorf("group is required: user manages %d groups", len(managed))
			}
			dto.GroupID = &managed[0]
		} else {
			managed, err := s.groups.GetManagedGroups(ctx, dto.Actor.ID)
			if err != nil {
				return fmt.Errorf("failed to check managed groups: %w", err)
			}
			isManager := false
			for _, gid := range managed {
				if gid == *dto.GroupID {
					isManager = true
					break
				}
			}
			if !isManager {
				return models.ErrPermissionDenied
			}
		}
	}

	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if dto.AssigneeID == nil && dto.GroupID != nil {
		if err := s.autoAssign(ctx, dto); err != nil {
			return fmt.Errorf("auto-assign: %w", err)
		}
	}

	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
		}

		log := &models.ActivityLogDTO{
			Action:        "created",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      *dto.ID,
			Entity:        dto.Title,
		}
		if err := log.SetNewValues(map[string]string{"title": dto.Title}); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := s.notifications.TicketCreated(ctx, dto); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func (s *TicketService) Update(ctx context.Context, dto *models.TicketDTO) error {
	realmStr := ""
	if dto.RealmID != nil {
		realmStr = dto.RealmID.String()
	}

	if err := s.CheckAccess(ctx, *dto.ID, dto.Actor.ID, string(access.Write), realmStr); err != nil {
		return err
	}

	var changes []*models.FieldChange
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if err != nil {
			return err
		}

		changes = dto.GetChanges(oldTicket)

		if err := s.repo.Update(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to update ticket. error: %w", err)
		}

		if len(changes) > 0 {
			oldMap := make(map[string]string, len(changes))
			newMap := make(map[string]string, len(changes))
			for _, c := range changes {
				oldMap[string(c.Tag)] = c.OldVal
				newMap[string(c.Tag)] = c.NewVal
			}

			log := &models.ActivityLogDTO{
				Action:        "updated",
				ChangedBy:     dto.Actor.ID,
				ChangedByName: dto.Actor.Name,
				EntityType:    string(access.ResourceTicket),
				EntityID:      *dto.ID,
				Entity:        oldTicket.Title,
			}
			if err := log.SetOldValues(oldMap); err != nil {
				return fmt.Errorf("set old values: %w", err)
			}
			if err := log.SetNewValues(newMap); err != nil {
				return fmt.Errorf("set new values: %w", err)
			}
			if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
				return fmt.Errorf("store logs: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(changes) > 0 {
		if err := s.notifications.TicketUpdated(ctx, *dto.ID, dto.Actor.ID, changes); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
	}
	return nil
}

func (s *TicketService) Delete(ctx context.Context, dto *models.DeleteTicketDTO) error {
	if err := s.CheckAccess(ctx, dto.ID, dto.Actor.ID, string(access.Write), dto.RealmID); err != nil {
		return err
	}

	var ticket *models.Ticket
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		var loadErr error
		ticket, loadErr = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if loadErr != nil {
			return fmt.Errorf("failed to load ticket: %w", loadErr)
		}

		snapshot := map[string]interface{}{
			"title":    ticket.Title,
			"status":   ticket.Status,
			"priority": ticket.Priority,
		}
		if ticket.Assignee != nil {
			snapshot["assignee"] = ticket.Assignee.ID.String()
		}
		log := &models.ActivityLogDTO{
			Action:        "deleted",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      dto.ID,
			Entity:        ticket.Title,
		}
		if err := log.SetOldValues(snapshot); err != nil {
			return fmt.Errorf("set old values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		if err := s.repo.Delete(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to delete ticket. error: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := s.notifications.TicketDeleted(ctx, ticket); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
