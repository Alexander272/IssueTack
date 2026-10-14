package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// EquipmentService — сервис учёта оборудования (справочник типов и единицы).
type EquipmentService struct {
	repo repository.Equipment
	tm   TransactionManager
}

// NewEquipmentService создаёт EquipmentService.
func NewEquipmentService(repo repository.Equipment, tm TransactionManager) *EquipmentService {
	return &EquipmentService{repo: repo, tm: tm}
}

// Equipment — интерфейс работы с оборудованием.
type Equipment interface {
	// GetTypes возвращает список типов оборудования реалма.
	GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error)
	// GetTypeByID возвращает тип оборудования по идентификатору.
	GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error)
	// CreateType создаёт тип оборудования.
	CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	// UpdateType обновляет тип оборудования.
	UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	// DeleteType удаляет тип оборудования.
	DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error

	// Get возвращает список единиц оборудования реалма с фильтрами и пагинацией.
	Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error)
	// GetByID возвращает единицу оборудования.
	GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error)
	// GetChildren возвращает состав комплекта (непосредственных потомков).
	GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error)
	// Create создаёт единицу оборудования.
	Create(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	// Update обновляет единицу оборудования (с проверкой циклов в дереве комплектов).
	Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	// Delete удаляет единицу оборудования.
	Delete(ctx context.Context, dto *models.DelEquipmentDTO) error

	// Assign выдаёт единицу (или комплект) держателю, пропагируя на состав.
	Assign(ctx context.Context, dto *models.AssignEquipmentDTO) error
	// Unassign возвращает единицу на хранение (снимает выдачу).
	Unassign(ctx context.Context, dto *models.UnassignEquipmentDTO) error
	// ChangeStatus меняет статус единицы оборудования.
	ChangeStatus(ctx context.Context, dto *models.ChangeStatusDTO) (*models.Equipment, error)
	// GetAssignmentHistory возвращает историю выдач единицы.
	GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error)
	// GetEvents возвращает журнал событий единицы.
	GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error)

	// GetComposition возвращает текущий состав логического комплекта.
	GetComposition(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.Equipment, error)
	// GetCompositionHistory возвращает историю состава комплекта.
	GetCompositionHistory(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.KitComposition, error)
	// AddComponents добавляет единицы в состав комплекта.
	AddComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error
	// RemoveComponents исключает единицы из состава комплекта.
	RemoveComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error

	// GetActs возвращает список актов реалма.
	GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error)
	// GetActByID возвращает акт по идентификатору.
	GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error)
	// CreateAct формирует акт вручную.
	CreateAct(ctx context.Context, dto *models.EquipmentActDTO) (*models.EquipmentAct, error)
}

// GetTypes возвращает список типов оборудования реалма.
func (s *EquipmentService) GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error) {
	data, err := s.repo.GetTypes(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment types. error: %w", err)
	}
	return data, nil
}

// GetTypeByID возвращает тип оборудования по идентификатору.
func (s *EquipmentService) GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error) {
	data, err := s.repo.GetTypeByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment type by id. error: %w", err)
	}
	return data, nil
}

// CreateType создаёт тип оборудования.
func (s *EquipmentService) CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	if err := s.repo.CreateType(ctx, dto); err != nil {
		return fmt.Errorf("failed to create equipment type. error: %w", err)
	}
	return nil
}

// UpdateType обновляет тип оборудования.
func (s *EquipmentService) UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	if err := s.repo.UpdateType(ctx, dto); err != nil {
		return fmt.Errorf("failed to update equipment type. error: %w", err)
	}
	return nil
}

// DeleteType удаляет тип оборудования, если к нему не привязано ни одной единицы.
func (s *EquipmentService) DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error {
	count, err := s.repo.CountByType(ctx, dto.RealmID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to count equipment by type. error: %w", err)
	}
	if count > 0 {
		return models.ErrEquipmentTypeInUse
	}

	if err := s.repo.DeleteType(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete equipment type. error: %w", err)
	}
	return nil
}

