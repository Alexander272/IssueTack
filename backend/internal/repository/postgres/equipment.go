package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	// Периодические задачи
	GetPeriodicTaskTemplates(ctx context.Context, realmID uuid.UUID, equipmentTypeID *uuid.UUID) ([]*models.EquipmentTypePeriodicTask, error)
	GetPeriodicTaskTemplateByID(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentTypePeriodicTask, error)
	CreatePeriodicTaskTemplate(ctx context.Context, tx Tx, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error)
	UpdatePeriodicTaskTemplate(ctx context.Context, tx Tx, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error)
	DeletePeriodicTaskTemplate(ctx context.Context, dto *models.DelEquipmentTypePeriodicTaskDTO) error

	// Правила автоматических задач после операций с оборудованием
	GetOperationTasks(ctx context.Context, realmID uuid.UUID, operation *models.EquipmentOperation) ([]*models.EquipmentOperationTask, error)
	GetOperationTaskByID(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentOperationTask, error)
	CreateOperationTask(ctx context.Context, tx Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error)
	UpdateOperationTask(ctx context.Context, tx Tx, dto *models.EquipmentOperationTaskDTO) (*models.EquipmentOperationTask, error)
	DeleteOperationTask(ctx context.Context, dto *models.DelEquipmentOperationTaskDTO) error

	// Экземпляры периодических задач
	GetPeriodicTaskInstances(ctx context.Context, filter *models.GetEquipmentPeriodicTaskInstancesFilter) ([]*models.EquipmentPeriodicTaskInstance, int, error)
	// GetPendingPeriodicTasks возвращает задачи со статусом pending, у которых scheduled_at <= now,
	// по включённым шаблонам (шаблон должен быть enabled).
	GetPendingPeriodicTasks(ctx context.Context, now time.Time, limit int) ([]*models.EquipmentPeriodicTaskInstance, error)
	CreatePeriodicTaskInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance) error
	UpdatePeriodicTaskInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance) error
	// ClaimPeriodicTaskInstance атомарно переводит pending-экземпляр в created
	// (claim сбрасывает ticket_id/created_at/next_scheduled_at — они проставляются
	// после фактического создания заявки; при сбое заявки claim откатывается).
	ClaimPeriodicTaskInstance(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (bool, error)
	// RevertPeriodicTaskClaim возвращает экзаемпляр из created обратно в pending
	// (раскрывает claim после сбоя создания заявки): расписание и последняя выдача
	// восстанавливаются из снимка, взятого до claim.
	RevertPeriodicTaskClaim(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID, scheduledAt time.Time, lastAssignmentID *uuid.UUID) error
	// SetPeriodicTaskInstanceTicket привязывает созданную заявку к экземпляру
	// и проставляет next_scheduled_at (расписание следующего запуска).
	SetPeriodicTaskInstanceTicket(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance, ticketID uuid.UUID, nextScheduledAt time.Time) error
	// ReschedulePendingInstance переписывает расписание pending-экземпляра
	// (ручной запуск с shift=true).
	ReschedulePendingInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance, scheduledAt time.Time) error
	// HasPeriodicTaskInstances сообщает, есть ли уже экземпляры (не отменённые)
	// для пары «шаблон — единица»: автостарт при первой выдаче должен сработать
	// только если цепочка ещё не заводилась.
	HasPeriodicTaskInstances(ctx context.Context, templateID, equipmentID, realmID uuid.UUID) (bool, error)
	// GetPendingPeriodicTaskInstance возвращает единственный pending-экземпляр
	// пары «шаблон — единица».
	GetPendingPeriodicTaskInstance(ctx context.Context, templateID, equipmentID, realmID uuid.UUID) (*models.EquipmentPeriodicTaskInstance, error)
	// GetLastAssignmentForEquipment возвращает последнюю выдачу оборудования
	GetLastAssignmentForEquipment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error)
}

