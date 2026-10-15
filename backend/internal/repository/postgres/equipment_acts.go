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

// actColumns — колонки акта для SELECT и порядок сканирования. Список один на
// список и карточку: два литерала разъезжались бы при каждой новой колонке.
const actColumns = `id, realm_id, number, status, kit_equipment_id, recipient_user_id, recipient_name, site_id,
	location, issued_by, issued_at, content, comment, signed_file_name, signed_file_path, signed_mime_type,
	signed_file_size, signed_at, signed_by, created_at`

func scanAct(row rowScanner) (*models.EquipmentAct, error) {
	a := &models.EquipmentAct{}
	if err := row.Scan(&a.ID, &a.RealmID, &a.Number, &a.Status, &a.KitEquipmentID, &a.RecipientUserID, &a.RecipientName,
		&a.SiteID, &a.Location, &a.IssuedBy, &a.IssuedAt, &a.Content, &a.Comment, &a.SignedFileName, &a.SignedFilePath,
		&a.SignedMimeType, &a.SignedFileSize, &a.SignedAt, &a.SignedBy, &a.CreatedAt); err != nil {
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
	if filter.EquipmentID != nil {
		// Единица может быть комплектом акта (kit_equipment_id) либо позицией его
		// состава (снимок content). Условия объединены через OR, поэтому номера
		// плейсхолдеров свои у каждого слагаемого.
		w.idx++
		equipment := w.idx
		w.idx++
		content := w.idx
		w.add(fmt.Sprintf("(kit_equipment_id = $%d OR content @> $%d::jsonb)", equipment, content))
		w.args = append(w.args, *filter.EquipmentID,
			fmt.Sprintf(`[{"equipmentId": %q}]`, filter.EquipmentID.String()))
	}

	query := fmt.Sprintf("SELECT "+actColumns+", COUNT(*) OVER() AS total_count FROM %s", Tables.EquipmentActs)
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
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Number, &item.Status, &item.KitEquipmentID, &item.RecipientUserID,
			&item.RecipientName, &item.SiteID, &item.Location, &item.IssuedBy, &item.IssuedAt, &item.Content, &item.Comment,
			&item.SignedFileName, &item.SignedFilePath, &item.SignedMimeType, &item.SignedFileSize, &item.SignedAt,
			&item.SignedBy, &item.CreatedAt, &total); err != nil {
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
	query := fmt.Sprintf("SELECT "+actColumns+" FROM %s WHERE id = $1 AND realm_id = $2", Tables.EquipmentActs)

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
	if a.Status == "" {
		a.Status = models.ActStatusFormed
	}
	content := a.Content
	if len(content) == 0 {
		content = []byte("[]")
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, number, status, kit_equipment_id, recipient_user_id, recipient_name, site_id, location, issued_by, issued_at, content, comment)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, NULLIF($9, ''), $10, $11, $12::jsonb, $13)`, Tables.EquipmentActs)

	_, err := r.getExec(tx).Exec(ctx, query, a.ID, a.RealmID, a.Number, a.Status, a.KitEquipmentID, a.RecipientUserID,
		a.RecipientName, a.SiteID, a.Location, a.IssuedBy, a.IssuedAt, content, a.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// SetActSignedFile переводит акт в статус «подписан» и записывает реквизиты
// файла. Повторная загрузка заменяет файл: путь прошлого возвращает сервис —
// он читал акт до транзакции, а удаляет прежний файл уже после коммита.
func (r *EquipmentRepo) SetActSignedFile(ctx context.Context, tx Tx, dto *models.SignEquipmentActDTO) error {
	var signedBy *uuid.UUID
	if dto.Actor != nil {
		signedBy = &dto.Actor.ID
	}

	query := fmt.Sprintf(`UPDATE %s SET status = $3, signed_file_name = $4, signed_file_path = $5,
			signed_mime_type = $6, signed_file_size = $7, signed_at = NOW(), signed_by = $8
		WHERE id = $1 AND realm_id = $2`, Tables.EquipmentActs)

	tag, err := r.getExec(tx).Exec(ctx, query, dto.ID, dto.RealmID, models.ActStatusSigned, dto.FileName, dto.FilePath,
		dto.MimeType, dto.FileSize, signedBy)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
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
