package services

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// GetRawByID возвращает подзадачу по идентификатору без проверки доступа —
	// для внутренних сервисов, которым нужен только сам агрегат (например,
	// разрешение родительского тикета вложений).
	GetRawByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error)
	// GetUnresolvedCount возвращает количество нерешённых подзадач тикета.
	GetUnresolvedCount(ctx context.Context, ticketID uuid.UUID) (int, error)
	// Create создаёт подзадачу.
	Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error
	// CreateSeveral создаёт несколько подзадач.
	CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO, realm string) error
	// CreateManyOnCreate создаёт подзадачи в момент создания заявки внутри той же
	// транзакции. Проверка CanCreateSubtask не выполняется: авторизация неявна —
	// создатель новой заявки всегда имеет право создавать в ней подзадачи, а сама
	// заявка ещё не закоммичена и невидима пулом (CanCreateSubtask читает тикет
	// через отдельное соединение). Вызывающий должен заполнить TicketID, Actor,
	// Status, Priority и SortOrder каждого DTO.
	CreateManyOnCreate(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error
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

// GetRawByID возвращает подзадачу по идентификатору без проверки доступа.
func (s *SubtaskService) GetRawByID(ctx context.Context, req *models.GetSubtaskDTO) (*models.Subtask, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	return data, nil
}

// GetByID возвращает подзадачу по идентификатору с проверкой права чтения
// родительского тикета.
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

// Create создаёт подзадачу с проверкой права создания (CanCreateSubtask: создатель/
// исполнитель тикета или «управление» тикетом) и записью в журнал активности.
func (s *SubtaskService) Create(ctx context.Context, tx postgres.Tx, dto *models.SubtaskDTO, realm string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	canCreate, err := s.ticketAccess.CanCreateSubtask(ctx, dto.Actor.ID, dto.TicketID, realm)
	if err != nil {
		return fmt.Errorf("failed to check subtask create access: %w", err)
	}
	if !canCreate {
		return models.ErrPermissionDenied
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

// CreateSeveral создаёт несколько подзадач с проверкой права создания и записью в журнал активности.
func (s *SubtaskService) CreateSeveral(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO, realm string) error {
	if s.ticketAccess == nil {
		return models.ErrPermissionDenied
	}
	if len(dto) > 0 {
		canCreate, err := s.ticketAccess.CanCreateSubtask(ctx, dto[0].Actor.ID, dto[0].TicketID, realm)
		if err != nil {
			return fmt.Errorf("failed to check subtask create access: %w", err)
		}
		if !canCreate {
			return models.ErrPermissionDenied
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

// CreateManyOnCreate создаёт подзадачи в момент создания заявки внутри той же транзакции
// (см. интерфейс Subtasks). В отличие от CreateSeveral не выполняет проверку
// CanCreateSubtask: заявка ещё не закоммичена (невидима пулу), а создатель всегда
// авторизован. Пропущенные Status/Priority/SortOrder заполняются значениями по умолчанию.
func (s *SubtaskService) CreateManyOnCreate(ctx context.Context, tx postgres.Tx, dto []*models.SubtaskDTO) error {
	if len(dto) == 0 {
		return nil
	}
	for i, v := range dto {
		if v == nil || v.Actor == nil {
			return errors.New("subtask actor is required")
		}
		if v.Status == "" {
			v.Status = models.StatusOpen
		}
		if v.Priority == "" {
			v.Priority = models.PriorityMedium
		}
		if v.SortOrder == 0 && i > 0 {
			v.SortOrder = i
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

// Update обновляет подзадачу. Требует:
//   - work-доступ к тикету (CheckWorkAccess) для любых изменений;
//   - права на правку содержимого (CanEditSubtask: автор подзадачи или менеджер группы /
//     realm supervisor) для всех полей, кроме status. Смену статуса может выполнить любой
//     обладатель work-доступа.
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

	for _, f := range []string{"title", "description", "priority", "assigneeId", "dueDate", "sortOrder"} {
		if !dto.HasField(f) {
			continue
		}
		canEdit, err := s.ticketAccess.CanEditSubtask(ctx, dto.Actor.ID, old)
		if err != nil {
			return fmt.Errorf("failed to check subtask edit access: %w", err)
		}
		if !canEdit {
			return models.ErrPermissionDenied
		}
		break
	}

	// Проставление closed_at по аналогии с тикетами: завершающие статусы
	// (resolved/closed) фиксируют время, возврат в активный статус сбрасывает его.
	if dto.HasField("status") && dto.Status != old.Status {
		now := time.Now()
		switch dto.Status {
		case models.StatusResolved, models.StatusClosed:
			if old.ClosedAt == nil {
				dto.ClosedAt = &now
				dto.MarkProvided("closedAt")
			}
		default:
			if old.ClosedAt != nil {
				dto.ClosedAt = nil
				dto.MarkProvided("closedAt")
			}
		}
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

// Delete удаляет подзадачу. Требует «рабочего» доступа (CheckWorkAccess — блокирует
// замороженные заявки) и права на удаление из агрегата тикета: менеджер группы
// (по атрибутам) или обладатель Casbin ticket:delete.
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
	if err := s.ticketAccess.CheckAccess(ctx, &models.AccessCheckDTO{TicketID: old.TicketID, UserID: dto.Actor.ID, Action: string(access.Delete), Realm: realm}); err != nil {
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