// equipmentSelectBase — выборка единицы со справочными полями (тип, площадка,
// ФИО держателя), именем контейнера (физический родитель или логический комплект)
// и COUNT(*) OVER() для тотала страницы.
//
// Вычисляемые колонки получают явные алиасы: чтение идёт по именам (db-теги
// models.Equipment), и без алиаса COALESCE(...) отдался бы колонкой «coalesce»,
// а полю модели соответствует «type_name».
func equipmentSelectBase() string {
	return fmt.Sprintf(`SELECT 
			e.id, e.realm_id, e.type_id,
			COALESCE(et.name, '') AS type_name,
			COALESCE(et.kind::text, '') AS type_kind,
			COALESCE(et.icon, '') AS type_icon,
			e.name, e.vendor, e.model,
			e.serial_number, e.inventory_number, e.specs, e.shop, e.order_number, e.status,
			e.parent_id, e.site_id, COALESCE(s.name, '') AS site_name,
			e.holder_user_id, e.holder_name,
			COALESCE(%s, '') AS holder_user_full_name,
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

// equipmentRow — строка выборки единицы: сама единица и тотал страницы
// (COUNT(*) OVER()), который в модель не входит.
type equipmentRow struct {
	models.Equipment
	TotalCount int `db:"total_count"`
}

// scanEquipmentRow читает одну строку equipmentSelectBase по именам колонок
// (db-теги models.Equipment). Порядок колонок в запросе не важен, а несовпадение
// «колонка ↔ поле» даёт ошибку, а не тихую подмену значений: при позиционном
// сканировании вставка колонки в середину SELECT сдвигала бы значения в чужие
// поля (две uuid-колонки — без единого признака).
func scanEquipmentRow(row pgx.CollectableRow) (*models.Equipment, int, error) {
	scanned, err := pgx.RowToStructByName[equipmentRow](row)
	if err != nil {
		return nil, 0, err
	}
	item := scanned.Equipment
	if len(item.Specs) == 0 {
		item.Specs = []byte("{}")
	}
	return &item, scanned.TotalCount, nil
}

// scanEquipmentRows читает все строки выборки единиц. Второе значение — тотал
// страницы из COUNT(*) OVER() (0, если строк нет).
func scanEquipmentRows(rows pgx.Rows) ([]*models.Equipment, int, error) {
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

	// Комплекты — логическая группировка, не физическая единица. На них нельзя
	// завести заявку, поэтому при подборе оборудования для заявки их исключаем.
	if filter.ExcludeKits {
		w.idx++
		w.add(fmt.Sprintf("e.is_kit = $%d", w.idx))
		w.args = append(w.args, false)
	}

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
	// Сортировка по типу (справочник): как у категорий по глобальному sort_order,
	// список единиц/комплектов идёт типами группирующим образом — клиенту не
	// нужно самому держать порядок групп, достаточно groupBy по type_id.
	query += " ORDER BY COALESCE(et.sort_order, 0), e.name"

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

	data, total, err := scanEquipmentRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *EquipmentRepo) GetByID(ctx context.Context, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	return r.getByID(ctx, nil, req)
}

// getByID — общая реализация чтения единицы. Внутри транзакции читать нужно через
// неё же (tx), иначе пул не увидит ещё не закоммиченную строку: так Create/Update
// возвращают состояние, записанное в той же транзакции.
//
// Чтение идёт через Query, а не QueryRow: сопоставление строки с моделью по
// именам колонок (pgx.RowToStructByName) требует описания полей, а его знает
// только набор строк — у одиночной строки pgx.Row их не отдаёт.
func (r *EquipmentRepo) getByID(ctx context.Context, tx Tx, req *models.GetEquipmentByIdDTO) (*models.Equipment, error) {
	query := equipmentSelectBase() + ` WHERE e.id = $1 AND e.realm_id = $2`

	rows, err := r.getExec(tx).Query(ctx, query, req.ID, req.RealmID)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data, _, err := scanEquipmentRows(rows)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, models.ErrNotFound
	}
	return data[0], nil
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

// GetPeriodicTaskTemplates возвращает шаблоны периодических задач реалма.
// Если equipmentTypeID != nil, фильтрует по типу.
func (r *EquipmentRepo) GetPeriodicTaskTemplates(ctx context.Context, realmID uuid.UUID, equipmentTypeID *uuid.UUID) ([]*models.EquipmentTypePeriodicTask, error) {
	query := fmt.Sprintf(`SELECT 
		pt.id, pt.realm_id, pt.equipment_type_id, pt.title, pt.description,
		pt.period_years, pt.period_months, pt.period_days,
		pt.category_id, pt.site_id,
		pt.priority, pt.auto_start_on_first_assign, pt.enabled,
		pt.created_at, pt.updated_at, pt.created_by,
		COALESCE(et.name, '') AS type_name,
		COALESCE(c.name, '') AS category_name,
		COALESCE(s.name, '') AS site_name
	FROM %s pt
	LEFT JOIN %s et ON et.id = pt.equipment_type_id
	LEFT JOIN %s c ON c.id = pt.category_id
	LEFT JOIN %s s ON s.id = pt.site_id
	WHERE pt.realm_id = $1`,
		Tables.EquipmentTypePeriodicTasks, Tables.EquipmentTypes, Tables.Categories, Tables.Sites)

	args := []any{realmID}
	if equipmentTypeID != nil {
		query += " AND pt.equipment_type_id = $2"
		args = append(args, *equipmentTypeID)
	}
	query += " ORDER BY pt.equipment_type_id, pt.title"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to query periodic task templates: %w", err))
	}
	defer rows.Close()

	var templates []*models.EquipmentTypePeriodicTask
	for rows.Next() {
		var t models.EquipmentTypePeriodicTask
		if err := rows.Scan(
			&t.ID, &t.RealmID, &t.EquipmentTypeID, &t.Title, &t.Description,
			&t.PeriodYears, &t.PeriodMonths, &t.PeriodDays,
			&t.CategoryID, &t.SiteID,
			&t.Priority, &t.AutoStartOnFirstAssign, &t.Enabled,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
			&t.EquipmentTypeName, &t.CategoryName, &t.SiteName,
		); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan periodic task template: %w", err))
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

// GetPeriodicTaskTemplateByID возвращает шаблон по ID.
func (r *EquipmentRepo) GetPeriodicTaskTemplateByID(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (*models.EquipmentTypePeriodicTask, error) {
	query := fmt.Sprintf(`SELECT 
		pt.id, pt.realm_id, pt.equipment_type_id, pt.title, pt.description,
		pt.period_years, pt.period_months, pt.period_days,
		pt.category_id, pt.site_id,
		pt.priority, pt.auto_start_on_first_assign, pt.enabled,
		pt.created_at, pt.updated_at, pt.created_by,
		COALESCE(et.name, '') AS type_name,
		COALESCE(c.name, '') AS category_name,
		COALESCE(s.name, '') AS site_name
	FROM %s pt
	LEFT JOIN %s et ON et.id = pt.equipment_type_id
	LEFT JOIN %s c ON c.id = pt.category_id
	LEFT JOIN %s s ON s.id = pt.site_id
	WHERE pt.id = $1 AND pt.realm_id = $2`,
		Tables.EquipmentTypePeriodicTasks, Tables.EquipmentTypes, Tables.Categories, Tables.Sites)

	var t models.EquipmentTypePeriodicTask
	if err := r.getExec(tx).QueryRow(ctx, query, id, realmID).Scan(
		&t.ID, &t.RealmID, &t.EquipmentTypeID, &t.Title, &t.Description,
		&t.PeriodYears, &t.PeriodMonths, &t.PeriodDays,
		&t.CategoryID, &t.SiteID,
		&t.Priority, &t.AutoStartOnFirstAssign, &t.Enabled,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
		&t.EquipmentTypeName, &t.CategoryName, &t.SiteName,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, MapError(fmt.Errorf("failed to query periodic task template: %w", err))
	}
	return &t, nil
}

// CreatePeriodicTaskTemplate создаёт шаблон периодической задачи.
func (r *EquipmentRepo) CreatePeriodicTaskTemplate(ctx context.Context, tx Tx, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error) {
	query := fmt.Sprintf(`INSERT INTO %s (
		realm_id, equipment_type_id, title, description,
		period_years, period_months, period_days,
		category_id, site_id, priority, auto_start_on_first_assign, enabled, created_by
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	RETURNING id, created_at, updated_at`,
		Tables.EquipmentTypePeriodicTasks)

	var t models.EquipmentTypePeriodicTask
	if err := r.getExec(tx).QueryRow(ctx, query,
		dto.RealmID, dto.EquipmentTypeID, dto.Title, dto.Description,
		dto.PeriodYears, dto.PeriodMonths, dto.PeriodDays,
		dto.CategoryID, dto.SiteID, dto.Priority, dto.AutoStartOnFirstAssign, dto.Enabled, dto.CreatedBy,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, MapError(fmt.Errorf("failed to create periodic task template: %w", err))
	}
	return r.GetPeriodicTaskTemplateByID(ctx, tx, t.ID, dto.RealmID)
}

// UpdatePeriodicTaskTemplate обновляет шаблон периодической задачи.
// created_by = COALESCE(пришедший автор, существующий): при отсутствии автора
// (CreatedBy == nil, например обновление старого шаблона) сохраняется прежний.
func (r *EquipmentRepo) UpdatePeriodicTaskTemplate(ctx context.Context, tx Tx, dto *models.EquipmentTypePeriodicTaskDTO) (*models.EquipmentTypePeriodicTask, error) {
	query := fmt.Sprintf(`UPDATE %s SET
		equipment_type_id = $2, title = $3, description = $4,
		period_years = $5, period_months = $6, period_days = $7,
		category_id = $8, site_id = $9, priority = $10,
		auto_start_on_first_assign = $11, enabled = $12,
		created_by = COALESCE($13, created_by), updated_at = NOW()
		WHERE id = $1 AND realm_id = $14`,
		Tables.EquipmentTypePeriodicTasks)

	result, err := r.getExec(tx).Exec(ctx, query,
		*dto.ID, dto.EquipmentTypeID, dto.Title, dto.Description,
		dto.PeriodYears, dto.PeriodMonths, dto.PeriodDays,
		dto.CategoryID, dto.SiteID, dto.Priority, dto.AutoStartOnFirstAssign, dto.Enabled, dto.CreatedBy, dto.RealmID,
	)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to update periodic task template: %w", err))
	}
	if result.RowsAffected() == 0 {
		return nil, models.ErrNotFound
	}
	return r.GetPeriodicTaskTemplateByID(ctx, tx, *dto.ID, dto.RealmID)
}

// DeletePeriodicTaskTemplate удаляет шаблон периодической задачи.
func (r *EquipmentRepo) DeletePeriodicTaskTemplate(ctx context.Context, dto *models.DelEquipmentTypePeriodicTaskDTO) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND realm_id = $2`,
		Tables.EquipmentTypePeriodicTasks)

	result, err := r.db.Exec(ctx, query, dto.ID, dto.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to delete periodic task template: %w", err))
	}
	if result.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// GetPeriodicTaskInstances возвращает экземпляры периодических задач.
func (r *EquipmentRepo) GetPeriodicTaskInstances(ctx context.Context, filter *models.GetEquipmentPeriodicTaskInstancesFilter) ([]*models.EquipmentPeriodicTaskInstance, int, error) {
	limit := filter.Limit
	offset := filter.Offset
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`SELECT 
		i.id, i.realm_id, i.template_id, i.equipment_id, i.ticket_id,
		i.status, i.scheduled_at, i.created_at, i.next_scheduled_at,
		i.last_assignment_id, i.created_by,
		COALESCE(eq.name, '') AS equipment_name,
		COALESCE(t.title, '') AS ticket_title,
		COALESCE(t.ticket_number, 0) AS ticket_number
	FROM %s i
	LEFT JOIN %s eq ON eq.id = i.equipment_id
	LEFT JOIN %s t ON t.id = i.ticket_id
	WHERE i.realm_id = $1`,
		Tables.EquipmentPeriodicTaskInstances, Tables.Equipment, Tables.Tickets)

	args := []any{filter.RealmID}
	if filter.TemplateID != nil {
		query += " AND i.template_id = " + fmt.Sprintf("$%d", len(args)+1)
		args = append(args, *filter.TemplateID)
	}
	if filter.EquipmentID != nil {
		query += " AND i.equipment_id = " + fmt.Sprintf("$%d", len(args)+1)
		args = append(args, *filter.EquipmentID)
	}
	if filter.Status != nil {
		query += " AND i.status = " + fmt.Sprintf("$%d", len(args)+1)
		args = append(args, *filter.Status)
	}

	countQuery := query + " OFFSET 0"
	countQuery = strings.ReplaceAll(countQuery, `SELECT 
		i.id, i.realm_id, i.template_id, i.equipment_id, i.ticket_id,
		i.status, i.scheduled_at, i.created_at, i.next_scheduled_at,
		i.last_assignment_id, i.created_by,
		COALESCE(eq.name, '') AS equipment_name,
		COALESCE(t.title, '') AS ticket_title,
		COALESCE(t.ticket_number, 0) AS ticket_number`, "SELECT COUNT(*)")

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to count periodic task instances: %w", err))
	}

	query += fmt.Sprintf(" ORDER BY i.scheduled_at DESC OFFSET $%d LIMIT $%d", len(args)+1, len(args)+2)
	args = append(args, offset, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to query periodic task instances: %w", err))
	}
	defer rows.Close()

	var instances []*models.EquipmentPeriodicTaskInstance
	for rows.Next() {
		var i models.EquipmentPeriodicTaskInstance
		if err := rows.Scan(
			&i.ID, &i.RealmID, &i.TemplateID, &i.EquipmentID, &i.TicketID,
			&i.Status, &i.ScheduledAt, &i.CreatedAt, &i.NextScheduledAt,
			&i.LastAssignmentID, &i.CreatedBy,
			&i.EquipmentName, &i.TicketTitle, &i.TicketNumber,
		); err != nil {
			return nil, 0, MapError(fmt.Errorf("failed to scan periodic task instance: %w", err))
		}
		instances = append(instances, &i)
	}
	return instances, total, nil
}

