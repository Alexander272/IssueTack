package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// GetKitTemplates возвращает шаблоны комплектов реалма вместе с позициями.
func (s *EquipmentService) GetKitTemplates(ctx context.Context, realmID uuid.UUID) ([]*models.EquipmentKitTemplate, error) {
	data, err := s.repo.GetKitTemplates(ctx, realmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get kit templates. error: %w", err)
	}
	return data, nil
}

// GetKitTemplateByID возвращает шаблон комплекта по идентификатору.
func (s *EquipmentService) GetKitTemplateByID(ctx context.Context, req *models.GetEquipmentKitTemplateByIdDTO) (*models.EquipmentKitTemplate, error) {
	data, err := s.repo.GetKitTemplateByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get kit template by id. error: %w", err)
	}
	return data, nil
}

// CreateKitTemplate создаёт шаблон комплекта.
func (s *EquipmentService) CreateKitTemplate(ctx context.Context, dto *models.EquipmentKitTemplateDTO) error {
	normalizeKitTemplateDTO(dto)

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.CreateKitTemplate(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to create kit template. error: %w", err)
		}
		return nil
	})
}

// UpdateKitTemplate обновляет шаблон комплекта; состав заменяется целиком.
func (s *EquipmentService) UpdateKitTemplate(ctx context.Context, dto *models.EquipmentKitTemplateDTO) error {
	if dto.ID == nil {
		return models.ErrInvalidInput
	}
	normalizeKitTemplateDTO(dto)

	return s.tm.WithinTransaction(ctx, func(tx postgres.Tx) error {
		if err := s.repo.UpdateKitTemplate(ctx, tx, dto); err != nil {
			return fmt.Errorf("failed to update kit template. error: %w", err)
		}
		return nil
	})
}

// DeleteKitTemplate удаляет шаблон комплекта.
func (s *EquipmentService) DeleteKitTemplate(ctx context.Context, dto *models.DelEquipmentKitTemplateDTO) error {
	if err := s.repo.DeleteKitTemplate(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete kit template. error: %w", err)
	}
	return nil
}

// normalizeKitTemplateDTO чистит состав шаблона: пустые типы отбрасываются,
// sort_order проставляется по порядку следования. Повторяющиеся типы НЕ
// схлопываются: в шаблоне «Стандартное рабочее место» два монитора — это две
// позиции одного типа, а не дубль. nil-состав остаётся nil — для правки это
// «не трогать позиции», в отличие от явного [].
func normalizeKitTemplateDTO(dto *models.EquipmentKitTemplateDTO) {
	if dto.Items == nil {
		return
	}

	items := make([]models.EquipmentKitTemplateItemDTO, 0, len(dto.Items))
	for _, item := range dto.Items {
		if item.TypeID == uuid.Nil {
			continue
		}
		items = append(items, models.EquipmentKitTemplateItemDTO{TypeID: item.TypeID, SortOrder: len(items)})
	}
	dto.Items = items
}
