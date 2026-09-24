package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// ChecklistService — сервис работы с шаблонами чек-листов.
//
// Модель владения: шаблон имеет автора (created_by). Пользователь с правом
// checklist:read/write видит и меняет любые шаблоны реалма; остальные (например,
// создатель подзадач) — только свои. Доступ проверяется на уровне сервиса, поэтому
// роуты не гейтятся Casbin-мидлваром на checklist:*.
type ChecklistService struct {
	repo     repository.Checklists
	subtasks Subtasks
	policies AccessPolicies
}

// NewChecklistService создаёт ChecklistService.
func NewChecklistService(repo repository.Checklists, subtasks Subtasks, policies AccessPolicies) *ChecklistService {
	return &ChecklistService{
		repo:     repo,
		subtasks: subtasks,
		policies: policies,
	}
}

// Checklists — интерфейс работы с шаблонами чек-листов.
type Checklists interface {
	// Get возвращает список шаблонов чек-листов. Без права checklist:read — только свои.
	Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error)
	// GetByID возвращает шаблон чек-листа вместе с его пунктами.
	GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO, actorID uuid.UUID, realm string) (*models.ChecklistTemplate, error)
	// Create создаёт шаблон чек-листа. Название должно быть уникальным в видимой области.
	Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error
	// Update обновляет шаблон чек-листа (владелец или checklist:write).
	Update(ctx context.Context, dto *models.ChecklistTemplateDTO, actorID uuid.UUID, realm string) error
	// Delete удаляет шаблон чек-листа (владелец или checklist:delete).
	Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO, actorID uuid.UUID, realm string) error
	// SetItems заменяет набор пунктов шаблона чек-листа (владелец или checklist:write).
	SetItems(ctx context.Context, tx postgres.Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO, actorID uuid.UUID, realm string) error
	// GetItems возвращает пункты шаблона чек-листа.
	GetItems(ctx context.Context, templateID uuid.UUID, actorID uuid.UUID, realm string) ([]*models.ChecklistTemplateItem, error)
	// ApplyTemplate создаёт подзадачи тикета по шаблону чек-листа.
	ApplyTemplate(ctx context.Context, tx postgres.Tx, dto *models.ApplyTemplateDTO) error
}

// hasChecklistPerm проверяет, есть ли у пользователя право action на ресурс чек-листов в реалме.
func (s *ChecklistService) hasChecklistPerm(ctx context.Context, userID uuid.UUID, realm string, action string) (bool, error) {
	if s.policies == nil {
		return false, nil
	}
	return s.policies.Enforce(userID.String(), realm, string(access.ResourceChecklist), action)
}

// canAccessTemplate — доступ к конкретному шаблону: право checklist:read или автор.
func (s *ChecklistService) canAccessTemplate(ctx context.Context, tpl *models.ChecklistTemplate, actorID uuid.UUID, realm string) (bool, error) {
	if tpl.CreatedBy != nil && *tpl.CreatedBy == actorID {
		return true, nil
	}
	if realm == "" {
		return s.hasChecklistPerm(ctx, actorID, tpl.RealmID.String(), string(access.Read))
	}
	return s.hasChecklistPerm(ctx, actorID, realm, string(access.Read))
}

// canManageTemplate — право менять шаблон: автор или checklist:write.
func (s *ChecklistService) canManageTemplate(ctx context.Context, tpl *models.ChecklistTemplate, actorID uuid.UUID, realm string, action string) (bool, error) {
	if tpl.CreatedBy != nil && *tpl.CreatedBy == actorID {
		return true, nil
	}
	if realm == "" {
		realm = tpl.RealmID.String()
	}
	return s.hasChecklistPerm(ctx, actorID, realm, action)
}

// Get возвращает список шаблонов чек-листов. Пользователи без права checklist:read
// видят только свои шаблоны (фильтр по created_by).
func (s *ChecklistService) Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error) {
	if req.Actor != nil && req.OwnerID == nil {
		hasRead, err := s.hasChecklistPerm(ctx, req.Actor.ID, req.RealmID.String(), string(access.Read))
		if err != nil {
			return nil, fmt.Errorf("failed to check checklist read access: %w", err)
		}
		if !hasRead {
			req.OwnerID = &req.Actor.ID
		}
	}

	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist templates: %w", err)
	}
	return data, nil
}

// GetByID возвращает шаблон чек-листа вместе с его пунктами.
func (s *ChecklistService) GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO, actorID uuid.UUID, realm string) (*models.ChecklistTemplate, error) {
	template, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist template: %w", err)
	}

	ok, err := s.canAccessTemplate(ctx, template, actorID, realm)
	if err != nil {
		return nil, fmt.Errorf("failed to check template access: %w", err)
	}
	if !ok {
		return nil, models.ErrPermissionDenied
	}

	items, err := s.repo.GetItems(ctx, template.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template items: %w", err)
	}
	template.Items = items

	return template, nil
}

