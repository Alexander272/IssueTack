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

type SubtaskService struct {
	repo         repository.Subtasks
	logs         ActivityLog
	ticketAccess TicketAccessChecker
}

func NewSubtaskService(repo repository.Subtasks, logs ActivityLog, ticketAccess TicketAccessChecker) *SubtaskService {
	return &SubtaskService{
		repo:         repo,
		logs:         logs,
		ticketAccess: ticketAccess,
	}
}

func (s *SubtaskService) SetTicketAccess(checker TicketAccessChecker) {
	s.ticketAccess = checker
}

type Subtasks interface {
	GetByTicketID(ctx context.Context, ticketID, actorID uuid.UUID) ([]*models.Subtask, error)
	GetByID(ctx context.Context, req *models.GetSubtaskDTO, actorID uuid.UUID) (*models.Subtask, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error
	Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error
}

func (s *SubtaskService) GetByTicketID(ctx context.Context, ticketID, actorID uuid.UUID) ([]*models.Subtask, error) {
	if s.ticketAccess == nil {
		return nil, models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, ticketID, actorID, string(access.Read)); err != nil {
		return nil, err
	}
	data, err := s.repo.GetByTicketID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	return data, nil
}

func (s *SubtaskService) GetByID(ctx context.Context, req *models.GetSubtaskDTO, actorID uuid.UUID) (*models.Subtask, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return nil, models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, data.TicketID, actorID, string(access.Read)); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *SubtaskService) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, dto.TicketID, dto.Actor.ID, string(access.Write)); err != nil {
		return err
	}
	if err := s.repo.Create(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create subtask: %w", err)
	}

	log := &models.ActivityLogDTO{
		Action:        "created",
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		EntityType:    "subtask",
		EntityID:      dto.ID,
		Entity:        dto.Title,
		ParentID:      &dto.TicketID,
	}
	if err := log.SetNewValues(map[string]string{"title": dto.Title}); err != nil {
		return fmt.Errorf("set new values: %w", err)
	}
	if err := s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log}); err != nil {
		return fmt.Errorf("store log: %w", err)
	}

	return nil
}

func (s *SubtaskService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if len(dto) > 0 {
		if err := s.ticketAccess.CheckAccess(ctx, dto[0].TicketID, dto[0].Actor.ID, string(access.Write)); err != nil {
			return err
		}
	}
	if err := s.repo.CreateSeveral(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create subtasks: %w", err)
	}

	logs := make([]*models.ActivityLogDTO, len(dto))
	for i, v := range dto {
		log := &models.ActivityLogDTO{
			Action:        "created",
			ChangedBy:     v.Actor.ID,
			ChangedByName: v.Actor.Name,
			EntityType:    "subtask",
			EntityID:      v.ID,
			Entity:        v.Title,
			ParentID:      &v.TicketID,
		}
		if err := log.SetNewValues(map[string]string{"title": v.Title}); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		logs[i] = log
	}
	if err := s.logs.Create(ctx, tx, logs); err != nil {
		return fmt.Errorf("store logs: %w", err)
	}

	return nil
}

func (s *SubtaskService) Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO) error {
	old, err := s.repo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, old.TicketID, dto.Actor.ID, string(access.Write)); err != nil {
		return err
	}

	changes := dto.GetChanges(old)

	if err := s.repo.Update(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to update subtask: %w", err)
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
			EntityType:    "subtask",
			EntityID:      dto.ID,
			Entity:        dto.Title,
			ParentID:      &dto.TicketID,
		}
		if err := log.SetOldValues(oldMap); err != nil {
			return fmt.Errorf("set old values: %w", err)
		}
		if err := log.SetNewValues(newMap); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}
	}

	return nil
}

func (s *SubtaskService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error {
	old, err := s.repo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, old.TicketID, dto.Actor.ID, string(access.Write)); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}

	snapshot := map[string]interface{}{
		"title":    old.Title,
		"status":   old.Status,
		"priority": old.Priority,
	}
	log := &models.ActivityLogDTO{
		Action:        "deleted",
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		EntityType:    "subtask",
		EntityID:      dto.ID,
		Entity:        old.Title,
		ParentID:      &old.TicketID,
	}
	if err := log.SetOldValues(snapshot); err != nil {
		return fmt.Errorf("set old values: %w", err)
	}
	if err := s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log}); err != nil {
		return fmt.Errorf("store log: %w", err)
	}

	return nil
}
