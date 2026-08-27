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

// SubtaskService — сервис работы с подзадачами тикетов.
type SubtaskService struct {
	repo         repository.Subtasks
	logs         ActivityLog
	ticketAccess TicketAccessChecker
}

// NewSubtaskService создаёт SubtaskService.
func NewSubtaskService(repo repository.Subtasks, logs ActivityLog, ticketAccess TicketAccessChecker) *SubtaskService {
	return &SubtaskService{
		repo:         repo,
		logs:         logs,
		ticketAccess: ticketAccess,
	}
}

// Subtasks — интерфейс работы с подзадачами.
type Subtasks interface {
	// GetByTicketID возвращает подзадачи тикета.
	GetByTicketID(ctx context.Context, ticketID, actorID uuid.UUID, realm string) ([]*models.Subtask, error)
	// GetByID возвращает подзадачу по идентификатору.
	GetByID(ctx context.Context, req *models.GetSubtaskDTO, actorID uuid.UUID, realm string) (*models.Subtask, error)
	// GetUnresolvedCount возвращает количество нерешённых подзадач тикета.
	GetUnresolvedCount(ctx context.Context, ticketID uuid.UUID) (int, error)
	// Create создаёт подзадачу.
	Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error
	// CreateSeveral создаёт несколько подзадач.
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO, realm string) error
	// Update обновляет подзадачу.
	Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error
	// Delete удаляет подзадачу.
	Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO, realm string) error
}

// GetByTicketID возвращает подзадачи тикета с проверкой доступа на чтение.
func (s *SubtaskService) GetByTicketID(ctx context.Context, ticketID, actorID uuid.UUID, realm string) ([]*models.Subtask, error) {
	if s.ticketAccess == nil {
		return nil, models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: ticketID, UserID: actorID, Action: string(access.Read), Realm: realm}); err != nil {
		return nil, err
	}
	data, err := s.repo.GetByTicketID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}
	return data, nil
}

// GetByID возвращает подзадачу по идентификатору с проверкой доступа на чтение.
func (s *SubtaskService) GetByID(ctx context.Context, req *models.GetSubtaskDTO, actorID uuid.UUID, realm string) (*models.Subtask, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return nil, models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: data.TicketID, UserID: actorID, Action: string(access.Read), Realm: realm}); err != nil {
		return nil, err
	}
	return data, nil
}

// GetUnresolvedCount возвращает количество подзадач тикета в нерешённом статусе.
func (s *SubtaskService) GetUnresolvedCount(ctx context.Context, ticketID uuid.UUID) (int, error) {
	data, err := s.repo.GetByTicketID(ctx, ticketID)
	if err != nil {
		return 0, fmt.Errorf("failed to get subtasks: %w", err)
	}
	count := 0
	for _, st := range data {
		switch st.Status {
		case models.StatusResolved, models.StatusClosed, models.StatusCancelled:
		default:
			count++
		}
	}
	return count, nil
}

// Create создаёт подзадачу с проверкой work-доступа и записью в журнал активности.
func (s *SubtaskService) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: dto.TicketID, UserID: dto.Actor.ID, Realm: realm}); err != nil {
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

// CreateSeveral создаёт несколько подзадач с проверкой work-доступа и записью в журнал активности.
func (s *SubtaskService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO, realm string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if len(dto) > 0 {
		if err := s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: dto[0].TicketID, UserID: dto[0].Actor.ID, Realm: realm}); err != nil {
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

// Update обновляет подзадачу с проверкой work-доступа и фиксацией изменений в журнале активности.
func (s *SubtaskService) Update(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error {
	old, err := s.repo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: old.TicketID, UserID: dto.Actor.ID, Realm: realm}); err != nil {
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

// Delete удаляет подзадачу с проверкой work-доступа и записью в журнал активности.
func (s *SubtaskService) Delete(ctx context.Context, tx postgres.Tx, dto *models.DelSubtaskDTO, realm string) error {
	old, err := s.repo.GetByID(ctx, &models.GetSubtaskDTO{ID: dto.ID})
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if err := s.ticketAccess.CheckWorkAccess(ctx, &models.AccessCheckDTO{TicketID: old.TicketID, UserID: dto.Actor.ID, Realm: realm}); err != nil {
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
