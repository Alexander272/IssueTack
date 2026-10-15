package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
)

// ===== Акты =====

func scanAct(row rowScanner) (*models.EquipmentAct, error) {
	a := &models.EquipmentAct{}
	if err := row.Scan(&a.ID, &a.RealmID, &a.Number, &a.KitEquipmentID, &a.RecipientUserID, &a.RecipientName,
		&a.SiteID, &a.Location, &a.IssuedBy, &a.IssuedAt, &a.Content, &a.Comment, &a.CreatedAt); err != nil {
		return nil, err
	}
	if len(a.Content) == 0 {
		a.Content = []byte("[]")
	}
	return a, nil
}

func (r *EquipmentRepo) GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error) {
	w := &whereBuilder{}
	eq(w, "realm_id", &filter.RealmID)
	if filter.KitID != nil {
		w.idx++
		w.add(fmt.Sprintf("kit_equipment_id = $%d", w.idx))
		w.args = append(w.args, *filter.KitID)
	}

	query := fmt.Sprintf(`SELECT id, realm_id, number, kit_equipment_id, recipient_user_id, recipient_name, site_id,
			location, issued_by, issued_at, content, comment, created_at, COUNT(*) OVER() AS total_count
		FROM %s`, Tables.EquipmentActs)
	if len(w.clauses) > 0 {
		query += " WHERE " + strings.Join(w.clauses, " AND ")
	}
	query += " ORDER BY issued_at DESC"

	if filter.Limit > 0 {
		w.idx++
		query += fmt.Sprintf(" LIMIT $%d", w.idx)
		w.args = append(w.args, filter.Limit)
	}
	if filter.Offset > 0 {
		w.idx++
		query += fmt.Sprintf(" OFFSET $%d", w.idx)
		w.args = append(w.args, filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentAct{}
	total := 0
	for rows.Next() {
		item := &models.EquipmentAct{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Number, &item.KitEquipmentID, &item.RecipientUserID, &item.RecipientName,
			&item.SiteID, &item.Location, &item.IssuedBy, &item.IssuedAt, &item.Content, &item.Comment, &item.CreatedAt, &total); err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if len(item.Content) == 0 {
			item.Content = []byte("[]")
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, total, nil
}

func (r *EquipmentRepo) GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, number, kit_equipment_id, recipient_user_id, recipient_name, site_id,
			location, issued_by, issued_at, content, comment, created_at
		FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentActs)

	a, err := scanAct(r.db.QueryRow(ctx, query, req.ID, req.RealmID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return a, nil
}

func (r *EquipmentRepo) CreateAct(ctx context.Context, tx Tx, a *models.EquipmentAct) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	content := a.Content
	if len(content) == 0 {
		content = []byte("[]")
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, number, kit_equipment_id, recipient_user_id, recipient_name, site_id, location, issued_by, issued_at, content, comment)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, NULLIF($8, ''), $9, $10, $11::jsonb, $12)`, Tables.EquipmentActs)

	_, err := r.getExec(tx).Exec(ctx, query, a.ID, a.RealmID, a.Number, a.KitEquipmentID, a.RecipientUserID, a.RecipientName,
		a.SiteID, a.Location, a.IssuedBy, a.IssuedAt, content, a.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// NextActSeq возвращает следующий порядковый номер акта в реалме за год.
func (r *EquipmentRepo) NextActSeq(ctx context.Context, tx Tx, realmID uuid.UUID) (int, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("АКТ-%d-%%", year)
	query := fmt.Sprintf(`SELECT COALESCE(MAX(NULLIF(regexp_replace(number, '^.*-', ''), '')::int), 0)
		FROM %s WHERE realm_id = $1 AND number LIKE $2`, Tables.EquipmentActs)

	var seq int
	if err := r.getExec(tx).QueryRow(ctx, query, realmID, prefix).Scan(&seq); err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return seq + 1, nil
}