// Get возвращает список единиц оборудования реалма с фильтрами и пагинацией.
func (s *EquipmentService) Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error) {
	data, total, err := s.repo.Get(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	return data, total, nil
}

// GetByID возвращает единицу оборудования.
func (s *EquipmentService) GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment by id. error: %w", err)
	}
	return data, nil
}

// GetChildren возвращает состав комплекта (непосредственных потомков).
func (s *EquipmentService) GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error) {
	data, err := s.repo.GetChildren(ctx, realmID, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment children. error: %w", err)
	}
	return data, nil
}

// Create создаёт единицу оборудования.
func (s *EquipmentService) Create(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	normalizeEquipmentDTO(dto)

	if err := s.ensureParentCanBePart(ctx, dto); err != nil {
		return nil, err
	}

	data, err := s.repo.Create(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf("failed to create equipment. error: %w", err)
	}
	return data, nil
}

// Update обновляет единицу оборудования. Смена родителя проверяется на цикл:
// новый родитель не может быть потомком (или самим) обновляемой единицы.
func (s *EquipmentService) Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	if dto.ID == nil {
		return nil, models.ErrInvalidInput
	}
	normalizeEquipmentDTO(dto)

	if err := s.ensureParentCanBePart(ctx, dto); err != nil {
		return nil, err
	}

	if dto.ParentID != nil {
		if *dto.ParentID == *dto.ID {
			return nil, models.ErrEquipmentCycle
		}
		ancestors, err := s.repo.GetAncestors(ctx, dto.RealmID, *dto.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get equipment ancestors. error: %w", err)
		}
		for _, ancestor := range ancestors {
			if ancestor == *dto.ID {
				return nil, models.ErrEquipmentCycle
			}
		}
	}

	data, err := s.repo.Update(ctx, dto)
	if err != nil {
		return nil, fmt.Errorf("failed to update equipment. error: %w", err)
	}
	return data, nil
}

// Delete удаляет единицу оборудования. Потомки (если были) автоматически
// отсоединяются от комплекта (parent_id = NULL).
func (s *EquipmentService) Delete(ctx context.Context, dto *models.DelEquipmentDTO) error {
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete equipment. error: %w", err)
	}
	return nil
}

// ensureParentCanBePart проверяет, что единица, у которой задан родитель
// (физический состав, parent_id), относится к типу, допустимому в составе:
// комплектующему или расходнику. Логические комплекты (is_kit) не
// ограничиваются — там состав ведётся через kit_compositions.
func (s *EquipmentService) ensureParentCanBePart(ctx context.Context, dto *models.EquipmentDTO) error {
	if dto.ParentID == nil {
		return nil
	}
	eqType, err := s.repo.GetTypeByID(ctx, &models.GetEquipmentTypeByIdDTO{ID: dto.TypeID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment type. error: %w", err)
	}
	if !eqType.Kind.CanBePart() {
		return models.ErrEquipmentParentNotPart
	}
	return nil
}

// normalizeEquipmentDTO подчищает опциональные строки: пустые -> NULL (иначе
// partial unique-индексы и CHECK работали бы с ”) и specs без значения -> {}.
func normalizeEquipmentDTO(dto *models.EquipmentDTO) {
	emptyToNil := func(v *string) *string {
		if v != nil && *v == "" {
			return nil
		}
		return v
	}
	dto.SerialNumber = emptyToNil(dto.SerialNumber)
	dto.InventoryNumber = emptyToNil(dto.InventoryNumber)
	dto.HolderName = emptyToNil(dto.HolderName)
	dto.Location = emptyToNil(dto.Location)
}

// collectSubtree собирает единицу и весь её состав: физических потомков
// (parent_id) и, для комплектов, текущих логических участников (kit_compositions).
func (s *EquipmentService) collectSubtree(ctx context.Context, realmID, rootID uuid.UUID) ([]*models.Equipment, error) {
	seen := map[uuid.UUID]bool{}
	result := []*models.Equipment{}
	queue := []uuid.UUID{rootID}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true

		item, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: id, RealmID: realmID})
		if err != nil {
			return nil, err
		}
		result = append(result, item)

		children, err := s.repo.GetChildren(ctx, realmID, id)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			queue = append(queue, c.ID)
		}
		if item.IsKit {
			members, err := s.repo.GetOpenComposition(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				queue = append(queue, m.ID)
			}
		}
	}
	return result, nil
}

