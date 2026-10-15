package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// GetActs возвращает список актов реалма.
func (s *EquipmentService) GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error) {
	data, total, err := s.repo.GetActs(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get acts. error: %w", err)
	}
	return data, total, nil
}

// GetActByID возвращает акт по идентификатору.
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
