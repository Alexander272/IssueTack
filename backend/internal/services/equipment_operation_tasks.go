package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// EquipmentOperationTaskService — правила автоматических задач после операций
// с оборудованием (выдача/возврат). Правила живут в БД и правятся из UI, а не в
// некоммитимом config.yaml; при отсутствии правил реалма остаётся старый
// fallback на реквизиты из конфигурации (см. EquipmentService.createERPTicket).
type EquipmentOperationTaskService struct {
	repo    repository.Equipment
	tm      TransactionManager
	tickets TicketCreator
}

func NewEquipmentOperationTaskService(repo repository.Equipment, tm TransactionManager) *EquipmentOperationTaskService {
	return &EquipmentOperationTaskService{repo: repo, tm: tm}
}

// SetTicketCreator подключает TicketService: задача создаётся обычным путём,
// со всеми проверками прав, журналом и уведомлениями.
func (s *EquipmentOperationTaskService) SetTicketCreator(tc TicketCreator) {
	s.tickets = tc
}

// OperationTaskRunner — узкий срез для EquipmentService: после коммита выдачи
// или возврата он спрашивает, есть ли правила для этой операции. Возвращаемое
// значение сообщает, что правила «властвуют» (список прочитан, пусть и пустым
// или с выключенными записями) и fallback на конфиг применять не нужно.
type OperationTaskRunner interface {
	RunForOperation(ctx context.Context, op models.EquipmentOperation, equipmentID, realmID uuid.UUID, actor *models.Actor) bool
}

var _ OperationTaskRunner = (*EquipmentOperationTaskService)(nil)

// RunForOperation запускает включённые правила операции. Ошибки не пробрасываются
// наверх: операция выдачи/возврата уже закоммичена, и её сбой не должен
// выглядеть как неудавшаяся. Как и у ERP-заявки, сбой уходит в bestEffortError.
func (s *EquipmentOperationTaskService) RunForOperation(ctx context.Context, op models.EquipmentOperation, equipmentID, realmID uuid.UUID, actor *models.Actor) bool {
	if s.repo == nil {
		return false
	}

	tasks, err := s.repo.GetOperationTasks(ctx, realmID, &op)
	if err != nil {
		bestEffortError("failed to load equipment operation tasks", err, map[string]string{
			"equipment_id": equipmentID.String(),
			"operation":    string(op),
		})
		// Правила есть источник истины: при сбое чтения не подменяем их конфигом.
		return true
	}
	if len(tasks) == 0 {
		// Ни одного правила — реалм живёт по старой конфигурации.
		return false
	}
	enabled := make([]*models.EquipmentOperationTask, 0, len(tasks))
	for _, task := range tasks {
		if task.Enabled && task.Action == models.EquipmentActionCreateTicket {
			enabled = append(enabled, task)
		}
	}
	if len(enabled) == 0 || s.tickets == nil || actor == nil {
		return true
	}

	// Единицу читаем заново: к моменту вызова операция уже закоммичена, и в
	// описании заявки должен быть новый держатель. Площадка берётся из единицы —
	// не из правила и не из заказчика.
	eq, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: equipmentID, RealmID: realmID})
	if err != nil || eq == nil {
		bestEffortError("failed to load equipment for operation task", err, map[string]string{
			"equipment_id": equipmentID.String(),
			"operation":    string(op),
		})
		return true
	}

	var siteID uuid.UUID
	if eq.SiteID != nil {
		siteID = *eq.SiteID
	}

	for _, task := range enabled {
		title, description := task.Title, task.Description
		if title == "" || description == "" {
			defaultTitle, defaultDescription := erpTicketContent(erpOperation(op), eq)
			if title == "" {
				title = defaultTitle
			}
			if description == "" {
				description = defaultDescription
			}
		}

		priority := models.Priority(task.Priority)
		if !priority.IsValid() {
			priority = models.PriorityMedium
		}
		dto := &models.TicketDTO{
			Title:       title,
			Description: description,
			Status:      models.StatusOpen,
			Priority:    priority,
			RealmID:     &eq.RealmID,
			SiteID:      siteID,
			CategoryID:  task.CategoryID,
			GroupID:     &task.GroupID,
			CreatorID:   actor.ID,
			// Заказчик и исполнитель — тот, кто провёл операцию: заявка «для себя».
			// Пустой AssigneeID ушёл бы в autoAssign и отдал бы её ответственному
			// группы, а не кладовщику.
			OwnerID:     &actor.ID,
			AssigneeID:  &actor.ID,
			EquipmentID: &eq.ID,
			Actor:       actor,
		}
		if err := s.tickets.Create(ctx, dto); err != nil {
			bestEffortError("failed to create equipment operation ticket", err, map[string]string{
				"equipment_id": equipmentID.String(),
				"operation":    string(op),
				"rule_id":      task.ID.String(),
			})
		}
	}
	return true
}

// GetOperationTasks — список правил реалма, опционально filtered по операции.
func (s *EquipmentOperationTaskService) GetOperationTasks(ctx context.Context, realmID uuid.UUID, operation *models.EquipmentOperation) ([]*models.EquipmentOperationTask, error) {
	return s.repo.GetOperationTasks(ctx, realmID, operation)
}

// GetOperationTaskByID — правило по id.
func (s *EquipmentOperationTaskService) GetOperationTaskByID(ctx context.Context, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentOperationTask, error) {
	return s.repo.GetOperationTaskByID(ctx, nil, id, realmID)
}

// CreateOperationTask создаёт правило после проверки обязательных полей.
func (s *EquipmentOperationTaskService) CreateOperationTask(ctx context.Context, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error) {
	if err := validateOperationTask(dto); err != nil {
		return nil, err
	}

	var task *models.EquipmentOperationTask
	err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		var txErr error
		task, txErr = s.repo.CreateOperationTask(ctx, tx, dto)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateOperationTask правит правило; id и realm приходят из маршрута/заголовка.
func (s *EquipmentOperationTaskService) UpdateOperationTask(ctx context.Context, dto *models.EquipmentOperationTaskDTO) error {
	if dto.ID == nil || *dto.ID == uuid.Nil {
		return models.ErrNotValid
	}
	if err := validateOperationTask(dto); err != nil {
		return err
	}
	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		return s.repo.UpdateOperationTask(ctx, tx, dto)
	})
}

// DeleteOperationTask удаляет правило.
func (s *EquipmentOperationTaskService) DeleteOperationTask(ctx context.Context, dto *models.DelEquipmentOperationTaskDTO) error {
	if dto.ID == uuid.Nil || dto.RealmID == uuid.Nil {
		return models.ErrNotValid
	}
	return s.repo.DeleteOperationTask(ctx, dto)
}

func validateOperationTask(dto *models.EquipmentOperationTaskDTO) error {
	if dto == nil || dto.RealmID == uuid.Nil {
		return models.ErrNotValid
	}
	if dto.Name == "" || !dto.Operation.IsValid() || dto.CategoryID == uuid.Nil || dto.GroupID == uuid.Nil {
		return models.ErrNotValid
	}
	if dto.Action == "" {
		// Клиент может не слать действие: пока единственное — создать заявку.
		dto.Action = models.EquipmentActionCreateTicket
	}
	if !dto.Action.IsValid() {
		return models.ErrNotValid
	}
	if dto.Priority == "" {
		dto.Priority = string(models.PriorityMedium)
	}
	return nil
}