// Assign выдаёт единицу (или комплект) держателю. Выдача пропагируется на весь
// состав. Если кто-то из состава уже выдан другому держателю — конфликт (обход
// только через force).
func (s *EquipmentService) Assign(ctx context.Context, dto *models.AssignEquipmentDTO) error {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return models.ErrEquipmentNotAssignable
	}
	if dto.UserID == nil && dto.HolderName == nil && dto.SiteID == nil && dto.Location == nil {
		return models.ErrEquipmentNoHoldersOrLocation
	}

	affected, err := s.collectSubtree(ctx, dto.RealmID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to collect equipment subtree. error: %w", err)
	}

	for _, eq := range affected {
		open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("failed to get open assignment. error: %w", err)
		}
		if err == nil && !sameHolder(open, dto) && !dto.Force {
			return models.ErrEquipmentAssigned
		}
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		now := time.Now()
		actorID := actorID(dto.Actor)

		for _, eq := range affected {
			open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
			if err != nil && !errors.Is(err, models.ErrNotFound) {
				return fmt.Errorf("failed to get open assignment. error: %w", err)
			}
			if err == nil {
				if err := s.repo.CloseAssignment(ctx, tx, open.ID, now, ""); err != nil {
					return fmt.Errorf("failed to close assignment. error: %w", err)
				}
			}

			if err := s.repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
				EquipmentID: eq.ID,
				UserID:      dto.UserID,
				HolderName:  dto.HolderName,
				SiteID:      dto.SiteID,
				Location:    dto.Location,
				AssignedAt:  now,
				AssignedBy:  actorID,
				Comment:     dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create assignment. error: %w", err)
			}

			if err := s.repo.SetHolderAndStatus(ctx, tx, eq.ID, dto.UserID, dto.HolderName, dto.SiteID, dto.Location, models.EquipmentStatusInUse); err != nil {
				return fmt.Errorf("failed to set holder. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: eq.ID, EventType: models.EquipmentEventAssigned, OperationID: opID,
				NewValue: assignmentJSON(dto), UserID: actorID, Comment: dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}

		if dto.SkipAct {
			return nil
		}
		return s.createActTx(ctx, tx, &models.EquipmentActDTO{
			RealmID:         dto.RealmID,
			KitEquipmentID:  &dto.ID,
			RecipientUserID: dto.UserID,
			RecipientName:   dto.HolderName,
			SiteID:          dto.SiteID,
			Location:        dto.Location,
			Comment:         dto.Comment,
			Actor:           dto.Actor,
		}, affected)
	})
}

