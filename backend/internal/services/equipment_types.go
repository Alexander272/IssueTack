package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
)

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