// Create создаёт шаблон чек-листа. Название должно быть уникальным в видимой
// пользователем области (свои шаблоны / все шаблоны реалма при checklist:read).
func (s *ChecklistService) Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	if dto.Actor == nil {
		return models.ErrPermissionDenied
	}
	if strings.TrimSpace(dto.Title) == "" {
		return models.ErrInvalidInput
	}

	var ownerID *uuid.UUID
	hasWrite, err := s.hasChecklistPerm(ctx, dto.Actor.ID, dto.RealmID.String(), string(access.Write))
	if err != nil {
		return fmt.Errorf("failed to check checklist write access: %w", err)
	}
	if !hasWrite {
		ownerID = &dto.Actor.ID
	}

	exists, err := s.repo.ExistsByTitle(ctx, dto.RealmID, strings.TrimSpace(dto.Title), ownerID)
	if err != nil {
		return fmt.Errorf("failed to check template title: %w", err)
	}
	if exists {
		return models.ErrTemplateNameExists
	}

	dto.CreatedBy = &dto.Actor.ID
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create checklist template: %w", err)
	}
	return nil
}

// Update обновляет шаблон чек-листа. Доступно владельцу или обладателю checklist:write.
func (s *ChecklistService) Update(ctx context.Context, dto *models.ChecklistTemplateDTO, actorID uuid.UUID, realm string) error {
	if strings.TrimSpace(dto.Title) == "" {
		return models.ErrInvalidInput
	}

	tpl, err := s.repo.GetByID(ctx, &models.GetChecklistTemplateDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get checklist template: %w", err)
	}
	ok, err := s.canManageTemplate(ctx, tpl, actorID, realm, string(access.Write))
	if err != nil {
		return fmt.Errorf("failed to check template edit access: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}

	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update checklist template: %w", err)
	}
	return nil
}

// Delete удаляет шаблон чек-листа. Доступно владельцу или обладателю checklist:delete.
func (s *ChecklistService) Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO, actorID uuid.UUID, realm string) error {
	tpl, err := s.repo.GetByID(ctx, &models.GetChecklistTemplateDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get checklist template: %w", err)
	}
	ok, err := s.canManageTemplate(ctx, tpl, actorID, realm, string(access.Delete))
	if err != nil {
		return fmt.Errorf("failed to check template delete access: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}

	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete checklist template: %w", err)
	}
	return nil
}

// SetItems заменяет набор пунктов шаблона чек-листа. Доступно владельцу или обладателю checklist:write.
func (s *ChecklistService) SetItems(ctx context.Context, tx postgres.Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO, actorID uuid.UUID, realm string) error {
	tpl, err := s.repo.GetByID(ctx, &models.GetChecklistTemplateDTO{ID: templateID})
	if err != nil {
		return fmt.Errorf("failed to get checklist template: %w", err)
	}
	ok, err := s.canManageTemplate(ctx, tpl, actorID, realm, string(access.Write))
	if err != nil {
		return fmt.Errorf("failed to check template edit access: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}

	if err := s.repo.SetItems(ctx, tx, templateID, items); err != nil {
		return fmt.Errorf("failed to set template items: %w", err)
	}
	return nil
}

// GetItems возвращает пункты шаблона чек-листа (с проверкой доступа к шаблону).
func (s *ChecklistService) GetItems(ctx context.Context, templateID uuid.UUID, actorID uuid.UUID, realm string) ([]*models.ChecklistTemplateItem, error) {
	tpl, err := s.repo.GetByID(ctx, &models.GetChecklistTemplateDTO{ID: templateID})
	if err != nil {
		return nil, fmt.Errorf("failed to get checklist template: %w", err)
	}
	ok, err := s.canAccessTemplate(ctx, tpl, actorID, realm)
	if err != nil {
		return nil, fmt.Errorf("failed to check template access: %w", err)
	}
	if !ok {
		return nil, models.ErrPermissionDenied
	}

	data, err := s.repo.GetItems(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template items: %w", err)
	}
	return data, nil
}

// ApplyTemplate создаёт подзадачи тикета по шаблону чек-листа.
func (s *ChecklistService) ApplyTemplate(ctx context.Context, tx postgres.Tx, dto *models.ApplyTemplateDTO) error {
	tpl, err := s.repo.GetByID(ctx, &models.GetChecklistTemplateDTO{ID: dto.TemplateID})
	if err != nil {
		return fmt.Errorf("failed to get checklist template: %w", err)
	}
	ok, err := s.canAccessTemplate(ctx, tpl, dto.Actor.ID, tpl.RealmID.String())
	if err != nil {
		return fmt.Errorf("failed to check template access: %w", err)
	}
	if !ok {
		return models.ErrPermissionDenied
	}

	items, err := s.repo.GetItems(ctx, dto.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get template items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	subtaskDTOs := make([]*models.SubtaskDTO, len(items))
	for i, item := range items {
		subtaskDTOs[i] = &models.SubtaskDTO{
			TicketID:    dto.TicketID,
			Title:       item.Title,
			Description: item.Description,
			Status:      models.StatusOpen,
			SortOrder:   item.SortOrder,
			Actor:       dto.Actor,
		}
	}

	if err := s.subtasks.CreateSeveral(ctx, tx, subtaskDTOs, ""); err != nil {
		return fmt.Errorf("failed to create subtasks from template: %w", err)
	}

	return nil
}
