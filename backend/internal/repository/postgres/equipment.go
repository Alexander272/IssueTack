package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EquipmentRepo struct {
	db *pgxpool.Pool
}

func NewEquipmentRepo(db *pgxpool.Pool) *EquipmentRepo {
	return &EquipmentRepo{db: db}
}

// Equipment — репозиторий оборудования: справочник типов и сами единицы.
type Equipment interface {
	// Справочник типов
	GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error)
	GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error)
	CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error
	DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error
	CountByType(ctx context.Context, realmID, typeID uuid.UUID) (int, error)

	// Единицы
	Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error)
	GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error)
	// GetChildren возвращает непосредственных потомков узла (состав комплекта).
	GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error)
	// GetAncestors возвращает цепочку предков узла (без самого узла) — для проверки циклов.
	GetAncestors(ctx context.Context, realmID, id uuid.UUID) ([]uuid.UUID, error)
	Create(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	Delete(ctx context.Context, dto *models.DelEquipmentDTO) error
}

// equipmentSelectBase — выборка единицы со справочными полями (тип, площадка,
// ФИО держателя) и COUNT(*) OVER() для тотала страницы.
func equipmentSelectBase() string {
	return fmt.Sprintf(`SELECT 
			e.id, e.realm_id, e.type_id, et.name, et.is_component, e.name, e.vendor, e.model,
			e.serial_number, e.inventory_number, e.specs, e.status, e.parent_id, e.site_id, s.name,
			e.holder_user_id, e.holder_name,
			COALESCE(NULLIF(TRIM(u.last_name || ' ' || u.first_name), ''), u.username),
			e.location, e.purchase_date::text, e.warranty_until::text, e.cost::float8, e.notes,
			e.created_by, e.created_at, e.updated_at,
			COUNT(*) OVER() AS total_count
		FROM %s e
		JOIN %s et ON et.id = e.type_id
		LEFT JOIN %s s ON s.id = e.site_id
		LEFT JOIN %s u ON u.id = e.holder_user_id`,
		Tables.Equipment, Tables.EquipmentTypes, Tables.Sites, Tables.Users)
}

// scanEquipmentRow читает одну строку выдачи equipmentSelectBase.
func scanEquipmentRow(row rowScanner) (*models.Equipment, int, error) {
	item := &models.Equipment{}
	var siteName *string
	var holderFullName *string
	var total int

	if err := row.Scan(
		&item.ID, &item.RealmID, &item.TypeID, &item.TypeName, &item.IsComponent,
		&item.Name, &item.Vendor, &item.Model,
		&item.SerialNumber, &item.InventoryNumber, &item.Specs, &item.Status,
		&item.ParentID, &item.SiteID, &siteName,
		&item.HolderUserID, &item.HolderName, &holderFullName,
		&item.Location, &item.PurchaseDate, &item.WarrantyUntil, &item.Cost, &item.Notes,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &total,
	); err != nil {
		return nil, 0, err
	}
	if siteName != nil {
		item.SiteName = *siteName
	}
	if holderFullName != nil {
		item.HolderFullName = *holderFullName
	}
	if len(item.Specs) == 0 {
		item.Specs = []byte("{}")
	}
	return item, total, nil
}

func (r *EquipmentRepo) GetTypes(ctx context.Context, req *models.GetEquipmentTypesDTO) ([]*models.EquipmentType, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, is_component, sort_order, created_at, updated_at
		FROM %s WHERE realm_id = $1 ORDER BY sort_order, name`, Tables.EquipmentTypes)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentType{}
	for rows.Next() {
		item := &models.EquipmentType{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.IsComponent,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *EquipmentRepo) GetTypeByID(ctx context.Context, req *models.GetEquipmentTypeByIdDTO) (*models.EquipmentType, error) {
	query := fmt.Sprintf(`SELECT id, realm_id, name, is_component, sort_order, created_at, updated_at
		FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentTypes)

	item := &models.EquipmentType{}
	if err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(
		&item.ID, &item.RealmID, &item.Name, &item.IsComponent, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *EquipmentRepo) CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	id := uuid.New()
	dto.ID = &id

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, is_component, sort_order)
		VALUES ($1, $2, $3, $4, $5)`, Tables.EquipmentTypes)

	_, err := r.db.Exec(ctx, query, id, dto.RealmID, dto.Name, dto.IsComponent, dto.SortOrder)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, is_component=$3, sort_order=$4, updated_at=NOW()
		WHERE id=$1 AND realm_id=$5`, Tables.EquipmentTypes)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.IsComponent, dto.SortOrder, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *EquipmentRepo) DeleteType(ctx context.Context, dto *models.DelEquipmentTypeDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentTypes)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// CountByType возвращает число единиц типа в реалме (для гарда удаления типа).
