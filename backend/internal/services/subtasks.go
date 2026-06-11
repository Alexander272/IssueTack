package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type SubtaskService struct {
	repo repository.Subtasks
	logs ActivityLog
}

func NewSubtaskService(repo repository.Subtasks, logs ActivityLog) *SubtaskService {
	return &SubtaskService{
		repo: repo,
		logs: logs,
	}
}

type Subtasks interface {
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*models.Subtask, error)
	GetByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error)
	Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, actor models.Actor) error
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error
	Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, actor models.Actor) error
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error
}

func (s *SubtaskService) GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*models.Subtask, error) {
	data, err := s.repo.GetByTicketID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	return data, nil
}

func (s *SubtaskService) GetByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	return data, nil
}

func (s *SubtaskService) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, actor models.Actor) error {
	if err := s.repo.Create(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create subtask: %w", err)
	}

	log := &models.ActivityLogDTO{
		Action:        "created",
		ChangedBy:     actor.ID,
		ChangedByName: actor.Name,
		EntityType:    "subtask",
		EntityID:      dto.ID,
		Entity:        dto.Title,
	}
	if err := s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log}); err != nil {
		return fmt.Errorf("store log: %w", err)
	}

	return nil
}

func (s *SubtaskService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error {
	if err := s.repo.CreateSeveral(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to create subtasks: %w", err)
	}
	return nil
}

func (s *SubtaskService) Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, actor models.Actor) error {
	old, err := s.repo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}

	if err := s.repo.Update(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to update subtask: %w", err)
	}

	var logs []*models.ActivityLogDTO
	if dto.Title != old.Title {
		logs = append(logs, &models.ActivityLogDTO{
			Action: "title_changed", ChangedBy: actor.ID, ChangedByName: actor.Name,
			EntityType: "subtask", EntityID: dto.ID, Entity: dto.Title,
			OldValue: &old.Title, NewValue: &dto.Title,
		})
	}
	if dto.Status != old.Status {
		logs = append(logs, &models.ActivityLogDTO{
			Action: "status_changed", ChangedBy: actor.ID, ChangedByName: actor.Name,
			EntityType: "subtask", EntityID: dto.ID, Entity: dto.Title,
			OldValue: strPtr(string(old.Status)), NewValue: strPtr(string(dto.Status)),
		})
	}
	if dto.AssigneeID != nil && (old.Assignee == nil || *dto.AssigneeID != old.Assignee.ID) {
		logs = append(logs, &models.ActivityLogDTO{
			Action: "assigned", ChangedBy: actor.ID, ChangedByName: actor.Name,
			EntityType: "subtask", EntityID: dto.ID, Entity: dto.Title,
		})
	}

	if len(logs) > 0 {
		if err := s.logs.Create(ctx, tx, logs); err != nil {
			return fmt.Errorf("store log: %w", err)
		}
	}

	return nil
}

func (s *SubtaskService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO) error {
	if err := s.repo.Delete(ctx, tx, dto); err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}

	log := &models.ActivityLogDTO{
		Action:        "deleted",
		ChangedBy:     dto.Actor.ID,
		ChangedByName: dto.Actor.Name,
		EntityType:    "subtask",
		EntityID:      dto.ID,
	}
	if err := s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log}); err != nil {
		return fmt.Errorf("store log: %w", err)
	}

	return nil
}

func strPtr(s string) *string { return &s }
