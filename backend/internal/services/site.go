package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
)

type SiteService struct {
	repo repository.Sites
}

func NewSiteService(repo repository.Sites) *SiteService {
	return &SiteService{repo: repo}
}

type Sites interface {
	Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error)
	GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error)
	Create(ctx context.Context, dto *models.SiteDTO) error
	Update(ctx context.Context, dto *models.SiteDTO) error
	Delete(ctx context.Context, dto *models.DelSiteDTO) error
}

func (s *SiteService) Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error) {
	data, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get sites. error: %w", err)
	}
	return data, nil
}

func (s *SiteService) GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error) {
	data, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get site by id. error: %w", err)
	}
	return data, nil
}

func (s *SiteService) Create(ctx context.Context, dto *models.SiteDTO) error {
	if err := s.repo.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create site. error: %w", err)
	}
	return nil
}

func (s *SiteService) Update(ctx context.Context, dto *models.SiteDTO) error {
	if err := s.repo.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update site. error: %w", err)
	}
	return nil
}

func (s *SiteService) Delete(ctx context.Context, dto *models.DelSiteDTO) error {
	if err := s.repo.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete site. error: %w", err)
	}
	return nil
}
