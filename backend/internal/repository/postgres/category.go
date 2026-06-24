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
	query := fmt.Sprintf(`SELECT id, name, description, group_id, def_priority, is_active, realm_id, created_at, updated_at FROM %s WHERE realm_id = $1`,
		Tables.Categories,
	)

	data := []*models.Category{}
	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Category{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Description, &item.GroupID, &item.Priority, &item.IsActive,
			&item.RealmID, &item.CreatedAt, &item.UpdatedAt,
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
	query := fmt.Sprintf(`SELECT id, name, description, group_id, def_priority, is_active, realm_id, created_at, updated_at FROM %s WHERE id = $1 AND realm_id = $2`,
		Tables.Categories,
	)

	category := &models.Category{}
	err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.GroupID,
		&category.Priority,
		&category.IsActive,
		&category.RealmID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return category, nil
}

func (r *CategoryRepo) Create(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, name, description, group_id, def_priority, is_active, realm_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		Tables.Categories,
	)
	id := uuid.New()
	dto.ID = &id

	_, err := r.db.Exec(ctx, query, id, dto.Name, dto.Description, dto.GroupID, dto.Priority, dto.IsActive, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, group_id=$4, def_priority=$5, is_active=$6, realm_id=$7 WHERE id=$1`,
		Tables.Categories,
	)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.GroupID, dto.Priority, dto.IsActive, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, dto *models.DelCategoryDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.Categories)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}
