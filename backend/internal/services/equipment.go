package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// EquipmentService — сервис учёта оборудования (справочник типов и единицы).
type EquipmentService struct {
	repo repository.Equipment
}

// NewEquipmentService создаёт EquipmentService.
func NewEquipmentService(repo repository.Equipment) *EquipmentService {
	return &EquipmentService{repo: repo}
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

// normalizeEquipmentDTO подчищает опциональные строки: пустые -> NULL (иначе
// partial unique-индексы и CHECK работали бы с '') и specs без значения -> {}.
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
