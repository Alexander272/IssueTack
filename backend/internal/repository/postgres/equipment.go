package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EquipmentRepo struct {
	db *pgxpool.Pool
	Transaction
}

func NewEquipmentRepo(db *pgxpool.Pool, tr Transaction) *EquipmentRepo {
	return &EquipmentRepo{db: db, Transaction: tr}
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
	// GetTree возвращает один уровень дерева оборудования: корни реалма
	// (ParentID == nil) либо непосредственных детей узла (физсостав + члены
	// комплекта). Узлы несут HasChildren/ChildCount, чтобы клиент рисовал шеврон
	// и счётчик без загрузки поддерева. Возвращает (узлы, всего на уровне).
	GetTree(ctx context.Context, dto *models.GetEquipmentTreeDTO) ([]*models.EquipmentTreeNode, int, error)
	// GetStats возвращает счётчики оборудования реалма для шапки страницы.
	GetStats(ctx context.Context, realmID uuid.UUID) (*models.EquipmentStats, error)
	// GetAncestors возвращает цепочку предков узла (без самого узла) — для проверки циклов.
	GetAncestors(ctx context.Context, realmID, id uuid.UUID) ([]uuid.UUID, error)
	Create(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	Update(ctx context.Context, dto *models.EquipmentDTO) (*models.Equipment, error)
	Delete(ctx context.Context, dto *models.DelEquipmentDTO) error
	// SetHolderAndStatus обновляет денормализованного держателя/локацию и статус единицы.
	SetHolderAndStatus(ctx context.Context, tx Tx, id uuid.UUID, userID *uuid.UUID, holderName *string, siteID *uuid.UUID, location *string, status models.EquipmentStatus) error
	// SetStatus меняет только статус единицы.
	SetStatus(ctx context.Context, tx Tx, realmID, id uuid.UUID, status models.EquipmentStatus) error

	// Выдачи
	GetOpenAssignment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error)
	GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error)
	CreateAssignment(ctx context.Context, tx Tx, a *models.EquipmentAssignment) error
	CloseAssignment(ctx context.Context, tx Tx, id uuid.UUID, at time.Time, comment string) error

	// События жизненного цикла
	CreateEvent(ctx context.Context, tx Tx, e *models.EquipmentEvent) error
	GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error)

	// Состав логического комплекта (kit_compositions)
	GetOpenComposition(ctx context.Context, kitID uuid.UUID) ([]*models.Equipment, error)
	GetCompositionHistory(ctx context.Context, kitID uuid.UUID) ([]*models.KitComposition, error)
	GetOpenByComponent(ctx context.Context, componentID uuid.UUID) (*models.KitComposition, error)
	OpenComposition(ctx context.Context, tx Tx, c *models.KitComposition) error
	CloseComposition(ctx context.Context, tx Tx, c *models.KitComposition) error
	CloseOpenByComponent(ctx context.Context, tx Tx, componentID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error
	CloseOpenByKit(ctx context.Context, tx Tx, kitID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error

	// Акты
	GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error)
	GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error)
	CreateAct(ctx context.Context, tx Tx, a *models.EquipmentAct) error
	NextActSeq(ctx context.Context, tx Tx, realmID uuid.UUID) (int, error)
}

// equipmentSelectBase — выборка единицы со справочными полями (тип, площадка,
// ФИО держателя), именем контейнера (физический родитель или логический комплект)
// и COUNT(*) OVER() для тотала страницы.
func equipmentSelectBase() string {
	return fmt.Sprintf(`SELECT 
			e.id, e.realm_id, e.type_id, et.name, et.kind, e.name, e.vendor, e.model,
			e.serial_number, e.inventory_number, e.specs, e.status, e.parent_id, e.site_id, s.name,
			e.holder_user_id, e.holder_name,
			COALESCE(NULLIF(TRIM(u.last_name || ' ' || u.first_name), ''), u.username),
			e.location, e.purchase_date::text, e.warranty_until::text, e.cost::float8, e.notes,
			e.is_kit, e.kit_id, e.created_by, e.created_at, e.updated_at,
			COALESCE(
				(SELECT p.name FROM %s p WHERE p.id = e.parent_id),
				(SELECT k.name FROM %s k WHERE k.id = e.kit_id)
			) AS container_name,
			COUNT(*) OVER() AS total_count
		FROM %s e
		JOIN %s et ON et.id = e.type_id
		LEFT JOIN %s s ON s.id = e.site_id
		LEFT JOIN %s u ON u.id = e.holder_user_id`,
		Tables.Equipment, Tables.Equipment, Tables.Equipment, Tables.EquipmentTypes, Tables.Sites, Tables.Users)
}

