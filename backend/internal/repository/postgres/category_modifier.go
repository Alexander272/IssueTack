package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryModifierRepo struct {
	db *pgxpool.Pool
}

func NewCategoryModifierRepo(db *pgxpool.Pool) *CategoryModifierRepo {
	return &CategoryModifierRepo{db: db}
}

type CategoryModifiers interface {
	Get(ctx context.Context, req *models.GetCategoryModifiersDTO) ([]*models.CategoryModifier, error)
	GetByID(ctx context.Context, req *models.GetCategoryModifierByIDDTO) (*models.CategoryModifier, error)
	Create(ctx context.Context, dto *models.CategoryModifierDTO) error
	Update(ctx context.Context, dto *models.CategoryModifierDTO) error
	Delete(ctx context.Context, dto *models.DelCategoryModifierDTO) error
	// CountTickets считает заявки со ссылкой на подкатегорию — удаление
	// используемой подкатегории запрещено.
	CountTickets(ctx context.Context, id uuid.UUID) (int, error)
	// CountByCategory считает подкатегории категории — удаление категории с
	// подкатегориями, на которые ссылаются заявки, запрещено.
	CountTicketsByCategory(ctx context.Context, categoryID uuid.UUID) (int, error)
}

const categoryModifierColumns = `cm.id, cm.realm_id, cm.category_id, cm.name, cm.description,
	cm.default_due_days, cm.priority, cm.is_active, cm.sort_order, cm.created_at, cm.updated_at`

// priorityArg — nullable-приоритет для запроса. Приводим к *string явно, как
// и при чтении: enum ticket_priority надёжнее передавать как текст, а не как
// указатель на именованный тип модели.
func priorityArg(p *models.Priority) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func scanCategoryModifier(row rowScanner, item *models.CategoryModifier) error {
	var priority *string
	if err := row.Scan(
		&item.ID, &item.RealmID, &item.CategoryID, &item.Name, &item.Description,
		&item.DefaultDueDays, &priority, &item.IsActive, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return fmt.Errorf("scan row error: %w", err)
	}
	if priority != nil {
		p := models.Priority(*priority)
		item.Priority = &p
	}
	return nil
}

// loadCategoryModifiers загружает подкатегории реалма одним запросом — без N+1
// на категории. Используется и отдельным репозиторием, и CategoryRepo, чтобы
// GET /categories сразу отдавал подкатегории вложенно.
func loadCategoryModifiers(ctx context.Context, db *pgxpool.Pool, realmID uuid.UUID) ([]*models.CategoryModifier, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s cm WHERE cm.realm_id = $1 ORDER BY cm.category_id, cm.sort_order, cm.name`,
		categoryModifierColumns, Tables.CategoryModifiers,
	)
	rows, err := db.Query(ctx, query, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.CategoryModifier{}
	for rows.Next() {
		item := &models.CategoryModifier{}
		if err := scanCategoryModifier(rows, item); err != nil {
			return nil, MapError(err)
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func attachCategoryModifiers(categories []*models.Category, modifiers []*models.CategoryModifier) {
	byCategory := make(map[uuid.UUID][]*models.CategoryModifier, len(modifiers))
	for _, m := range modifiers {
		byCategory[m.CategoryID] = append(byCategory[m.CategoryID], m)
	}
	for _, category := range categories {
		if items := byCategory[category.ID]; len(items) > 0 {
			category.Modifiers = items
		}
	}
}

func (r *CategoryModifierRepo) Get(ctx context.Context, req *models.GetCategoryModifiersDTO) ([]*models.CategoryModifier, error) {
	data, err := loadCategoryModifiers(ctx, r.db, req.RealmID)
	if err != nil {
		return nil, err
	}
	if req.CategoryID == nil {
		return data, nil
	}
	filtered := make([]*models.CategoryModifier, 0, len(data))
	for _, item := range data {
		if item.CategoryID == *req.CategoryID {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (r *CategoryModifierRepo) GetByID(ctx context.Context, req *models.GetCategoryModifierByIDDTO) (*models.CategoryModifier, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s cm WHERE cm.id = $1 AND cm.realm_id = $2`,
		categoryModifierColumns, Tables.CategoryModifiers,
	)
	item := &models.CategoryModifier{}
	if err := scanCategoryModifier(r.db.QueryRow(ctx, query, req.ID, req.RealmID), item); err != nil {
		return nil, MapError(err)
	}
	return item, nil
}

func (r *CategoryModifierRepo) Create(ctx context.Context, dto *models.CategoryModifierDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, category_id, name, description, default_due_days, priority, is_active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE((SELECT MAX(sort_order) + 1 FROM %s WHERE category_id = $3), 1))`,
		Tables.CategoryModifiers, Tables.CategoryModifiers,
	)
	id := uuid.New()
	dto.ID = &id

	_, err := r.db.Exec(ctx, query, id, dto.RealmID, dto.CategoryID, dto.Name, dto.Description, dto.DefaultDueDays, priorityArg(dto.Priority), dto.IsActive)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryModifierRepo) Update(ctx context.Context, dto *models.CategoryModifierDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, description=$3, default_due_days=$4, priority=$5, is_active=$6
		WHERE id=$1 AND realm_id=$7`, Tables.CategoryModifiers)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Description, dto.DefaultDueDays, priorityArg(dto.Priority), dto.IsActive, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *CategoryModifierRepo) Delete(ctx context.Context, dto *models.DelCategoryModifierDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.CategoryModifiers)
	if _, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *CategoryModifierRepo) CountTickets(ctx context.Context, id uuid.UUID) (int, error) {
	var count int
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE category_modifier_id = $1`, Tables.Tickets)
	if err := r.db.QueryRow(ctx, query, id).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to count tickets: %w", err))
	}
	return count, nil
}

func (r *CategoryModifierRepo) CountTicketsByCategory(ctx context.Context, categoryID uuid.UUID) (int, error) {
	var count int
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s t JOIN %s cm ON t.category_modifier_id = cm.id WHERE cm.category_id = $1`,
		Tables.Tickets, Tables.CategoryModifiers,
	)
	if err := r.db.QueryRow(ctx, query, categoryID).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to count tickets: %w", err))
	}
	return count, nil
}
