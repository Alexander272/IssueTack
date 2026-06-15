package postgres

import (
	"context"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecklistRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewChecklistRepo(db *pgxpool.Pool, tr Transaction) *ChecklistRepo {
	return &ChecklistRepo{
		db:          db,
		Transaction: tr,
	}
}

type Checklists interface {
	Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error)
	GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO) (*models.ChecklistTemplate, error)
	Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error
	Update(ctx context.Context, dto *models.ChecklistTemplateDTO) error
	Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO) error
	GetItems(ctx context.Context, templateID uuid.UUID) ([]*models.ChecklistTemplateItem, error)
	SetItems(ctx context.Context, tx Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO) error
}

func (r *ChecklistRepo) Get(ctx context.Context, req *models.GetChecklistTemplatesDTO) ([]*models.ChecklistTemplate, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, title, description, created_at, updated_at FROM %s WHERE realm_id = $1 ORDER BY title`,
		Tables.ChecklistTemplates,
	)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.ChecklistTemplate
	for rows.Next() {
		item := &models.ChecklistTemplate{}
		if err := rows.Scan(
			&item.ID, &item.RealmID, &item.Title, &item.Description,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.ChecklistTemplate{}, nil
	}
	return data, nil
}

func (r *ChecklistRepo) GetByID(ctx context.Context, req *models.GetChecklistTemplateDTO) (*models.ChecklistTemplate, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, title, description, created_at, updated_at FROM %s WHERE id = $1`,
		Tables.ChecklistTemplates,
	)

	item := &models.ChecklistTemplate{}
	if err := r.db.QueryRow(ctx, query, req.ID).Scan(
		&item.ID, &item.RealmID, &item.Title, &item.Description,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *ChecklistRepo) GetItems(ctx context.Context, templateID uuid.UUID) ([]*models.ChecklistTemplateItem, error) {
	query := fmt.Sprintf(`SELECT id, template_id, title, description, sort_order FROM %s WHERE template_id = $1 ORDER BY sort_order`,
		Tables.ChecklistTemplateItems,
	)

	rows, err := r.db.Query(ctx, query, templateID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	var data []*models.ChecklistTemplateItem
	for rows.Next() {
		item := &models.ChecklistTemplateItem{}
		if err := rows.Scan(&item.ID, &item.TemplateID, &item.Title, &item.Description, &item.SortOrder); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	if data == nil {
		return []*models.ChecklistTemplateItem{}, nil
	}
	return data, nil
}

func (r *ChecklistRepo) Create(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, title, description) VALUES ($1, $2, $3, $4)`,
		Tables.ChecklistTemplates,
	)
	if dto.ID == uuid.Nil {
		dto.ID = uuid.New()
	}

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID, dto.Title, dto.Description)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *ChecklistRepo) Update(ctx context.Context, dto *models.ChecklistTemplateDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET title=$2, description=$3, updated_at=NOW() WHERE id=$1`,
		Tables.ChecklistTemplates,
	)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.Title, dto.Description)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *ChecklistRepo) Delete(ctx context.Context, dto *models.DelChecklistTemplateDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, Tables.ChecklistTemplates)

	_, err := r.db.Exec(ctx, query, dto.ID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *ChecklistRepo) SetItems(ctx context.Context, tx Tx, templateID uuid.UUID, items []*models.ChecklistTemplateItemDTO) error {
	if _, err := r.getExec(tx).Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE template_id = $1`, Tables.ChecklistTemplateItems), templateID); err != nil {
		return MapError(fmt.Errorf("failed to delete items: %w", err))
	}

	if len(items) == 0 {
		return nil
	}

	rows := make([][]interface{}, len(items))
	for i, v := range items {
		if v.ID == uuid.Nil {
			v.ID = uuid.New()
		}
		rows[i] = []interface{}{v.ID, templateID, v.Title, v.Description, v.SortOrder}
	}

	columns := []string{"id", "template_id", "title", "description", "sort_order"}
	_, err := r.getExec(tx).CopyFrom(
		ctx,
		pgx.Identifier{Tables.ChecklistTemplateItems},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to insert items: %w", err))
	}
	return nil
}
