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

// categorySelect возвращает список категорий с присоединённым разделом
// (category_groups). Раздел NULL, если категория не привязана ни к одному.
func (r *CategoryRepo) categorySelect(where string) string {
	return fmt.Sprintf(`SELECT c.id, c.name, c.description, c.group_id, c.category_group_id, c.def_priority, c.is_active, c.realm_id, c.created_at, c.updated_at,
			cg.id, cg.name
		FROM %s c
		LEFT JOIN %s cg ON c.category_group_id = cg.id
		WHERE %s`,
		Tables.Categories, Tables.CategoryGroups, where,
	)
}

func (r *CategoryRepo) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	query := r.categorySelect(`c.realm_id = $1 AND c.group_id IS NOT NULL`) +
		` ORDER BY COALESCE(cg.sort_order, 2147483647), c.name`

	data := []*models.Category{}
	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Category{}
		if err := scanCategory(rows, item); err != nil {
			return nil, MapError(err)
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *CategoryRepo) GetByID(ctx context.Context, req *models.GetCategoryByIdDTO) (*models.Category, error) {
	query := r.categorySelect(`c.id = $1 AND c.realm_id = $2`)

	category := &models.Category{}
	if err := scanCategory(r.db.QueryRow(ctx, query, req.ID, req.RealmID), category); err != nil {
		return nil, MapError(err)
	}
	return category, nil
}

func (r *CategoryRepo) Create(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, name, description, group_id, category_group_id, def_priority, is_active, realm_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		Tables.Categories,
	)
	id := uuid.New()
	dto.ID = &id

	_, err := r.db.Exec(ctx, query, id, dto.Name, dto.Description, dto.GroupID, dto.CategoryGroupID, dto.Priority, dto.IsActive, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, dto *models.CategoryDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, group_id=$4, category_group_id=$5, def_priority=$6, is_active=$7, realm_id=$8 WHERE id=$1`,
		Tables.Categories,
	)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.GroupID, dto.CategoryGroupID, dto.Priority, dto.IsActive, dto.RealmID)
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row rowScanner, category *models.Category) error {
	var cgID *uuid.UUID
	var cgName *string
	if err := row.Scan(
		&category.ID, &category.Name, &category.Description, &category.GroupID, &category.CategoryGroupID, &category.Priority,
		&category.IsActive, &category.RealmID, &category.CreatedAt, &category.UpdatedAt,
		&cgID, &cgName,
	); err != nil {
		return fmt.Errorf("scan row error: %w", err)
	}
	if cgID != nil {
		category.CategoryGroup = &models.CategoryGroupShort{ID: *cgID, Name: *cgName}
	}
	return nil
}
