package services

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/google/uuid"
)

// SiteService — сервис работы с сайтами и их расположениями.
type SiteService struct {
	repo      repository.Sites
	locations repository.SiteLocations
}

// NewSiteService создаёт SiteService.
func NewSiteService(repo repository.Sites, locations repository.SiteLocations) *SiteService {
	return &SiteService{repo: repo, locations: locations}
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

	// Расположения площадки.
	GetAllLocations(ctx context.Context) ([]*models.SiteLocation, error)
	GetLocations(ctx context.Context, siteID uuid.UUID) ([]*models.SiteLocation, error)
	CreateLocation(ctx context.Context, dto *models.SiteLocationDTO) error
	UpdateLocation(ctx context.Context, dto *models.SiteLocationDTO) error
	DeleteLocation(ctx context.Context, dto *models.DelSiteLocationDTO) error
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

// GetAllLocations возвращает все расположения сразу — для общего селектора
// места, где площадка и расположение выбраны одним контролом.
func (s *SiteService) GetAllLocations(ctx context.Context) ([]*models.SiteLocation, error) {
	data, err := s.locations.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get site locations. error: %w", err)
	}
	return data, nil
}

// GetLocations возвращает расположения площадки — варианты для карточки площадки.
func (s *SiteService) GetLocations(ctx context.Context, siteID uuid.UUID) ([]*models.SiteLocation, error) {
	data, err := s.locations.GetBySite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site locations. error: %w", err)
	}
	return data, nil
}

// CreateLocation добавляет расположение в карточку площадки.
func (s *SiteService) CreateLocation(ctx context.Context, dto *models.SiteLocationDTO) error {
	if err := s.ensureSite(ctx, dto.SiteID); err != nil {
		return err
	}
	if err := s.locations.Create(ctx, dto); err != nil {
		return fmt.Errorf("failed to create site location. error: %w", err)
	}
	return nil
}

// UpdateLocation переименовывает расположение. Снимки событий и актов хранят
// название на момент операции, поэтому старые записи не меняются.
func (s *SiteService) UpdateLocation(ctx context.Context, dto *models.SiteLocationDTO) error {
	if err := s.locations.Update(ctx, dto); err != nil {
		return fmt.Errorf("failed to update site location. error: %w", err)
	}
	return nil
}

// DeleteLocation запрещает удаление, пока расположение используется: FK сработал
// бы только на уровне БД и вернул сырую ошибку, а здесь понятный ответ клиенту.
func (s *SiteService) DeleteLocation(ctx context.Context, dto *models.DelSiteLocationDTO) error {
	used, err := s.locations.HasReferences(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("failed to check site location references. error: %w", err)
	}
	if used {
		return models.ErrSiteLocationInUse
	}
	if err := s.locations.Delete(ctx, dto); err != nil {
		return fmt.Errorf("failed to delete site location. error: %w", err)
	}
	return nil
}

func (s *SiteService) ensureSite(ctx context.Context, siteID uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, &models.GetSiteByIdDTO{ID: siteID}); err != nil {
		return fmt.Errorf("failed to get site. error: %w", err)
	}
	return nil
}
