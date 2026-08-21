package services

import (
	"context"
	"fmt"
	"time"

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

func NewTicketService(deps *TicketDeps) *TicketService {
	return &TicketService{
		repo:          deps.Repo,
		tx:            deps.TxManager,
		logs:          deps.Logs,
		subtasks:      deps.Subtasks,
		attachments:   deps.Attachments,
		notifications: deps.Notifications,
		groups:        deps.Groups,
		policies:      deps.Policies,
	}
}

type TicketDeps struct {
	Repo          repository.Tickets
	TxManager     TransactionManager
	Logs          ActivityLog
	Subtasks      Subtasks
	Attachments   Attachments
	Notifications Notifications
	Groups        Groups
	Policies      AccessPolices
}

type Tickets interface {
	Get(ctx context.Context, req *models.TicketFilter) ([]*models.Ticket, int, error)
	GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error)
	Create(ctx context.Context, dto *models.TicketDTO) error
	Update(ctx context.Context, dto *models.TicketDTO) error
	Delete(ctx context.Context, dto *models.DeleteTicketDTO) error
	AutoCloseResolved(ctx context.Context, delay time.Duration) (int64, error)
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
		managed, err := s.groups.GetManagedGroups(ctx, req.Actor.ID, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get managed groups: %w", err)
		}
		member, err := s.groups.GetMemberGroups(ctx, req.Actor.ID, nil)
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

		if len(all) > 0 {
			req.GroupIDs = all
		}

		req.IncludeUngroupedAssignedTo = &req.Actor.ID
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

func (s *TicketService) GetByID(ctx context.Context, req *models.GetTicketByIdDTO) (*models.Ticket, error) {
	if err := s.CheckAccess(ctx, req.ID, req.Actor.ID, string(access.Read), req.RealmID); err != nil {
		return nil, err
	}

	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket by id. error: %w", err)
	}

	subtasks, err := s.subtasks.GetByTicketID(ctx, data.ID, req.Actor.ID, req.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	data.Subtasks = subtasks

	attachments, err := s.attachments.GetByEntity(ctx, string(access.ResourceTicket), data.ID, req.Actor.ID, req.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	data.Attachments = attachments

	realmStr := ""
	if req.RealmID != "" {
		realmStr = req.RealmID
	}
	flags, err := s.GetAccessFlags(ctx, data, req.Actor.ID, realmStr)
	if err == nil {
		data.Access = flags
	}

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
		return models.ErrPermissionDenied
	}

	if dto.GroupID == nil {
		return fmt.Errorf("group is required")
	}

	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if dto.AssigneeID == nil {
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

	assignedOnly := false
	ownerOnly := false
	if err := s.CheckAccess(ctx, *dto.ID, dto.Actor.ID, string(access.Write), realmStr); err != nil {
		if workErr := s.CheckWorkAccess(ctx, *dto.ID, dto.Actor.ID, realmStr); workErr != nil {
			old, loadErr := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
			if loadErr != nil {
				return fmt.Errorf("failed to load ticket for access check: %w", loadErr)
			}
			if !s.isOwner(old, dto.Actor.ID) {
				return err
			}
			ownerOnly = true
		}
		assignedOnly = true
	}

	var changes []*models.FieldChange
	err := s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		oldTicket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
		if err != nil {
			return err
		}

		if ownerOnly && !s.ownerTransitionAllowed(oldTicket, dto) {
			return models.ErrPermissionDenied
		}

		if dto.HasField("status") && dto.Status != oldTicket.Status &&
			(dto.Status == models.StatusClosed || dto.Status == models.StatusCancelled) {
			ok, err := s.isCreatorOrManager(ctx, oldTicket, dto.Actor.ID)
			if err != nil {
				return fmt.Errorf("failed to check close access: %w", err)
			}
			if !ok && s.isOwner(oldTicket, dto.Actor.ID) {
				ok = true
			}
			if !ok {
				return models.ErrPermissionDenied
			}
		}

		if dto.HasField("status") && dto.Status != oldTicket.Status {
			switch dto.Status {
			case models.StatusResolved:
				n, err := s.subtasks.GetUnresolvedCount(ctx, *dto.ID)
				if err != nil {
					return fmt.Errorf("failed to check subtasks: %w", err)
				}
				if n > 0 {
					return models.ErrSubtasksNotResolved
				}
			case models.StatusClosed:
				if oldTicket.Status != models.StatusResolved {
					return models.ErrCloseRequiresResolved
				}
			}
		}

		if dto.HasField("status") {
			now := time.Now()
			switch dto.Status {
			case models.StatusResolved:
				if oldTicket.ResolvedAt == nil {
					dto.ResolvedAt = &now
					dto.MarkProvided("resolvedAt")
				}
			case models.StatusClosed, models.StatusCancelled:
				if oldTicket.ClosedAt == nil {
					dto.ClosedAt = &now
					dto.MarkProvided("closedAt")
				}
			default:
				if oldTicket.ClosedAt != nil {
					dto.ClosedAt = nil
					dto.MarkProvided("closedAt")
				}
				if oldTicket.ResolvedAt != nil {
					dto.ResolvedAt = nil
					dto.MarkProvided("resolvedAt")
				}
			}
		}

		changes = dto.GetChanges(oldTicket)

		if assignedOnly {
			for _, change := range changes {
				if change.Tag != models.ActionStatusChanged && change.Tag != models.ActionClosed {
					return models.ErrPermissionDenied
				}
			}
		}

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
	if err := s.CheckAccess(ctx, dto.ID, dto.Actor.ID, string(access.Delete), dto.RealmID); err != nil {
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

func (s *TicketService) isCreatorOrManager(ctx context.Context, ticket *models.Ticket, actorID uuid.UUID) (bool, error) {
	if ticket.Creator.ID == actorID {
		return true, nil
	}
	if ticket.Group == nil {
		return false, nil
	}

	managed, err := s.groups.GetManagedGroups(ctx, actorID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get managed groups: %w", err)
	}
	for _, gid := range managed {
		if gid == ticket.Group.ID {
			return true, nil
		}
	}
	return false, nil
}

func (s *TicketService) isOwner(ticket *models.Ticket, actorID uuid.UUID) bool {
	return ticket.Owner != nil && ticket.Owner.ID == actorID
}

func (s *TicketService) ownerTransitionAllowed(ticket *models.Ticket, dto *models.TicketDTO) bool {
	if !dto.HasField("status") || dto.Status == ticket.Status {
		return false
	}
	switch dto.Status {
	case models.StatusCancelled:
		return ticket.Status != models.StatusResolved &&
			ticket.Status != models.StatusClosed &&
			ticket.Status != models.StatusCancelled
	case models.StatusClosed:
		return ticket.Status == models.StatusResolved
	case models.StatusInProgress:
		return ticket.Status == models.StatusResolved
	default:
		return false
	}
}

func (s *TicketService) AutoCloseResolved(ctx context.Context, delay time.Duration) (int64, error) {
	if delay <= 0 {
		return 0, nil
	}

	cutoff := time.Now().Add(-delay)
	return s.repo.CloseResolved(ctx, cutoff)
}

var activeStatuses = []models.TicketStatus{
	models.StatusOpen,
	models.StatusInProgress,
	models.StatusPending,
	models.StatusOnHold,
}

func isActive(s models.TicketStatus) bool {
	return s == models.StatusOpen || s == models.StatusInProgress ||
		s == models.StatusPending || s == models.StatusOnHold
}

func (s *TicketService) GetAccessFlags(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, realm string) (*models.AccessFlags, error) {
	flags := &models.AccessFlags{CanRead: true}

	writeErr := s.checkAccessForTicket(ctx, ticket, userID, string(access.Write), realm)
	flags.CanWrite = writeErr == nil

	deleteErr := s.checkAccessForTicket(ctx, ticket, userID, string(access.Delete), realm)
	flags.CanDelete = deleteErr == nil

	flags.CanWork = flags.CanWrite || (ticket.Assignee != nil && ticket.Assignee.ID == userID)

	flags.AllowedStatuses = s.computeAllowedStatuses(ctx, ticket, userID, realm)

	return flags, nil
}

func (s *TicketService) checkAccessForTicket(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, action string, realm string) error {
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

func (s *TicketService) computeAllowedStatuses(ctx context.Context, ticket *models.Ticket, userID uuid.UUID, realm string) []models.TicketStatus {
	hasWrite := s.checkAccessForTicket(ctx, ticket, userID, string(access.Write), realm) == nil
	isCreator := ticket.Creator.ID == userID

	isMgr := false
	if ticket.Group != nil {
		managed, err := s.groups.GetManagedGroups(ctx, userID, nil)
		if err == nil {
			for _, gid := range managed {
				if gid == ticket.Group.ID {
					isMgr = true
					break
				}
			}
		}
	}
	isCreatorOrMgr := isCreator || isMgr
	isOwner := ticket.Owner != nil && ticket.Owner.ID == userID
	hasWork := hasWrite || (ticket.Assignee != nil && ticket.Assignee.ID == userID)

	current := ticket.Status
	var allowed []models.TicketStatus

	add := func(statuses ...models.TicketStatus) {
		for _, st := range statuses {
			found := false
			for _, a := range allowed {
				if a == st {
					found = true
					break
				}
			}
			if !found {
				allowed = append(allowed, st)
			}
		}
	}

	switch {
	case current == models.StatusClosed || current == models.StatusCancelled:
		if hasWrite || hasWork {
			add(activeStatuses...)
		}
		if isCreatorOrMgr || isOwner {
			if current == models.StatusClosed {
				add(models.StatusCancelled)
			} else {
				add(models.StatusClosed)
			}
		}

	case current == models.StatusResolved:
		if isCreatorOrMgr || isOwner {
			add(models.StatusClosed)
		}
		if hasWrite || hasWork || isOwner {
			add(models.StatusInProgress)
		}
		if hasWrite || hasWork {
			for _, st := range activeStatuses {
				if st != models.StatusInProgress {
					add(st)
				}
			}
		}
		if isCreatorOrMgr {
			add(models.StatusCancelled)
		}

	default:
		if hasWrite || hasWork {
			for _, st := range activeStatuses {
				if st != current {
					add(st)
				}
			}
			n, err := s.subtasks.GetUnresolvedCount(ctx, ticket.ID)
			if err == nil && n == 0 {
				add(models.StatusResolved)
			}
		}
		if isCreatorOrMgr || isOwner {
			add(models.StatusCancelled)
		}
	}

	return allowed
}
