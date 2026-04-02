package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SiteRepo struct {
	db *pgxpool.Pool
}

func NewSiteRepo(db *pgxpool.Pool) *SiteRepo {
	return &SiteRepo{
		db: db,
	}
}

type Sites interface {
	GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error)
	Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error)
	Create(ctx context.Context, dto *models.SiteDTO) error
	Update(ctx context.Context, dto *models.SiteDTO) error
	Delete(ctx context.Context, dto *models.DelSiteDTO) error
}

func (r *SiteRepo) GetByID(ctx context.Context, req *models.GetSiteByIdDTO) (*models.Site, error) {
	query := fmt.Sprintf(`SELECT id, name, address, created_at, updated_at FROM %s WHERE id = $1`,
		Tables.Sites,
	)

	site := &models.Site{}
	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&site.ID,
		&site.Name,
		&site.Address,
		&site.CreatedAt,
		&site.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return site, nil
}

func (r *SiteRepo) Get(ctx context.Context, req *models.GetSitesDTO) ([]*models.Site, error) {
	query := fmt.Sprintf(`SELECT id, name, address, created_at, updated_at FROM %s`, Tables.Sites)

	var data []*models.Site
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Site{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Address, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *SiteRepo) Create(ctx context.Context, dto *models.SiteDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, name, address) VALUES ($1, $2, $3)`, Tables.Sites)
	dto.ID = uuid.New()

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Address)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *SiteRepo) Update(ctx context.Context, dto *models.SiteDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, address=$3 WHERE id=$1`, Tables.Sites)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Address)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *SiteRepo) Delete(ctx context.Context, dto *models.DelSiteDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Sites)

	_, err := r.db.Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