// GetPendingPeriodicTasks возвращает задачи со статусом pending, у которых scheduled_at <= now,
// по включённым шаблонам (правило должно быть enabled, иначе тик не должен стартить).
func (r *EquipmentRepo) GetPendingPeriodicTasks(ctx context.Context, now time.Time, limit int) ([]*models.EquipmentPeriodicTaskInstance, error) {
	if limit <= 0 {
		limit = 100
	}
	query := fmt.Sprintf(`SELECT 
		i.id, i.realm_id, i.template_id, i.equipment_id, i.ticket_id,
		i.status, i.scheduled_at, i.created_at, i.next_scheduled_at,
		i.last_assignment_id, i.created_by,
		COALESCE(eq.name, '') AS equipment_name,
		COALESCE(t.title, '') AS ticket_title,
		COALESCE(t.ticket_number, 0) AS ticket_number
	FROM %s i
	JOIN %s pt ON pt.id = i.template_id AND pt.realm_id = i.realm_id AND pt.enabled = true
	LEFT JOIN %s eq ON eq.id = i.equipment_id
	LEFT JOIN %s t ON t.id = i.ticket_id
	WHERE i.status = 'pending' AND i.scheduled_at <= $1
	ORDER BY i.scheduled_at ASC LIMIT $2`,
		Tables.EquipmentPeriodicTaskInstances, Tables.EquipmentTypePeriodicTasks, Tables.Equipment, Tables.Tickets)

	rows, err := r.db.Query(ctx, query, now, limit)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to query pending periodic tasks: %w", err))
	}
	defer rows.Close()

	var instances []*models.EquipmentPeriodicTaskInstance
	for rows.Next() {
		var i models.EquipmentPeriodicTaskInstance
		if err := rows.Scan(
			&i.ID, &i.RealmID, &i.TemplateID, &i.EquipmentID, &i.TicketID,
			&i.Status, &i.ScheduledAt, &i.CreatedAt, &i.NextScheduledAt,
			&i.LastAssignmentID, &i.CreatedBy,
			&i.EquipmentName, &i.TicketTitle, &i.TicketNumber,
		); err != nil {
			return nil, MapError(fmt.Errorf("failed to scan pending periodic task: %w", err))
		}
		instances = append(instances, &i)
	}
	return instances, nil
}

