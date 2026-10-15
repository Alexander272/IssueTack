package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

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
