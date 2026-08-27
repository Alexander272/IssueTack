package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

// SiteService — сервис работы с сайтами.
type SiteService struct {
	repo repository.Sites
}

// NewSiteService создаёт SiteService.
func NewSiteService(repo repository.Sites) *SiteService {
	return &SiteService{repo: repo}
}

// Sites — интерфейс работы с сайтами.
type Sites interface {
	// Get возвращает список сайтов.
	Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error)
	// GetByID возвращает сайт по идентификатору.
	GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error)
	// Create создаёт сайт.
	Create(ctx context.Context, dto *models.SiteDTO) error
	// Update обновляет сайт.
	Update(ctx context.Context, dto *models.SiteDTO) error
	// Delete удаляет сайт.
	Delete(ctx context.Context, dto *models.DelSiteDTO) error
}

// Get возвращает список сайтов.
func (s *SiteService) Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites. error: %w", err)
	}
	return data, nil
}

// GetByID возвращает сайт по идентификатору.
func (s *SiteService) GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get site by id. error: %w", err)
	}
	return data, nil
}

// Create создаёт сайт.
func (s *SiteService) Create(ctx context.Context, dto *models.SiteDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create site. error: %w", err)
	}
	return nil
}

// Update обновляет сайт.
func (s *SiteService) Update(ctx context.Context, dto *models.SiteDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update site. error: %w", err)
	}
	return nil
}

// Delete удаляет сайт.
func (s *SiteService) Delete(ctx context.Context, dto *models.DelSiteDTO) error {
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete site. error: %w", err)
	}
	return nil
}