// CreatePeriodicTaskInstance создаёт экземпляр периодической задачи.
// ON CONFLICT DO NOTHING — дедуп по (realm, template, equipment, scheduled_at):
// повторный старт на то же расписание не порождает второй pending.
func (r *EquipmentRepo) CreatePeriodicTaskInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance) error {
	query := fmt.Sprintf(`INSERT INTO %s (
		id, realm_id, template_id, equipment_id, ticket_id,
		status, scheduled_at, created_at, next_scheduled_at,
		last_assignment_id, created_by
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (realm_id, template_id, equipment_id, scheduled_at) DO NOTHING`,
		Tables.EquipmentPeriodicTaskInstances)

	_, err := r.getExec(tx).Exec(ctx, query,
		instance.ID, instance.RealmID, instance.TemplateID, instance.EquipmentID,
		instance.TicketID, string(instance.Status), instance.ScheduledAt,
		instance.CreatedAt, instance.NextScheduledAt,
		instance.LastAssignmentID, instance.CreatedBy,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to create periodic task instance: %w", err))
	}
	return nil
}

// UpdatePeriodicTaskInstance обновляет экземпляр периодической задачи.
func (r *EquipmentRepo) UpdatePeriodicTaskInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance) error {
	query := fmt.Sprintf(`UPDATE %s SET
		status = $2, ticket_id = $3, created_at = $4, next_scheduled_at = $5,
		last_assignment_id = $6, created_by = $7
		WHERE id = $1 AND realm_id = $8`,
		Tables.EquipmentPeriodicTaskInstances)

	result, err := r.getExec(tx).Exec(ctx, query,
		instance.ID, string(instance.Status), instance.TicketID,
		instance.CreatedAt, instance.NextScheduledAt,
		instance.LastAssignmentID, instance.CreatedBy,
		instance.RealmID,
	)
	if err != nil {
		return MapError(fmt.Errorf("failed to update periodic task instance: %w", err))
	}
	if result.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// HasPeriodicTaskInstances сообщает, есть ли уже экземпляры (не отменённые)
// для пары «шаблон — единица».
func (r *EquipmentRepo) HasPeriodicTaskInstances(ctx context.Context, templateID, equipmentID, realmID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS (
		SELECT 1 FROM %s i
		WHERE i.realm_id = $1 AND i.template_id = $2 AND i.equipment_id = $3 AND i.status <> 'cancelled'
	) AS exists`,
		Tables.EquipmentPeriodicTaskInstances)

	var exists bool
	if err := r.db.QueryRow(ctx, query, realmID, templateID, equipmentID).Scan(&exists); err != nil {
		return false, MapError(fmt.Errorf("failed to check periodic task instances: %w", err))
	}
	return exists, nil
}

// GetPendingPeriodicTaskInstance возвращает единственный pending-экземпляр
// пары «шаблон — единица» (ближайший по расписанию).
func (r *EquipmentRepo) GetPendingPeriodicTaskInstance(ctx context.Context, templateID, equipmentID, realmID uuid.UUID) (*models.EquipmentPeriodicTaskInstance, error) {
	query := fmt.Sprintf(`SELECT 
		i.id, i.realm_id, i.template_id, i.equipment_id, i.ticket_id,
		i.status, i.scheduled_at, i.created_at, i.next_scheduled_at,
		i.last_assignment_id, i.created_by,
		COALESCE(eq.name, '') AS equipment_name,
		COALESCE(t.title, '') AS ticket_title,
		COALESCE(t.ticket_number, 0) AS ticket_number
	FROM %s i
	LEFT JOIN %s eq ON eq.id = i.equipment_id
	LEFT JOIN %s t ON t.id = i.ticket_id
	WHERE i.realm_id = $1 AND i.template_id = $2 AND i.equipment_id = $3 AND i.status = 'pending'
	ORDER BY i.scheduled_at DESC LIMIT 1`,
		Tables.EquipmentPeriodicTaskInstances, Tables.Equipment, Tables.Tickets)

	var i models.EquipmentPeriodicTaskInstance
	if err := r.db.QueryRow(ctx, query, realmID, templateID, equipmentID).Scan(
		&i.ID, &i.RealmID, &i.TemplateID, &i.EquipmentID, &i.TicketID,
		&i.Status, &i.ScheduledAt, &i.CreatedAt, &i.NextScheduledAt,
		&i.LastAssignmentID, &i.CreatedBy,
		&i.EquipmentName, &i.TicketTitle, &i.TicketNumber,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, MapError(fmt.Errorf("failed to query pending periodic task: %w", err))
	}
	return &i, nil
}

// ClaimPeriodicTaskInstance атомарно переводит pending-экземпляр в created.
// Возвращает true, если claim сработал (статус реально pending → created); false —
// экземпляр уже обрабатывался (конкурентный тик): повторно не трогаем.
func (r *EquipmentRepo) ClaimPeriodicTaskInstance(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`UPDATE %s SET
		status = 'created', ticket_id = NULL, created_at = NULL, next_scheduled_at = NULL
		WHERE id = $1 AND realm_id = $2 AND status = 'pending'`,
		Tables.EquipmentPeriodicTaskInstances)

	result, err := r.getExec(tx).Exec(ctx, query, id, realmID)
	if err != nil {
		return false, MapError(fmt.Errorf("failed to claim periodic task instance: %w", err))
	}
	return result.RowsAffected() > 0, nil
}

// RevertPeriodicTaskClaim возвращает экземпляр из created обратно в pending
// (разворачивает claim после сбоя создания заявки): расписание и привязка к
// последней выдаче восстанавливаются из снимка, взятого до claim.
func (r *EquipmentRepo) RevertPeriodicTaskClaim(ctx context.Context, tx Tx, id uuid.UUID, realmID uuid.UUID, scheduledAt time.Time, lastAssignmentID *uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET
		status = 'pending', ticket_id = NULL, created_at = NULL, next_scheduled_at = NULL,
		scheduled_at = $3, last_assignment_id = $4
		WHERE id = $1 AND realm_id = $2 AND status = 'created'`,
		Tables.EquipmentPeriodicTaskInstances)

	_, err := r.getExec(tx).Exec(ctx, query, id, realmID, scheduledAt, lastAssignmentID)
	if err != nil {
		return MapError(fmt.Errorf("failed to revert periodic task claim: %w", err))
	}
	return nil
}

