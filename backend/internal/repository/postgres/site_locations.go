package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SiteLocationRepo — справочник расположений площадки. Площадки глобальные,
// поэтому здесь нет realm_id: расположение принадлежит площадке, а не реалму.
type SiteLocationRepo struct {
	db *pgxpool.Pool
}

func NewSiteLocationRepo(db *pgxpool.Pool) *SiteLocationRepo {
	return &SiteLocationRepo{db: db}
}

type SiteLocations interface {
	// GetAll — плоский список всех расположений для общего селектора места:
	// формы больше не тянут локации по одной на каждую площадку.
	GetAll(ctx context.Context) ([]*models.SiteLocation, error)
	GetBySite(ctx context.Context, siteID uuid.UUID) ([]*models.SiteLocation, error)
	GetByID(ctx context.Context, req *models.GetSiteLocationDTO) (*models.SiteLocation, error)
	GetByName(ctx context.Context, siteID uuid.UUID, name string) (*models.SiteLocation, error)
	// GetName резолвит подпись и одновременно проверяет, что расположение
	// принадлежит указанной площадке.
	GetName(ctx context.Context, siteID, locationID uuid.UUID) (string, error)
	Create(ctx context.Context, dto *models.SiteLocationDTO) error
	Update(ctx context.Context, dto *models.SiteLocationDTO) error
	Delete(ctx context.Context, dto *models.DelSiteLocationDTO) error
	// HasReferences сообщает, что расположение указано у оборудования, в
	// выдачах или актах: такое удаление запрещено сервисом.
	HasReferences(ctx context.Context, id uuid.UUID) (bool, error)
}

func (r *SiteLocationRepo) GetAll(ctx context.Context) ([]*models.SiteLocation, error) {
	// Порядок по площадке и названию нужен фронту: Autocomplete группирует
	// только соседние варианты с одинаковой группой, поэтому локации одной
	// площадки должны идти подряд.
	query := fmt.Sprintf(`SELECT sl.id, sl.site_id, sl.name, sl.created_at, sl.updated_at FROM %s sl
		JOIN %s s ON s.id = sl.site_id ORDER BY s.name, sl.name`, Tables.SiteLocations, Tables.Sites)

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.SiteLocation{}
	for rows.Next() {
		item := &models.SiteLocation{}
		if err := rows.Scan(&item.ID, &item.SiteID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *SiteLocationRepo) GetBySite(ctx context.Context, siteID uuid.UUID) ([]*models.SiteLocation, error) {
	query := fmt.Sprintf(`SELECT id, site_id, name, created_at, updated_at FROM %s
		WHERE site_id = $1 ORDER BY name`, Tables.SiteLocations)

	rows, err := r.db.Query(ctx, query, siteID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.SiteLocation{}
	for rows.Next() {
		item := &models.SiteLocation{}
		if err := rows.Scan(&item.ID, &item.SiteID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *SiteLocationRepo) GetByID(ctx context.Context, req *models.GetSiteLocationDTO) (*models.SiteLocation, error) {
	query := fmt.Sprintf(`SELECT id, site_id, name, created_at, updated_at FROM %s
		WHERE id = $1 AND site_id = $2`, Tables.SiteLocations)

	item := &models.SiteLocation{}
	err := r.db.QueryRow(ctx, query, req.ID, req.SiteID).Scan(
		&item.ID, &item.SiteID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *SiteLocationRepo) GetByName(ctx context.Context, siteID uuid.UUID, name string) (*models.SiteLocation, error) {
	query := fmt.Sprintf(`SELECT id, site_id, name, created_at, updated_at FROM %s
		WHERE site_id = $1 AND name = $2`, Tables.SiteLocations)

	item := &models.SiteLocation{}
	err := r.db.QueryRow(ctx, query, siteID, strings.TrimSpace(name)).Scan(
		&item.ID, &item.SiteID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *SiteLocationRepo) GetName(ctx context.Context, siteID, locationID uuid.UUID) (string, error) {
	var name string
	query := fmt.Sprintf(`SELECT name FROM %s WHERE id = $1 AND site_id = $2`, Tables.SiteLocations)
	if err := r.db.QueryRow(ctx, query, locationID, siteID).Scan(&name); err != nil {
		return "", MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return name, nil
}

func (r *SiteLocationRepo) Create(ctx context.Context, dto *models.SiteLocationDTO) error {
	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return models.ErrInvalidInput
	}
	dto.ID = uuid.New()
	query := fmt.Sprintf(`INSERT INTO %s (id, site_id, name) VALUES ($1, $2, $3)`, Tables.SiteLocations)
	if _, err := r.db.Exec(ctx, query, dto.ID, dto.SiteID, dto.Name); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *SiteLocationRepo) Update(ctx context.Context, dto *models.SiteLocationDTO) error {
	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return models.ErrInvalidInput
	}
	query := fmt.Sprintf(`UPDATE %s SET name = $3, updated_at = NOW() WHERE id = $1 AND site_id = $2`, Tables.SiteLocations)
	res, err := r.db.Exec(ctx, query, dto.ID, dto.SiteID, dto.Name)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if res.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *SiteLocationRepo) Delete(ctx context.Context, dto *models.DelSiteLocationDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND site_id = $2`, Tables.SiteLocations)
	res, err := r.db.Exec(ctx, query, dto.ID, dto.SiteID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if res.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *SiteLocationRepo) HasReferences(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS (
		SELECT 1 FROM %s WHERE location_id = $1
		UNION ALL SELECT 1 FROM %s WHERE location_id = $1
		UNION ALL SELECT 1 FROM %s WHERE location_id = $1
	)`, Tables.Equipment, Tables.EquipmentAssignments, Tables.EquipmentActs)

	var exists bool
	if err := r.db.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return exists, nil
}