func (r *EquipmentRepo) CountByType(ctx context.Context, realmID, typeID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE realm_id = $1 AND type_id = $2`, Tables.Equipment)

	var count int
	if err := r.db.QueryRow(ctx, query, realmID, typeID).Scan(&count); err != nil {
		return 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return count, nil
}

func (r *EquipmentRepo) Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error) {
	w := &whereBuilder{}
	eq(w, "e.realm_id", &filter.RealmID)
	eq(w, "e.type_id", filter.TypeID)
	eq(w, "e.status", filter.Status)
	eq(w, "e.site_id", filter.SiteID)
	eq(w, "e.holder_user_id", filter.HolderID)

	like := func(column string, val *string) {
		if val == nil || *val == "" {
			return
		}
		w.idx++
		w.add(fmt.Sprintf("%s ILIKE $%d", column, w.idx))
		w.args = append(w.args, "%"+*val+"%")
	}
	like("e.serial_number", filter.Serial)
	like("e.inventory_number", filter.Inventory)

	if filter.Search != nil && *filter.Search != "" {
		pattern := "%" + *filter.Search + "%"
		w.idx++
		w.add(fmt.Sprintf(`(e.name ILIKE $%d OR e.vendor ILIKE $%d OR e.model ILIKE $%d
			OR e.serial_number ILIKE $%d OR e.inventory_number ILIKE $%d)`, w.idx, w.idx+1, w.idx+2, w.idx+3, w.idx+4))
		w.args = append(w.args, pattern, pattern, pattern, pattern, pattern)
		w.idx += 4
	}

	query := equipmentSelectBase()
	if len(w.clauses) > 0 {
		query += " WHERE " + strings.Join(w.clauses, " AND ")
	}
	query += " ORDER BY e.created_at DESC"

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

	data := []*models.Equipment{}
	total := 0
	for rows.Next() {
		item, rowTotal, err := scanEquipmentRow(rows)
		if err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		total = rowTotal
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, total, nil
}

func (r *EquipmentRepo) GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	query := equipmentSelectBase() + ` WHERE e.id = $1 AND e.realm_id = $2`

	row := r.db.QueryRow(ctx, query, req.ID, req.RealmID)
	item, _, err := scanEquipmentRow(row)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *EquipmentRepo) GetChildren(ctx context.Context, realmID, parentID uuid.UUID) ([]*models.Equipment, error) {
	query := equipmentSelectBase() + ` WHERE e.parent_id = $1 AND e.realm_id = $2 ORDER BY e.name`

	rows, err := r.db.Query(ctx, query, parentID, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.Equipment{}
	for rows.Next() {
		item, _, err := scanEquipmentRow(rows)
		if err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// GetAncestors возвращает цепочку предков узла (родитель, родитель родителя, …).
func (r *EquipmentRepo) GetAncestors(ctx context.Context, realmID, id uuid.UUID) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		WITH RECURSIVE chain AS (
			SELECT parent_id FROM %s WHERE id = $2 AND realm_id = $1
			UNION ALL
			SELECT e.parent_id FROM %s e JOIN chain c ON e.id = c.parent_id WHERE e.realm_id = $1
		)
		SELECT parent_id FROM chain WHERE parent_id IS NOT NULL`, Tables.Equipment, Tables.Equipment)

	rows, err := r.db.Query(ctx, query, realmID, id)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []uuid.UUID{}
	for rows.Next() {
		var pid uuid.UUID
		if err := rows.Scan(&pid); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

func (r *EquipmentRepo) Create(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	id := uuid.New()
	dto.ID = &id

	specs := dto.Specs
	if len(specs) == 0 {
		specs = []byte("{}")
	}

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, type_id, name, vendor, model, serial_number, inventory_number,
			specs, status, parent_id, site_id, holder_user_id, holder_name, location, purchase_date, warranty_until, cost, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9::jsonb, $10, $11, $12, $13, NULLIF($14, ''), NULLIF($15, ''), $16::date, $17::date, $18::numeric, $19, $20)`,
		Tables.Equipment)

	_, err := r.db.Exec(ctx, query,
		id, dto.RealmID, dto.TypeID, dto.Name, dto.Vendor, dto.Model,
		dto.SerialNumber, dto.InventoryNumber, specs, dto.Status, dto.ParentID, dto.SiteID,
		dto.HolderUserID, dto.HolderName, dto.Location, dto.PurchaseDate, dto.WarrantyUntil,
		dto.Cost, dto.Notes, dto.CreatedBy)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return r.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: id, RealmID: dto.RealmID})
}

func (r *EquipmentRepo) Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error) {
	specs := dto.Specs
	if len(specs) == 0 {
		specs = []byte("{}")
	}

	query := fmt.Sprintf(`UPDATE %s SET type_id=$2, name=$3, vendor=$4, model=$5, serial_number=$6, inventory_number=$7,
			specs=$8::jsonb, status=$9, parent_id=$10, site_id=$11, holder_user_id=$12, holder_name=$13, location=$14,
			purchase_date=$15::date, warranty_until=$16::date, cost=$17::numeric, notes=$18, updated_at=NOW()
		WHERE id=$1 AND realm_id=$19`, Tables.Equipment)

	res, err := r.db.Exec(ctx, query,
		dto.ID, dto.TypeID, dto.Name, dto.Vendor, dto.Model,
		dto.SerialNumber, dto.InventoryNumber, specs, dto.Status, dto.ParentID, dto.SiteID,
		dto.HolderUserID, dto.HolderName, dto.Location, dto.PurchaseDate, dto.WarrantyUntil,
		dto.Cost, dto.Notes, dto.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return nil, models.ErrNotFound
	}
	return r.GetByID(ctx, &models.GetEquipmentByIdDTO{ID: *dto.ID, RealmID: dto.RealmID})
}

func (r *EquipmentRepo) Delete(ctx context.Context, dto *models.DelEquipmentDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`, Tables.Equipment)

	_, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}