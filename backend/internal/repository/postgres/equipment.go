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

	// Справочник магазинов: варианты для поля «Магазин» и upsert нового значения.
	GetShops(ctx context.Context, req *models.GetEquipmentShopsDTO) ([]*models.EquipmentShop, error)
	EnsureShop(ctx context.Context, tx Tx, realmID uuid.UUID, name string) error

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
	// Create вставляет единицу и возвращает её. Событие журнала пишется в той же
	// транзакции (см. сервис), поэтому tx приходит снаружи: nil — работа через пул.
	Create(ctx context.Context, tx Tx, dto *models.EquipmentDTO) (*models.Equipment, error)
	// Update перезаписывает единицу и возвращает её новое состояние.
	Update(ctx context.Context, tx Tx, dto *models.EquipmentDTO) (*models.Equipment, error)
	Delete(ctx context.Context, dto *models.DelEquipmentDTO) error
	// SetHolderAndStatus обновляет денормализованного держателя/локацию и статус единицы.
	SetHolderAndStatus(ctx context.Context, tx Tx, id uuid.UUID, userID *uuid.UUID, holderName *string, siteID *uuid.UUID, location *string, status models.EquipmentStatus) error
	// SetStatus меняет только статус единицы.
	SetStatus(ctx context.Context, tx Tx, realmID, id uuid.UUID, status models.EquipmentStatus) error
	// GetUserFullName и GetSiteName резолвят подписи ссылок для снимка события
	// (журнал иммутабелен, а FK объявлены ON DELETE SET NULL).
	GetUserFullName(ctx context.Context, userID uuid.UUID) (string, error)
	GetSiteName(ctx context.Context, siteID uuid.UUID) (string, error)

	// Выдачи
	GetOpenAssignment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error)
	// GetLastStoredPlace возвращает место последнего периода без держателя — «дом»
	// единицы, куда её ставит возврат.
	GetLastStoredPlace(ctx context.Context, equipmentID uuid.UUID) (*uuid.UUID, *string, error)
	GetAssignmentHistory(ctx context.Context, realmID, equipmentID uuid.UUID) ([]*models.EquipmentAssignment, error)
	// GetDueReturns возвращает открытые выдачи со сроком возврата не позже now:
	// по ним cron шлёт напоминание тому, кто выдал.
	GetDueReturns(ctx context.Context, now time.Time) ([]*models.EquipmentDueReturn, error)
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

	// Шаблоны комплектов
	GetKitTemplates(ctx context.Context, realmID uuid.UUID) ([]*models.EquipmentKitTemplate, error)
	GetKitTemplateByID(ctx context.Context, req *models.GetEquipmentKitTemplateByIdDTO) (*models.EquipmentKitTemplate, error)
	CreateKitTemplate(ctx context.Context, tx Tx, dto *models.EquipmentKitTemplateDTO) error
	UpdateKitTemplate(ctx context.Context, tx Tx, dto *models.EquipmentKitTemplateDTO) error
	DeleteKitTemplate(ctx context.Context, dto *models.DelEquipmentKitTemplateDTO) error

	// Акты
	GetActs(ctx context.Context, filter *models.EquipmentActsFilter) ([]*models.EquipmentAct, int, error)
	GetActByID(ctx context.Context, req *models.GetEquipmentActByIDDTO) (*models.EquipmentAct, error)
	CreateAct(ctx context.Context, tx Tx, a *models.EquipmentAct) error
	// SetActSignedFile помечает акт подписанным и записывает реквизиты файла.
	SetActSignedFile(ctx context.Context, tx Tx, dto *models.SignEquipmentActDTO) error
	NextActSeq(ctx context.Context, tx Tx, realmID uuid.UUID) (int, error)

	// Инвентарные номера
	// NextInventoryNumber возвращает следующее значение счётчика реалма для серии
	// (unit/kit). Вызывается в транзакции создания/правки: откат не «сжигает» номер.
	NextInventoryNumber(ctx context.Context, tx Tx, realmID uuid.UUID, series models.EquipmentInventorySeries) (int, error)
	// InventoryNumberExists сообщает, занят ли номер в реалме: ручной ввод может
	// обогнать счётчик, и сгенерированному значению нельзя занять чужое.
	InventoryNumberExists(ctx context.Context, tx Tx, realmID uuid.UUID, number string) (bool, error)
}