// SetPeriodicTaskInstanceTicket привязывает созданную заявку к экземпляру и
// проставляет next_scheduled_at — момент следующего запуска (расписание + период).
func (r *EquipmentRepo) SetPeriodicTaskInstanceTicket(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance, ticketID uuid.UUID, nextScheduledAt time.Time) error {
	query := fmt.Sprintf(`UPDATE %s SET
		status = 'created', ticket_id = $2, created_at = NOW(), next_scheduled_at = $3
		WHERE id = $1 AND realm_id = $4 AND status = 'created'`,
		Tables.EquipmentPeriodicTaskInstances)

	result, err := r.getExec(tx).Exec(ctx, query, instance.ID, ticketID, nextScheduledAt, instance.RealmID)
	if err != nil {
		return MapError(fmt.Errorf("failed to set periodic task instance ticket: %w", err))
	}
	if result.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// ReschedulePendingInstance переписывает расписание pending-экземпляра
// (ручной запуск с shift=true).
func (r *EquipmentRepo) ReschedulePendingInstance(ctx context.Context, tx Tx, instance *models.EquipmentPeriodicTaskInstance, scheduledAt time.Time) error {
	query := fmt.Sprintf(`UPDATE %s SET scheduled_at = $2, last_assignment_id = $3
		WHERE id = $1 AND realm_id = $4 AND status = 'pending'`,
		Tables.EquipmentPeriodicTaskInstances)

	result, err := r.getExec(tx).Exec(ctx, query, instance.ID, instance.RealmID, scheduledAt, instance.LastAssignmentID)
	if err != nil {
		return MapError(fmt.Errorf("failed to reschedule pending periodic task: %w", err))
	}
	if result.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

// GetLastAssignmentForEquipment возвращает последнюю выдачу оборудования.
// Строка собирается тем же assignmentSelectBase, что и остальные выдачи: иначе
// набор колонок (и их алиасы) разъезжается с scanAssignment, а неполные имена
// колонок в SELECT становятся неоднозначными из-за JOIN'ов справочников.
func (r *EquipmentRepo) GetLastAssignmentForEquipment(ctx context.Context, equipmentID uuid.UUID) (*models.EquipmentAssignment, error) {
	query := assignmentSelectBase() +
		` WHERE a.equipment_id = $1 ORDER BY a.assigned_at DESC LIMIT 1`

	a, err := scanAssignment(r.db.QueryRow(ctx, query, equipmentID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, MapError(fmt.Errorf("failed to query last assignment: %w", err))
	}
	return a, nil
}
