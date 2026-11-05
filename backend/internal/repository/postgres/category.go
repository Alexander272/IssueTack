package postgres

import (
	"context"
	"fmt"
	"strings"

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
	Reorder(ctx context.Context, dto *models.ReorderCategoriesDTO) error
}

// categorySelect возвращает список категорий с присоединённым разделом
// (category_groups). Раздел NULL, если категория не привязана ни к одному.
func (r *CategoryRepo) categorySelect(where string) string {
	return fmt.Sprintf(`SELECT c.id, c.name, c.description, c.group_id, c.category_group_id, c.def_priority, c.default_due_days, c.is_active, c.is_hidden, c.sort_order, c.realm_id, c.created_at, c.updated_at,
			cg.id, cg.name, cg.is_hidden
		FROM %s c
		LEFT JOIN %s cg ON c.category_group_id = cg.id
		WHERE %s`,
		Tables.Categories, Tables.CategoryGroups, where,
	)
}

func (r *CategoryRepo) Get(ctx context.Context, req *models.GetCategoriesDTO) ([]*models.Category, error) {
	where := `c.realm_id = $1 AND c.group_id IS NOT NULL`
	if !req.IncludeHidden {
		// Скрытая категория и категория в скрытом разделе одинаково невидимы
		// безгрупповому заказчику (эффективная скрытость).
		where += ` AND c.is_hidden = FALSE AND COALESCE(cg.is_hidden, FALSE) = FALSE`
	}
	// Порядок глобальный: категории реалма идут как задал админ. Разделы
	// порядком не управляют — блок раздела стоит там, где стоит его первая
	// категория (см. models.Category.SortOrder).
	query := r.categorySelect(where) + ` ORDER BY c.sort_order, c.name`

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
	// Новая категория встаёт в конец глобального порядка — sort_order
	// вычисляется по текущему максимуму реалма.
	query := fmt.Sprintf(`INSERT INTO %s (id, name, description, group_id, category_group_id, def_priority, default_due_days, is_active, is_hidden, realm_id, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM %s WHERE realm_id = $10))`,
		Tables.Categories, Tables.Categories,
	)
	id := uuid.New()
	dto.ID = &id

	_, err := r.db.Exec(ctx, query, id, dto.Name, dto.Description, dto.GroupID, dto.CategoryGroupID, dto.Priority, dto.DefaultDueDays, dto.IsActive, dto.IsHidden, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryRepo) Update(ctx context.Context, dto *models.CategoryDTO) error {
	// realm_id не в SET: категория принадлежит тому реалму, в котором создана,
	// а предикт ниже не даёт переписать (и «перевезти» в свой реалм) чужую.
	// sort_order тоже не в SET: порядок меняется только переупорядочиванием.
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, group_id=$4, category_group_id=$5, def_priority=$6, default_due_days=$7, is_active=$8, is_hidden=$9 WHERE id=$1 AND realm_id=$10`,
		Tables.Categories,
	)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.GroupID, dto.CategoryGroupID, dto.Priority, dto.DefaultDueDays, dto.IsActive, dto.IsHidden, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}
	return nil
}

// Reorder перезаписывает глобальный порядок категорий реалма: категории из
// dto.CategoryIDs получают позиции 1..N. Порядок — не содержимое карточки,
// поэтому updated_at не трогаем.
func (r *CategoryRepo) Reorder(ctx context.Context, dto *models.ReorderCategoriesDTO) error {
	if len(dto.CategoryIDs) == 0 {
		return nil
	}

	values := make([]string, 0, len(dto.CategoryIDs))
	args := make([]any, 0, len(dto.CategoryIDs)+1)
	args = append(args, dto.RealmID)
	for i, id := range dto.CategoryIDs {
		values = append(values, fmt.Sprintf("($%d::uuid, %d)", i+2, i+1))
		args = append(args, id)
	}

	query := fmt.Sprintf(`UPDATE %s c SET sort_order = v.ord FROM (VALUES %s) AS v(id, ord) WHERE c.id = v.id AND c.realm_id = $1`,
		Tables.Categories, strings.Join(values, ", "),
	)

	if _, err := r.db.Exec(ctx, query, args...); err != nil {
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
	var cgHidden *bool
	if err := row.Scan(
		&category.ID, &category.Name, &category.Description, &category.GroupID, &category.CategoryGroupID, &category.Priority,
		&category.DefaultDueDays, &category.IsActive, &category.IsHidden, &category.SortOrder, &category.RealmID, &category.CreatedAt, &category.UpdatedAt,
		&cgID, &cgName, &cgHidden,
	); err != nil {
		return fmt.Errorf("scan row error: %w", err)
	}
	if cgID != nil {
		group := &models.CategoryGroupShort{ID: *cgID, Name: *cgName}
		if cgHidden != nil {
			group.IsHidden = *cgHidden
		}
		category.CategoryGroup = group
	}
	return nil
}