// equipmentSelectBase — выборка единицы со справочными полями (тип, площадка,
// ФИО держателя), именем контейнера (физический родитель или логический комплект)
// и COUNT(*) OVER() для тотала страницы.
func equipmentSelectBase() string {
	return fmt.Sprintf(`SELECT 
			e.id, e.realm_id, e.type_id, COALESCE(et.name, ''), COALESCE(et.kind::text, ''), COALESCE(et.icon, ''), e.name, e.vendor, e.model,
			e.serial_number, e.inventory_number, e.specs, e.shop, e.order_number, e.status, e.parent_id, e.site_id, s.name,
			e.holder_user_id, e.holder_name,
			%s,
			(SELECT a.return_due_at FROM %s a
				WHERE a.equipment_id = e.id AND a.unassigned_at IS NULL
				ORDER BY a.assigned_at DESC LIMIT 1) AS return_due_at,
			e.location, e.purchase_date::text, e.warranty_until::text, e.notes,
			e.is_kit, e.kit_id, e.created_by, e.created_at, e.updated_at,
			COALESCE(
				(SELECT p.name FROM %s p WHERE p.id = e.parent_id),
				(SELECT k.name FROM %s k WHERE k.id = e.kit_id)
			) AS container_name,
			COUNT(*) OVER() AS total_count
		FROM %s e
		LEFT JOIN %s et ON et.id = e.type_id
		LEFT JOIN %s s ON s.id = e.site_id
		LEFT JOIN %s u ON u.id = e.holder_user_id`,
		fullNameSQL("u"),
		Tables.EquipmentAssignments,
		Tables.Equipment, Tables.Equipment, Tables.Equipment, Tables.EquipmentTypes, Tables.Sites, Tables.Users)
}

