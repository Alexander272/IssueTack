package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
)

// GetShops возвращает справочник магазинов реалма — варианты для поля «Магазин».
func (s *EquipmentService) GetShops(ctx context.Context, req *models.GetEquipmentShopsDTO) ([]*models.EquipmentShop, error) {
	data, err := s.repo.GetShops(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment shops. error: %w", err)
	}
	return data, nil
}
