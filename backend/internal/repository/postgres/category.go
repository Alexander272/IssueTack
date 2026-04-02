package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepo struct {
	db *pgxpool.Pool
}

func NewCategoryRepo(db *pgxpool.Pool) *CategoryRepo {
	return &CategoryRepo{
		db: db,
	}
}

type Categories interface {
	Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error)
	GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error)
	Create(ctx context.Context, dto *models.CategoryDTO) error
	Update(ctx context.Context, dto *models.CategoryDTO) error
	Delete(ctx context.Context, dto *models.DelCategoryDTO) error
}

func (r *CategoryRepo) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	query := fmt.Sprintf(`SELECT id, name, description, group_id, priority, is_active, created_at, updated_at FROM %s`,
		Tables.Categories,
	)

	var data []*models.Category
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Category{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.GroupID, &item.Priority, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *CategoryRepo) GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
	query := fmt.Sprintf(`SELECT id, name, description, group_id, priority, is_active, created_at, updated_at FROM %s WHERE id = $1`,
		Tables.Categories,
	)

	category := &models.Category{}
	err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.GroupID,
		&category.Priority,
		&category.IsActive,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return category, nil
}

func (r *CategoryRepo) Create(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, name, description, group_id, priority, is_active) VALUES ($1, $2, $3, $4, $5, $6)`,
		Tables.Categories,
	)
	dto.ID = uuid.New()

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.GroupID, dto.Priority, dto.IsActive)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, group_id=$4, priority=$5, is_active=$6 WHERE id=$1`,
		Tables.Categories,
	)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.GroupID, dto.Priority, dto.IsActive)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, dto *models.DelCategoryDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.Categories)

	_, err := r.db.Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
