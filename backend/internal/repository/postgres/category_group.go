package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryGroupRepo struct {
	db *pgxpool.Pool
}

func NewCategoryGroupRepo(db *pgxpool.Pool) *CategoryGroupRepo {
	return &CategoryGroupRepo{
		db: db,
	}
}

type CategoryGroups interface {
	Get(ctx context.Context, req *models.GetCategoryGroupsDTO) ([]*models.CategoryGroup, error)
	GetByID(ctx context.Context, req *models.GetCategoryGroupByIdDTO) (*models.CategoryGroup, error)
	Create(ctx context.Context, dto *models.CategoryGroupDTO) error
	Update(ctx context.Context, dto *models.CategoryGroupDTO) error
	Delete(ctx context.Context, dto *models.DelCategoryGroupDTO) error
	CountByCategory(ctx context.Context, id uuid.UUID) (int, error)
}

func (r *CategoryGroupRepo) Get(ctx context.Context, req *models.GetCategoryGroupsDTO) ([]*models.CategoryGroup, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, description, sort_order, created_at, updated_at FROM %s WHERE realm_id = $1 ORDER BY sort_order, name`,
		Tables.CategoryGroups,
	)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.CategoryGroup{}
	for rows.Next() {
		item := &models.CategoryGroup{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.Description, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

func (r *CategoryGroupRepo) GetByID(ctx context.Context, req *models.GetCategoryGroupByIdDTO) (*models.CategoryGroup, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, description, sort_order, created_at, updated_at FROM %s WHERE id = $1 AND realm_id = $2`,
		Tables.CategoryGroups,
	)

	item := &models.CategoryGroup{}
	if err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(
		&item.ID, &item.RealmID, &item.Name, &item.Description, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *CategoryGroupRepo) Create(ctx context.Context, dto *models.CategoryGroupDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, description, sort_order) VALUES ($1, $2, $3, $4, $5)`,
		Tables.CategoryGroups,
	)
	id := uuid.New()
	dto.ID = &id

	_, err := r.db.Exec(ctx, query, id, dto.RealmID, dto.Name, dto.Description, dto.SortOrder)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryGroupRepo) Update(ctx context.Context, dto *models.CategoryGroupDTO) error {
	// realm_id не в SET — см. CategoryRepo.Update.
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, sort_order=$4 WHERE id=$1 AND realm_id=$5`,
		Tables.CategoryGroups,
	)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.SortOrder, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *CategoryGroupRepo) Delete(ctx context.Context, dto *models.DelCategoryGroupDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.CategoryGroups)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// CountByCategory считает категории раздела — используется сервисом, чтобы не
// дать удалить раздел, в котором ещё есть категории.
func (r *CategoryGroupRepo) CountByCategory(ctx context.Context, id uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE category_group_id = $1`, Tables.Categories)

	var count int
	if err := r.db.QueryRow(ctx, query, id).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return count, nil
}