// Unassign снимает выдачу с единицы и её состава, возвращая на хранение.
func (s *EquipmentService) Unassign(ctx context.Context, dto *models.UnassignEquipmentDTO) error {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return models.ErrEquipmentNotAssignable
	}
	if dto.SiteID == nil && dto.Location == nil {
		return models.ErrEquipmentNoHoldersOrLocation
	}

	affected, err := s.collectSubtree(ctx, dto.RealmID, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to collect equipment subtree. error: %w", err)
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		now := time.Now()
		actorID := actorID(dto.Actor)

		for _, eq := range affected {
			open, err := s.repo.GetOpenAssignment(ctx, eq.ID)
			if err != nil && !errors.Is(err, models.ErrNotFound) {
				return fmt.Errorf("failed to get open assignment. error: %w", err)
			}
			if err == nil {
				if err := s.repo.CloseAssignment(ctx, tx, open.ID, now, dto.Comment); err != nil {
					return fmt.Errorf("failed to close assignment. error: %w", err)
				}
			}

			if err := s.repo.CreateAssignment(ctx, tx, &models.EquipmentAssignment{
				EquipmentID: eq.ID,
				SiteID:      dto.SiteID,
				Location:    dto.Location,
				AssignedAt:  now,
				AssignedBy:  actorID,
				Comment:     dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create assignment. error: %w", err)
			}

			if err := s.repo.SetHolderAndStatus(ctx, tx, eq.ID, nil, nil, dto.SiteID, dto.Location, models.EquipmentStatusInStock); err != nil {
				return fmt.Errorf("failed to set holder. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: eq.ID, EventType: models.EquipmentEventUnassigned, OperationID: opID,
				NewValue: assignmentJSON(&models.AssignEquipmentDTO{SiteID: dto.SiteID, Location: dto.Location}),
				UserID:   actorID, Comment: dto.Comment,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}
		return nil
	})
}

// ChangeStatus меняет статус единицы. Списанную единицу менять нельзя.
func (s *EquipmentService) ChangeStatus(ctx context.Context, dto *models.ChangeStatusDTO) (*models.Equipment, error) {
	root, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if root.Status == models.EquipmentStatusWrittenOff {
		return nil, models.ErrEquipmentWrittenOff
	}

	err = s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.SetStatus(ctx, tx, dto.RealmID, dto.ID, dto.Status); err != nil {
			return fmt.Errorf("failed to set status. error: %w", err)
		}
		eventType := models.EquipmentEventStatusChanged
		if dto.Status == models.EquipmentStatusWrittenOff {
			eventType = models.EquipmentEventWrittenOff
		}
		if dto.Status == models.EquipmentStatusRepair {
			eventType = models.EquipmentEventRepaired
		}
		return s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
			EquipmentID: dto.ID, EventType: eventType, OperationID: uuid.New(),
			OldValue: statusJSON(root.Status), NewValue: statusJSON(dto.Status),
			UserID: actorID(dto.Actor), Comment: dto.Comment,
		})
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
}

// GetAssignmentHistory возвращает историю выдач единицы.
func (s *EquipmentService) GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error) {
	data, err := s.repo.GetAssignmentHistory(ctx, realmID, equipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment history. error: %w", err)
	}
	return data, nil
}

// GetEvents возвращает журнал событий единицы.
func (s *EquipmentService) GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error) {
	data, err := s.repo.GetEvents(ctx, realmID, equipmentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment events. error: %w", err)
	}
	return data, nil
}

// GetComposition возвращает текущий состав логического комплекта.
func (s *EquipmentService) GetComposition(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.Equipment, error) {
	kit, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: kitID, RealmID: realmID})
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if !kit.IsKit {
		return nil, models.ErrEquipmentNotKit
	}
	data, err := s.repo.GetOpenComposition(ctx, kitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get composition. error: %w", err)
	}
	return data, nil
}

// GetCompositionHistory возвращает историю состава комплекта.
func (s *EquipmentService) GetCompositionHistory(ctx context.Context, realmID, kitID uuid.UUID) ([]*models.KitComposition, error) {
	if _, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: kitID, RealmID: realmID}); err != nil {
		return nil, fmt.Errorf("failed to get equipment. error: %w", err)
	}
	data, err := s.repo.GetCompositionHistory(ctx, kitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get composition history. error: %w", err)
	}
	return data, nil
}

