package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

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

	return s.tm.WithinTransaction(ctx, func(tx repository.Tx) error {
		return s.openComponents(ctx, tx, &models.OpenComponentsDTO{
			RealmID: dto.RealmID, KitID: dto.ID, Components: dto.Components,
			Reason: dto.Reason, Actor: dto.Actor,
		})
	})
}

// openComponents открывает членство компонентов в комплекте и пишет событие
// журнала на каждый — в переданной транзакции. Один набор правил на два входа:
// AddComponents и Create (стартовый состав нового комплекта).
func (s *EquipmentService) openComponents(ctx context.Context, tx repository.Tx, dto *models.OpenComponentsDTO) error {
	opID := uuid.New()
	now := time.Now()
	actor := actorID(dto.Actor)

	for _, comp := range dto.Components {
		if comp.ComponentID == dto.KitID {
			return models.ErrEquipmentCycle
		}
		item, err := s.repo.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: comp.ComponentID, RealmID: dto.RealmID})
		if err != nil {
			return fmt.Errorf("failed to get component. error: %w", err)
		}
		// Логический комплект — это про устройства (ноутбук, монитор, мышь),
		// а комплектующие и расходники — про физический состав (parent_id).
		// Оси разные, смешивать их нельзя: проверка одна на оба входа
		// (AddComponents и стартовый состав в Create).
		if item.TypeKind.CanBePart() {
			return models.ErrEquipmentPartInKit
		}
		open, err := s.repo.GetOpenByComponent(ctx, comp.ComponentID)
		if err != nil && !errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("failed to get open composition. error: %w", err)
		}
		if err == nil {
			if open.KitID == dto.KitID {
				continue
			}
			return models.ErrEquipmentComponentInKit
		}

		if err := s.repo.OpenComposition(ctx, tx, &models.KitComposition{
			KitID: dto.KitID, ComponentID: comp.ComponentID, StartedAt: now,
			StartedBy: actor, Reason: dto.Reason,
		}); err != nil {
			return fmt.Errorf("failed to open composition. error: %w", err)
		}
		if err := s.repo.CreateEvent(ctx, tx, &models.EquipmentEvent{
			EquipmentID: dto.KitID, EventType: models.EquipmentEventComponentAdded, OperationID: opID,
			ComponentID: &comp.ComponentID, NewValue: componentJSON(item),
			UserID: actor, Comment: dto.Reason,
		}); err != nil {
			return fmt.Errorf("failed to create event. error: %w", err)
		}
	}
	return nil
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

	return s.tm.WithinTransaction(ctx, func(tx repository.Tx) error {
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
				ComponentID: &comp.ComponentID, OldValue: componentRefJSON(comp.ComponentID, open.ComponentName),
				UserID: actorID, Comment: dto.Reason,
			}); err != nil {
				return fmt.Errorf("failed to create event. error: %w", err)
			}
		}
		return nil
	})
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

// componentRefJSON — снимок исключаемого компонента. Полного набора полей под рукой
// нет (открытая запись состава знает только имя из compositionSelectBase), но ленте
// истории для подписи этого достаточно — и она не тянет справочник оборудования.
func componentRefJSON(id uuid.UUID, name string) []byte {
	data, err := json.Marshal(map[string]any{"equipmentId": id, "name": name})
	if err != nil {
		return nil
	}
	return data
}