// scanEquipmentRow читает одну строку выдачи equipmentSelectBase.
func scanEquipmentRow(row rowScanner) (*models.Equipment, int, error) {
	item := &models.Equipment{}
	var siteName *string
	var holderFullName *string
	var total int

	if err := row.Scan(
		&item.ID, &item.RealmID, &item.TypeID, &item.TypeName, &item.TypeKind, &item.TypeIcon,
		&item.Name, &item.Vendor, &item.Model,
		&item.SerialNumber, &item.InventoryNumber, &item.Specs, &item.Shop, &item.OrderNumber, &item.Status,
		&item.ParentID, &item.SiteID, &siteName,
		&item.HolderUserID, &item.HolderName, &holderFullName,
		&item.ReturnDueAt,
		&item.Location, &item.PurchaseDate, &item.WarrantyUntil, &item.Notes,
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

// addEquipmentSearch добавляет подстрочный поиск по реквизитам единицы. Одна
// формула на плоский список (Get) и дерево (GetTree): иначе «найти по номеру
// заказа» работало бы в одном режиме и не работало в другом. Пустая строка —
// условие не добавляется.
func addEquipmentSearch(w *whereBuilder, search *string) {
	if search == nil || *search == "" {
		return
	}
	pattern := "%" + *search + "%"
	w.idx++
	w.add(fmt.Sprintf(`(e.name ILIKE $%d OR e.vendor ILIKE $%d OR e.model ILIKE $%d
		OR e.serial_number ILIKE $%d OR e.inventory_number ILIKE $%d OR e.order_number ILIKE $%d
		OR e.shop ILIKE $%d)`, w.idx, w.idx+1, w.idx+2, w.idx+3, w.idx+4, w.idx+5, w.idx+6))
	w.args = append(w.args, pattern, pattern, pattern, pattern, pattern, pattern, pattern)
	w.idx += 6
}

func (r *EquipmentRepo) Get(ctx context.Context, filter *models.EquipmentFilter) ([]*models.Equipment, int, error) {
	w := &whereBuilder{}
	eq(w, "e.realm_id", &filter.RealmID)
	eq(w, "e.type_id", filter.TypeID)
	eq(w, "e.status", filter.Status)
	eq(w, "e.is_kit", filter.IsKit)
	eq(w, "e.site_id", filter.SiteID)
	eq(w, "e.holder_user_id", filter.HolderID)

	// Списанное — «архив»: в общем списке его нет. Явный фильтр статуса (в т.ч.
	// written_off) или флаг includeWrittenOff возвращают записи. Условие добавляем,
	// только когда статус не задан: при явном статусе фильтр по нему уже есть.
	if filter.Status == nil && !filter.IncludeWrittenOff {
		w.add("e.status <> 'written_off'")
	}

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
		addEquipmentSearch(w, filter.Search)
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
	return r.getByID(ctx, nil, req)
}

// getByID — общая реализация чтения единицы. Внутри транзакции читать нужно через
// неё же (tx), иначе пул не увидит ещё не закоммиченную строку: так Create/Update
// возвращают состояние, записанное в той же транзакции.
func (r *EquipmentRepo) getByID(ctx context.Context, tx Tx, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	query := equipmentSelectBase() + ` WHERE e.id = $1 AND e.realm_id = $2`

	row := r.getExec(tx).QueryRow(ctx, query, req.ID, req.RealmID)
	item, _, err := scanEquipmentRow(row)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return item, nil
}

// GetStats возвращает счётчики единиц реалма (всего не-комплектов, комплектов и
// комплектующих/расходников на складе) для подписи в шапке страницы.
func (r *EquipmentRepo) GetStats(ctx context.Context, realmID uuid.UUID) (*models.EquipmentStats, error) {
	query := fmt.Sprintf(`SELECT
			COUNT(*) FILTER (WHERE NOT e.is_kit)::int,
			COUNT(*) FILTER (WHERE e.is_kit)::int,
			COUNT(*) FILTER (WHERE NOT e.is_kit AND e.status = 'in_stock' AND et.kind IN ('component', 'consumable'))::int
		FROM %s e
		LEFT JOIN %s et ON et.id = e.type_id
		WHERE e.realm_id = $1`, Tables.Equipment, Tables.EquipmentTypes)

	stats := &models.EquipmentStats{}
	if err := r.db.QueryRow(ctx, query, realmID).Scan(&stats.Units, &stats.Kits, &stats.StockParts); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return stats, nil
}

func (r *EquipmentRepo) Create(ctx context.Context, tx Tx, dto *models.EquipmentDTO) (*models.Equipment, error) {
	id := uuid.New()
	dto.ID = &id

	specs := dto.Specs
	if len(specs) == 0 {
		specs = []byte("{}")
	}

	// type_id NULL-able: у комплекта типа нет (см. equipment_type_nullable_for_kits).
	var typeID any
	if dto.TypeID != nil {
		typeID = dto.TypeID
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, realm_id, type_id, name, vendor, model, serial_number, inventory_number,
			specs, shop, order_number, status, parent_id, site_id, holder_user_id, holder_name, location, purchase_date, warranty_until, notes, created_by, is_kit)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9::jsonb, NULLIF($10, ''), NULLIF($11, ''), $12, $13, $14, $15, NULLIF($16, ''), NULLIF($17, ''), $18::date, $19::date, $20, $21, $22)`,
		Tables.Equipment)

	_, err := r.getExec(tx).Exec(ctx, query,
		id, dto.RealmID, typeID, dto.Name, dto.Vendor, dto.Model,
		dto.SerialNumber, dto.InventoryNumber, specs, dto.Shop, dto.OrderNumber, dto.Status, dto.ParentID, dto.SiteID,
		dto.HolderUserID, dto.HolderName, dto.Location, dto.PurchaseDate, dto.WarrantyUntil,
		dto.Notes, dto.CreatedBy, dto.IsKit)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}

	// Новое название магазина из freeSolo-поля попадает в справочник подсказок —
	// в той же транзакции, чтобы список и единица не расходились.
	if dto.Shop != nil {
		if err := r.EnsureShop(ctx, tx, dto.RealmID, *dto.Shop); err != nil {
			return nil, err
		}
	}

	return r.getByID(ctx, tx, &models.GetEquipmentByIdDTO{ID: id, RealmID: dto.RealmID})
}

func (r *EquipmentRepo) Update(ctx context.Context, tx Tx, dto *models.EquipmentDTO) (*models.Equipment, error) {
	specs := dto.Specs
	if len(specs) == 0 {
		specs = []byte("{}")
	}

	// type_id NULL-able: у комплекта типа нет.
	var typeID any
	if dto.TypeID != nil {
		typeID = dto.TypeID
	}
	query := fmt.Sprintf(`UPDATE %s SET type_id=$2, name=$3, vendor=$4, model=$5, serial_number=$6, inventory_number=$7,
			specs=$8::jsonb, shop=NULLIF($9, ''), order_number=NULLIF($10, ''), status=$11, parent_id=$12, site_id=$13, holder_user_id=$14, holder_name=$15, location=$16,
			purchase_date=$17::date, warranty_until=$18::date, notes=$19, updated_at=NOW()
		WHERE id=$1 AND realm_id=$20`, Tables.Equipment)

	res, err := r.getExec(tx).Exec(ctx, query,
		dto.ID, typeID, dto.Name, dto.Vendor, dto.Model,
		dto.SerialNumber, dto.InventoryNumber, specs, dto.Shop, dto.OrderNumber, dto.Status, dto.ParentID, dto.SiteID,
		dto.HolderUserID, dto.HolderName, dto.Location, dto.PurchaseDate, dto.WarrantyUntil,
		dto.Notes, dto.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	if tag := res.RowsAffected(); tag == 0 {
		return nil, models.ErrNotFound
	}
	// Магазин мог быть изменён только в этой форме — добавляем его в справочник.
	if dto.Shop != nil {
		if err := r.EnsureShop(ctx, tx, dto.RealmID, *dto.Shop); err != nil {
			return nil, err
		}
	}
	return r.getByID(ctx, tx, &models.GetEquipmentByIdDTO{ID: *dto.ID, RealmID: dto.RealmID})
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