// AddComponents добавляет единицы в состав комплекта. Компонент одновременно
// может входить только в один комплект.
func (s *EquipmentService) AddComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error {
	kit, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if !kit.IsKit {
		return models.ErrEquipmentNotKit
	}
	if len(dto.Components) == 0 {
		return models.ErrInvalidInput
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		now := time.Now()
		actorID := actorID(dto.Actor)

		for _, comp := range dto.Components {
			if comp.ComponentID == dto.ID {
				return models.ErrEquipmentCycle
			}
			item, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: comp.ComponentID, RealmID: dto.RealmID})
			if err != nil {
				return fmt.Errorf("failed to get component. error: %w", err)
			}
			open, err := s.repo.GetOpenByComponent(ctx, comp.ComponentID)
			if err != nil && !errors.Is(err, models.ErrNotFound) {
				return fmt.Errorf("failed to get open composition. error: %w", err)
			}
			if err == nil {
				if open.KitID == dto.ID {
					continue
				}
				return models.ErrEquipmentComponentInKit
			}

			if err := s.repo.OpenComposition(ctx, tx, &models.KitComposition{
				KitID: dto.ID, ComponentID: comp.ComponentID, StartedAt: now,
				StartedBy: actorID, Reason: dto.Reason,
			}); err != nil {
				return fmt.Errorf("failed to open composition. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: dto.ID, EventType: models.EquipmentEventComponentAdded, OperationID: opID,
				ComponentID: &comp.ComponentID, NewValue: componentJSON(item),
				UserID: actorID, Comment: dto.Reason,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}
		return nil
	})
}

// RemoveComponents исключает единицы из состава комплекта.
func (s *EquipmentService) RemoveComponents(ctx context.Context, dto *models.KitCompositionOpDTO) error {
	kit, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: dto.ID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get equipment. error: %w", err)
	}
	if !kit.IsKit {
		return models.ErrEquipmentNotKit
	}
	if len(dto.Components) == 0 {
		return models.ErrInvalidInput
	}

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		opID := uuid.New()
		actorID := actorID(dto.Actor)

		for _, comp := range dto.Components {
			open, err := s.repo.GetOpenByComponent(ctx, comp.ComponentID)
			if err != nil {
				if errors.Is(err, models.ErrNotFound) {
					continue
				}
				return fmt.Errorf("failed to get open composition. error: %w", err)
			}
			if open.KitID != dto.ID {
				return models.ErrEquipmentComponentInKit
			}

			if err := s.repo.CloseOpenByComponent(ctx, tx, comp.ComponentID, actorID, dto.Reason, nil); err != nil {
				return fmt.Errorf("failed to close composition. error: %w", err)
			}
			if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
				EquipmentID: dto.ID, EventType: models.EquipmentEventComponentRemoved, OperationID: opID,
				ComponentID: &comp.ComponentID, UserID: actorID, Comment: dto.Reason,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}
		return nil
	})
}

// GetActs возвращает список актов реалма.
func (s *EquipmentService) GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error) {
	data, total, err := s.repo.GetActs(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get acts. error: %w", err)
	}
	return data, total, nil
}

// GetActByID返回 акт по идентификатору.
func (s *EquipmentService) GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error) {
	data, err := s.repo.GetActByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get act. error: %w", err)
	}
	return data, nil
}

// CreateAct формирует акт вручную по текущему составу комплекта либо по единице.
func (s *EquipmentService) CreateAct(ctx context.Context, dto *models.EquipmentActDTO) (*models.EquipmentAct, error) {
	var affected []*models.Equipment
	if dto.KitEquipmentID != nil {
		items, err := s.collectSubtree(ctx, dto.RealmID, *dto.KitEquipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to collect equipment subtree. error: %w", err)
		}
		affected = items
	}
	if len(affected) == 0 {
		return nil, models.ErrEquipmentActGeneration
	}

	var actID uuid.UUID
	err := s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		act, err := s.buildAct(ctx, tx, dto, affected)
		if err != nil {
			return err
		}
		actID = act.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetActByID(ctx, &models.GetEquipmentActByIDDTO{ID: actID, RealmID: dto.RealmID})
}

