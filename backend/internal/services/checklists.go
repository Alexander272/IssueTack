package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

type ChecklistService struct {
	repo     repository.Checklists
	subtasks Subtasks
}

func NewChecklistService(repo repository.Checklists, subtasks Subtasks) *ChecklistService {
	return &ChecklistService{
		repo:     repo,
		subtasks: subtasks,
	}
}

type Checklists interface {
	Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error)
	GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO) (*models.ChecklistTemplate, error)
	Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error
	Update(ctx context.Context, dto *models.ChecklistTemplateDTO) error
	Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO) error
	SetItems(ctx context.Context, tx postgres.Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO) error
	GetItems(ctx context.Context, templateID uuid.UUID) ([]*models.ChecklistTemplateItem, error)
	ApplyTemplate(ctx context.Context, tx postgres.Tx, ticketID uuid.UUID, templateID uuid.UUID, actor *models.Actor) error
}

func (s *ChecklistService) Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist templates: %w", err)
	}
	return data, nil
}

func (s *ChecklistService) GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO) (*models.ChecklistTemplate, error) {
	template, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist template: %w", err)
	}

	items, err := s.repo.GetItems(ctx, template.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template items: %w", err)
	}
	template.Items = items

	return template, nil
}

func (s *ChecklistService) Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create checklist template: %w", err)
	}
	return nil
}

func (s *ChecklistService) Update(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update checklist template: %w", err)
	}
	return nil
}

func (s *ChecklistService) Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO) error {
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete checklist template: %w", err)
	}
	return nil
}

func (s *ChecklistService) SetItems(ctx context.Context, tx postgres.Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO) error {
	if err := s.repo.SetItems(ctx, tx, templateID, items); err != nil {
		return fmt.Errorf("failed to set template items: %w", err)
	}
	return nil
}

func (s *ChecklistService) GetItems(ctx context.Context, templateID uuid.UUID) ([]*models.ChecklistTemplateItem, error) {
	data, err := s.repo.GetItems(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template items: %w", err)
	}
	return data, nil
}

func (s *ChecklistService) ApplyTemplate(ctx context.Context, tx postgres.Tx, ticketID uuid.UUID, templateID uuid.UUID, actor *models.Actor) error {
	items, err := s.repo.GetItems(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	subtaskDTOs := make([]*models.SubtaskDTO, len(items))
	for i, item := range items {
		subtaskDTOs[i] = &models.SubtaskDTO{
			TicketID:  ticketID,
			Title:     item.Title,
			Status:    models.StatusOpen,
			SortOrder: item.SortOrder,
			Actor:     actor,
		}
	}

	if err := s.subtasks.CreateSeveral(ctx, tx, subtaskDTOs); err != nil {
		return fmt.Errorf("failed to create subtasks from template: %w", err)
	}

	return nil
}