// scanEquipmentRow читает одну строку выдачи equipmentSelectBase.
func scanEquipmentRow(row rowScanner) (*models.Equipment, int, error) {
	item := &models.Equipment{}
	var siteName *string
	var holderFullName *string
	var total int

	if err := row.Scan(
		&item.ID, &item.RealmID, &item.TypeID, &item.TypeName, &item.TypeKind,
		&item.Name, &item.Vendor, &item.Model,
		&item.SerialNumber, &item.InventoryNumber, &item.Specs, &item.Status,
		&item.ParentID, &item.SiteID, &siteName,
		&item.HolderUserID, &item.HolderName, &holderFullName,
		&item.Location, &item.PurchaseDate, &item.WarrantyUntil, &item.Cost, &item.Notes,
		&item.IsKit, &item.KitID, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.ContainerName, &total,
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
	query := fmt.Sprintf(`SELECT id, realm_id, name, kind, sort_order, created_at, updated_at
		FROM %s WHERE realm_id = $1 ORDER BY sort_order, name`, Tables.EquipmentTypes)

	rows, err := r.db.Query(ctx, query, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentType{}
	for rows.Next() {
		item := &models.EquipmentType{}
		if err := rows.Scan(&item.ID, &item.RealmID, &item.Name, &item.Kind,
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
	query := fmt.Sprintf(`SELECT id, realm_id, name, kind, sort_order, created_at, updated_at
		FROM %s WHERE id = $1 AND realm_id = $2`, Tables.EquipmentTypes)

	item := &models.EquipmentType{}
	if err := r.db.QueryRow(ctx, query, req.ID, req.RealmID).Scan(
		&item.ID, &item.RealmID, &item.Name, &item.Kind, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

func (r *EquipmentRepo) CreateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	id := uuid.New()
	dto.ID = &id

	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, name, kind, sort_order)
		VALUES ($1, $2, $3, $4, $5)`, Tables.EquipmentTypes)

	_, err := r.db.Exec(ctx, query, id, dto.RealmID, dto.Name, dto.Kind, dto.SortOrder)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) UpdateType(ctx context.Context, dto *models.EquipmentTypeDTO) error {
	query := fmt.Sprintf(`UPDATE %s SET name=$2, kind=$3, sort_order=$4, updated_at=NOW()
		WHERE id=$1 AND realm_id=$5`, Tables.EquipmentTypes)

	res, err := r.db.Exec(ctx, query, dto.ID, dto.Name, dto.Kind, dto.SortOrder, dto.RealmID)
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
	eq(w, "e.is_kit", filter.IsKit)
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

// GetTree возвращает один уровень дерева оборудования реалма. Дерево грузится
// лениво: при ParentID == nil это корни (нет физического родителя в реалме и нет
// открытого членства в комплекте), иначе — непосредственные дети узла: физический
// состав (parent_id) плюс текущие члены логического комплекта (kit_compositions).
// У члена комплекта с физическим родителем приоритет у родителя — как в прежней
// сборке дерева. Узлы несут HasChildren/ChildCount, а не только своё имя.
func (r *EquipmentRepo) GetTree(ctx context.Context, dto *models.GetEquipmentTreeDTO) ([]*models.EquipmentTreeNode, int, error) {
	w := &whereBuilder{}
	eq(w, "e.realm_id", &dto.RealmID)

	if dto.ParentID == nil {
		// $1 — realmID (первый аргумент eq). Статус сужает только корни: дети
		// раскрытого узла показываются целиком (как было на клиенте).
		eq(w, "e.status", dto.Status)
		w.add(`NOT EXISTS (SELECT 1 FROM ` + Tables.Equipment + ` p WHERE p.id = e.parent_id AND p.realm_id = $1)`)
		w.add(`NOT EXISTS (SELECT 1 FROM ` + Tables.KitCompositions + ` kc WHERE kc.component_id = e.id AND kc.ended_at IS NULL)`)
	} else {
		w.idx++
		parent := w.idx
		w.args = append(w.args, *dto.ParentID)
		w.add(fmt.Sprintf(`(e.parent_id = $%d OR (EXISTS (SELECT 1 FROM %s kc WHERE kc.component_id = e.id AND kc.kit_id = $%d AND kc.ended_at IS NULL)
			AND NOT EXISTS (SELECT 1 FROM %s p WHERE p.id = e.parent_id AND p.realm_id = $1)))`,
			parent, Tables.KitCompositions, parent, Tables.Equipment))
	}

	query := equipmentSelectBase() + " WHERE " + strings.Join(w.clauses, " AND ")
	query += " ORDER BY e.name"

	if dto.Limit > 0 {
		w.idx++
		query += fmt.Sprintf(" LIMIT $%d", w.idx)
		w.args = append(w.args, dto.Limit)
	}
	if dto.Offset > 0 {
		w.idx++
		query += fmt.Sprintf(" OFFSET $%d", w.idx)
		w.args = append(w.args, dto.Offset)
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
	if len(data) == 0 {
		return []*models.EquipmentTreeNode{}, 0, nil
	}

	ids := make([]uuid.UUID, len(data))
	for i, item := range data {
		ids[i] = item.ID
	}
	childCounts, err := r.getChildCounts(ctx, dto.RealmID, ids)
	if err != nil {
		return nil, 0, err
	}

	nodes := make([]*models.EquipmentTreeNode, 0, len(data))
	for _, item := range data {
		count := childCounts[item.ID]
		nodes = append(nodes, &models.EquipmentTreeNode{
			Equipment:   item,
			Children:    []*models.EquipmentTreeNode{},
			HasChildren: count > 0,
			ChildCount:  count,
		})
	}
	return nodes, total, nil
}

// getChildCounts считает детей узлов одного уровня: физический состав (parent_id)
// и текущих членов комплекта (kit_compositions). Член комплекта, у которого есть
// физический родитель в реалме, не считается — он показывается внутри родителя.
func (r *EquipmentRepo) getChildCounts(ctx context.Context, realmID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(ids))
	if len(ids) == 0 {
		return counts, nil
	}

	physical := fmt.Sprintf(`SELECT c.parent_id, COUNT(*)::int FROM %s c
		WHERE c.realm_id = $2 AND c.parent_id = ANY($1) GROUP BY c.parent_id`, Tables.Equipment)
	members := fmt.Sprintf(`SELECT kc.kit_id, COUNT(*)::int FROM %s kc
		JOIN %s c ON c.id = kc.component_id AND c.realm_id = $2
		WHERE kc.kit_id = ANY($1) AND kc.ended_at IS NULL
			AND NOT EXISTS (SELECT 1 FROM %s p WHERE p.id = c.parent_id AND p.realm_id = $2)
		GROUP BY kc.kit_id`, Tables.KitCompositions, Tables.Equipment, Tables.Equipment)

	for _, query := range []string{physical, members} {
		rows, err := r.db.Query(ctx, query, ids, realmID)
		if err != nil {
			return nil, MapError(fmt.Errorf("failed to count equipment children: %w", err))
		}
		for rows.Next() {
			var id uuid.UUID
			var count int
			if err := rows.Scan(&id, &count); err != nil {
				rows.Close()
				return nil, MapError(fmt.Errorf("scan row error: %w", err))
			}
			counts[id] += count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
		}
		rows.Close()
	}
	return counts, nil
}

// GetStats возвращает счётчики единиц реалма (всего не-комплектов, комплектов и
// комплектующих/расходников на складе) для подписи в шапке страницы.
func (r *EquipmentRepo) GetStats(ctx context.Context, realmID uuid.UUID) (*models.EquipmentStats, error) {
	query := fmt.Sprintf(`SELECT
			COUNT(*) FILTER (WHERE NOT e.is_kit)::int,
			COUNT(*) FILTER (WHERE e.is_kit)::int,
			COUNT(*) FILTER (WHERE NOT e.is_kit AND e.status = 'in_stock' AND et.kind IN ('component', 'consumable'))::int
		FROM %s e
		JOIN %s et ON et.id = e.type_id
		WHERE e.realm_id = $1`, Tables.Equipment, Tables.EquipmentTypes)

	stats := &models.EquipmentStats{}
	if err := r.db.QueryRow(ctx, query, realmID).Scan(&stats.Units, &stats.Kits, &stats.StockParts); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return stats, nil
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
			specs, status, parent_id, site_id, holder_user_id, holder_name, location, purchase_date, warranty_until, cost, notes, created_by, is_kit)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9::jsonb, $10, $11, $12, $13, NULLIF($14, ''), NULLIF($15, ''), $16::date, $17::date, $18::numeric, $19, $20, $21)`,
		Tables.Equipment)

	_, err := r.db.Exec(ctx, query,
		id, dto.RealmID, dto.TypeID, dto.Name, dto.Vendor, dto.Model,
		dto.SerialNumber, dto.InventoryNumber, specs, dto.Status, dto.ParentID, dto.SiteID,
		dto.HolderUserID, dto.HolderName, dto.Location, dto.PurchaseDate, dto.WarrantyUntil,
		dto.Cost, dto.Notes, dto.CreatedBy, dto.IsKit)
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

// SetHolderAndStatus обновляет держателя/локацию (денормализация) и статус.
func (r *EquipmentRepo) SetHolderAndStatus(ctx context.Context, tx Tx, id uuid.UUID, userID *uuid.UUID, holderName *string, siteID *uuid.UUID, location *string, status models.EquipmentStatus) error {
	query := fmt.Sprintf(`UPDATE %s SET holder_user_id = $2, holder_name = NULLIF($3, ''), site_id = $4,
			location = NULLIF($5, ''), status = $6, updated_at = NOW() WHERE id = $1`, Tables.Equipment)

	_, err := r.getExec(tx).Exec(ctx, query, id, userID, holderName, siteID, location, status)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// SetStatus меняет только статус единицы.
func (r *EquipmentRepo) SetStatus(ctx context.Context, tx Tx, realmID, id uuid.UUID, status models.EquipmentStatus) error {
	query := fmt.Sprintf(`UPDATE %s SET status = $3, updated_at = NOW() WHERE id = $1 AND realm_id = $2`, Tables.Equipment)

	res, err := r.getExec(tx).Exec(ctx, query, id, realmID, status)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if res.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// ===== Выдачи =====

func scanAssignment(row rowScanner) (*models.EquipmentAssignment, error) {
	a := &models.EquipmentAssignment{}
	if err := row.Scan(&a.ID, &a.EquipmentID, &a.UserID, &a.HolderName, &a.SiteID, &a.Location,
		&a.AssignedAt, &a.UnassignedAt, &a.AssignedBy, &a.Comment); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *EquipmentRepo) GetOpenAssignment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error) {
	query := fmt.Sprintf(`SELECT id, equipment_id, user_id, holder_name, site_id, location, assigned_at, unassigned_at, assigned_by, comment
		FROM %s WHERE equipment_id = $1 AND unassigned_at IS NULL ORDER BY assigned_at DESC LIMIT 1`, Tables.EquipmentAssignments)

	a, err := scanAssignment(r.db.QueryRow(ctx, query, equipmentID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return a, nil
}

func (r *EquipmentRepo) GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error) {
	query := fmt.Sprintf(`SELECT a.id, a.equipment_id, a.user_id, a.holder_name, a.site_id, a.location,
			a.assigned_at, a.unassigned_at, a.assigned_by, a.comment
		FROM %s a JOIN %s e ON e.id = a.equipment_id
		WHERE a.equipment_id = $1 AND e.realm_id = $2 ORDER BY a.assigned_at DESC`,
		Tables.EquipmentAssignments, Tables.Equipment)

	rows, err := r.db.Query(ctx, query, equipmentID, realmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentAssignment{}
	for rows.Next() {
		item, err := scanAssignment(rows)
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

func (r *EquipmentRepo) CreateAssignment(ctx context.Context, tx Tx, a *models.EquipmentAssignment) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, equipment_id, user_id, holder_name, site_id, location, assigned_at, unassigned_at, assigned_by, comment)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, $8, $9, $10)`, Tables.EquipmentAssignments)

	_, err := r.getExec(tx).Exec(ctx, query, a.ID, a.EquipmentID, a.UserID, a.HolderName, a.SiteID, a.Location,
		a.AssignedAt, a.UnassignedAt, a.AssignedBy, a.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseAssignment(ctx context.Context, tx Tx, id uuid.UUID, at time.Time, comment string) error {
	query := fmt.Sprintf(`UPDATE %s SET unassigned_at = $2, comment = CASE WHEN $3 = '' THEN comment ELSE $3 END WHERE id = $1`, Tables.EquipmentAssignments)

	_, err := r.getExec(tx).Exec(ctx, query, id, at, comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

// ===== События =====

func (r *EquipmentRepo) CreateEvent(ctx context.Context, tx Tx, e *models.EquipmentEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, equipment_id, event_type, operation_id, component_id, old_value, new_value, user_id, comment)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8, $9)`, Tables.EquipmentEvents)

	_, err := r.getExec(tx).Exec(ctx, query, e.ID, e.EquipmentID, e.EventType, e.OperationID, e.ComponentID,
		nullableJSON(e.OldValue), nullableJSON(e.NewValue), e.UserID, e.Comment)
	if err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) GetEvents(ctx context.Context, realmID, equipmentID uuid.UUID, limit, offset int) ([]*models.EquipmentEvent, error) {
	query := fmt.Sprintf(`SELECT ev.id, ev.equipment_id, ev.event_type, ev.operation_id, ev.component_id,
			ev.old_value, ev.new_value, ev.user_id, ev.comment, ev.created_at
		FROM %s ev JOIN %s e ON e.id = ev.equipment_id
		WHERE ev.equipment_id = $1 AND e.realm_id = $2 ORDER BY ev.created_at DESC`,
		Tables.EquipmentEvents, Tables.Equipment)

	args := []any{equipmentID, realmID}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $4"
			args = append(args, offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.EquipmentEvent{}
	for rows.Next() {
		item := &models.EquipmentEvent{}
		if err := rows.Scan(&item.ID, &item.EquipmentID, &item.EventType, &item.OperationID, &item.ComponentID,
			&item.OldValue, &item.NewValue, &item.UserID, &item.Comment, &item.CreatedAt); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// nullableJSON возвращает nil для пустого jsonb, иначе строку.
func nullableJSON(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return string(v)
}

// ===== Состав логического комплекта =====

func (r *EquipmentRepo) GetOpenComposition(ctx context.Context, kitID uuid.UUID) ([]*models.Equipment, error) {
	query := equipmentSelectBase() + fmt.Sprintf(` JOIN %s kc ON kc.component_id = e.id AND kc.ended_at IS NULL
		WHERE kc.kit_id = $1 ORDER BY e.name`, Tables.KitCompositions)

	rows, err := r.db.Query(ctx, query, kitID)
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

func scanComposition(row rowScanner) (*models.KitComposition, error) {
	c := &models.KitComposition{}
	if err := row.Scan(&c.ID, &c.KitID, &c.ComponentID, &c.StartedAt, &c.EndedAt, &c.StartedBy, &c.EndedBy, &c.Reason, &c.ActID); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *EquipmentRepo) GetCompositionHistory(ctx context.Context, kitID uuid.UUID) ([]*models.KitComposition, error) {
	query := fmt.Sprintf(`SELECT id, kit_id, component_id, started_at, ended_at, started_by, ended_by, reason, act_id
		FROM %s WHERE kit_id = $1 ORDER BY started_at DESC`, Tables.KitCompositions)

	rows, err := r.db.Query(ctx, query, kitID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.KitComposition{}
	for rows.Next() {
		item, err := scanComposition(rows)
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

func (r *EquipmentRepo) GetOpenByComponent(ctx context.Context, componentID uuid.UUID) (*models.KitComposition, error) {
	query := fmt.Sprintf(`SELECT id, kit_id, component_id, started_at, ended_at, started_by, ended_by, reason, act_id
		FROM %s WHERE component_id = $1 AND ended_at IS NULL ORDER BY started_at DESC LIMIT 1`, Tables.KitCompositions)

	c, err := scanComposition(r.db.QueryRow(ctx, query, componentID))
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return c, nil
}

func (r *EquipmentRepo) OpenComposition(ctx context.Context, tx Tx, c *models.KitComposition) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	insert := fmt.Sprintf(`INSERT INTO %s (id, kit_id, component_id, started_at, started_by, reason, act_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, insert, c.ID, c.KitID, c.ComponentID, c.StartedAt, c.StartedBy, c.Reason, c.ActID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = $2 WHERE id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, c.ComponentID, c.KitID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseComposition(ctx context.Context, tx Tx, c *models.KitComposition) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = $2, ended_by = $3, reason = CASE WHEN $4 = '' THEN reason ELSE $4 END, act_id = COALESCE($5, act_id)
		WHERE id = $1`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, c.ID, c.EndedAt, c.EndedBy, c.Reason, c.ActID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseOpenByComponent(ctx context.Context, tx Tx, componentID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = NOW(), ended_by = $2, reason = CASE WHEN $3 = '' THEN reason ELSE $3 END, act_id = COALESCE($4, act_id)
		WHERE component_id = $1 AND ended_at IS NULL`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, componentID, endedBy, reason, actID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = NULL WHERE id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, componentID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

func (r *EquipmentRepo) CloseOpenByKit(ctx context.Context, tx Tx, kitID uuid.UUID, endedBy *uuid.UUID, reason string, actID *uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET ended_at = NOW(), ended_by = $2, reason = CASE WHEN $3 = '' THEN reason ELSE $3 END, act_id = COALESCE($4, act_id)
		WHERE kit_id = $1 AND ended_at IS NULL`, Tables.KitCompositions)
	if _, err := r.getExec(tx).Exec(ctx, query, kitID, endedBy, reason, actID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	update := fmt.Sprintf(`UPDATE %s SET kit_id = NULL WHERE kit_id = $1`, Tables.Equipment)
	if _, err := r.getExec(tx).Exec(ctx, update, kitID); err != nil {
		return MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return nil
}

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