// createActTx формирует акт выдачи в существующей транзакции.
func (s *EquipmentService) createActTx(ctx context.Context, tx postgres.Tx, dto *models.EquipmentActDTO, affected []*models.Equipment) error {
	_, err := s.buildAct(ctx, tx, dto, affected)
	return err
}

// buildAct собирает и сохраняет акт со снимком состава.
func (s *EquipmentService) buildAct(ctx context.Context, tx postgres.Tx, dto *models.EquipmentActDTO, affected []*models.Equipment) (*models.EquipmentAct, error) {
	content, err := actContent(affected)
	if err != nil {
		return nil, fmt.Errorf("failed to build act content. error: %w", err)
	}

	seq, err := s.repo.NextActSeq(ctx, tx, dto.RealmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get act seq. error: %w", err)
	}
	number := fmt.Sprintf("АКТ-%d-%04d", time.Now().Year(), seq)

	act := &models.EquipmentAct{
		RealmID:         dto.RealmID,
		Number:          number,
		KitEquipmentID:  dto.KitEquipmentID,
		RecipientUserID: dto.RecipientUserID,
		RecipientName:   dto.RecipientName,
		SiteID:          dto.SiteID,
		Location:        dto.Location,
		IssuedBy:        actorID(dto.Actor),
		IssuedAt:        time.Now(),
		Content:         content,
		Comment:         dto.Comment,
	}
	if err := s.repo.CreateAct(ctx, tx, act); err != nil {
		return nil, fmt.Errorf("failed to create act. error: %w", err)
	}
	return act, nil
}

func actorID(a *models.Actor) *uuid.UUID {
	if a == nil || a.ID == uuid.Nil {
		return nil
	}
	id := a.ID
	return &id
}

func sameHolder(open *models.EquipmentAssignment, dto *models.AssignEquipmentDTO) bool {
	if open == nil {
		return false
	}
	if !sameUUID(open.UserID, dto.UserID) || !sameUUID(open.SiteID, dto.SiteID) {
		return false
	}
	return sameString(open.HolderName, dto.HolderName) && sameString(open.Location, dto.Location)
}

func sameUUID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// actPosition — позиция снимка состава в акте.
type actPosition struct {
	EquipmentID     uuid.UUID `json:"equipmentId"`
	TypeName        string    `json:"typeName"`
	Name            string    `json:"name"`
	SerialNumber    *string   `json:"serialNumber"`
	InventoryNumber *string   `json:"inventoryNumber"`
}

// actContent формирует jsonb-снимок состава для акта.
func actContent(affected []*models.Equipment) ([]byte, error) {
	positions := make([]actPosition, 0, len(affected))
	for _, eq := range affected {
		positions = append(positions, actPosition{
			EquipmentID:     eq.ID,
			TypeName:        eq.TypeName,
			Name:            eq.Name,
			SerialNumber:    eq.SerialNumber,
			InventoryNumber: eq.InventoryNumber,
		})
	}
	data, err := json.Marshal(positions)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// assignmentJSON сериализует держателя/локацию для old/new_value события.
func assignmentJSON(dto *models.AssignEquipmentDTO) []byte {
	data, err := json.Marshal(map[string]any{
		"userId":     dto.UserID,
		"holderName": dto.HolderName,
		"siteId":     dto.SiteID,
		"location":   dto.Location,
	})
	if err != nil {
		return nil
	}
	return data
}

// statusJSON сериализует статус для old/new_value события.
func statusJSON(status models.EquipmentStatus) []byte {
	data, err := json.Marshal(map[string]any{"status": status})
	if err != nil {
		return nil
	}
	return data
}

// componentJSON сериализует компонент для old/new_value события.
func componentJSON(eq *models.Equipment) []byte {
	data, err := json.Marshal(actPosition{
		EquipmentID: eq.ID, TypeName: eq.TypeName, Name: eq.Name,
		SerialNumber: eq.SerialNumber, InventoryNumber: eq.InventoryNumber,
	})
	if err != nil {
		return nil
	}
	return data
}
