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

// StatisticsRepo — агрегаты статистики заявок. Живёт отдельно от TicketRepo:
// только читает по таблице tickets (и связанным справочникам), транзакции не
// использует, поэтому держит лишь пул.
type StatisticsRepo struct {
	db *pgxpool.Pool
}

func NewStatisticsRepo(db *pgxpool.Pool) *StatisticsRepo {
	return &StatisticsRepo{
		db: db,
	}
}

type Statistics interface {
	GetStatisticsSummary(ctx context.Context, query models.StatisticsQuery) (*models.StatisticsSummary, error)
	GetStatisticsByStatus(ctx context.Context, query models.StatisticsQuery) ([]*models.StatusBucket, error)
	GetStatisticsByDimension(ctx context.Context, query models.StatisticsQuery) ([]*models.StatisticsBucket, error)
	GetStatisticsByOwner(ctx context.Context, query models.StatisticsQuery) ([]*models.StatisticsBreakdownBucket, error)
	GetStatisticsWorkload(ctx context.Context, query models.StatisticsQuery) ([]*models.WorkloadBucket, error)
	GetStatisticsTrend(ctx context.Context, query models.StatisticsQuery) ([]*models.TrendPoint, error)
	GetStatisticsTickets(ctx context.Context, query models.StatisticsTicketsQuery) ([]*models.Ticket, int, error)
	GetKpiTickets(ctx context.Context, query models.StatisticsKpiQuery) ([]*models.Ticket, int, error)
}

// statisticsScope добавляет в построитель условие видимости заявок для актора:
// реалм плюс (если актор не начальник области) дизъюнкция измерений среза.
// Ровно одно измерение заполнено вызывающим сервисом (см. models.StatisticsScope),
// но дизъюнкция устойчива и к нескольким: менеджер видит управляемые группы И свои
// назначенные заявки.
func (w *whereBuilder) statisticsScope(scope models.StatisticsScope) {
	w.idx++
	w.add(fmt.Sprintf("t.realm_id = $%d", w.idx))
	w.args = append(w.args, scope.RealmID)

	if scope.AllRealm {
		return
	}

	disj := make([]string, 0, 3)
	if scope.AssigneeID != nil {
		w.idx++
		disj = append(disj, fmt.Sprintf("t.assignee_id = $%d", w.idx))
		w.args = append(w.args, *scope.AssigneeID)
	}
	if scope.RequesterID != nil {
		w.idx++
		creator := w.idx
		w.idx++
		owner := w.idx
		disj = append(disj, fmt.Sprintf("(t.creator_id = $%d OR t.owner_id = $%d)", creator, owner))
		w.args = append(w.args, *scope.RequesterID, *scope.RequesterID)
	}
	if len(scope.GroupIDs) > 0 {
		w.args = append(w.args, toAny(scope.GroupIDs)...)
		disj = append(disj, "t.group_id IN ("+w.place(len(scope.GroupIDs))+")")
	}

	if len(disj) == 0 {
		// Защита от «среза без измерений»: пустой срез не должен раскрывать весь реалм.
		w.add("FALSE")
		return
	}
	w.add("(" + strings.Join(disj, " OR ") + ")")
}

// statisticsRefinements добавляет пользовательские уточнения выборки (исполнитель,
// категория, группа, площадка) как обычные AND. Они всегда идут поверх
// statisticsScope, поэтому не могут расширить срез доступа.
func (w *whereBuilder) statisticsRefinements(filter *models.StatisticsFilter) {
	if filter == nil {
		return
	}
	inList(w, "t.assignee_id", filter.AssigneeIDs)
	inList(w, "t.category_id", filter.CategoryIDs)
	inList(w, "t.group_id", filter.GroupIDs)
	inList(w, "t.site_id", filter.SiteIDs)
}

// statisticsWhere применяет срез доступа и уточнения фильтра одной точкой входа,
// чтобы ни один статистический запрос не остался без ограничения доступа.
func (w *whereBuilder) statisticsWhere(query models.StatisticsQuery) {
	w.statisticsScope(query.Scope)
	w.statisticsRefinements(query.Filter)
}

// rangeArgs добавляет аргументы периода [from, to) и возвращает их плейсхолдеры.
func (w *whereBuilder) rangeArgs(from, to time.Time) (string, string) {
	w.idx++
	fromPh := fmt.Sprintf("$%d", w.idx)
	w.idx++
	toPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, from, to)
	return fromPh, toPh
}

// GetStatisticsSummary возвращает агрегаты верхнего уровня по видимым заявкам.
// Периодные метрики (Total, Resolved) считаются за [from, to), а TotalPrev и
// ResolvedPrev — за предыдущее окно той же длины, чтобы клиент мог показать дельту.
func (r *StatisticsRepo) GetStatisticsSummary(ctx context.Context, q models.StatisticsQuery) (*models.StatisticsSummary, error) {
	from, to := q.Filter.From, q.Filter.To
	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)

	span := to.Sub(from)
	prevFromPh, prevToPh := w.rangeArgs(from.Add(-span), from)

	// Один проход по видимым заявкам: LEFT JOIN LATERAL считает подзадачи только
	// для попавших в выборку тикетов (индекс subtasks(ticket_id)), а условные
	// агрегаты дают и число карточек, и цифры блока «подзадачи» для каждой из них.
	activeCond := "t.status IN ('open', 'in_progress', 'pending', 'on_hold')"
	overdueCond := "t.due_date IS NOT NULL AND t.due_date < NOW() AND t.status NOT IN ('closed', 'cancelled', 'resolved')"
	createdInPeriod := fmt.Sprintf("t.created_at >= %s AND t.created_at < %s", fromPh, toPh)
	resolvedInPeriod := fmt.Sprintf("t.resolved_at >= %s AND t.resolved_at < %s", fromPh, toPh)

	query := fmt.Sprintf(`SELECT
			COUNT(*) FILTER (WHERE %[1]s),
			COUNT(*) FILTER (WHERE %[2]s),
			COUNT(*) FILTER (WHERE %[3]s),
			COUNT(*) FILTER (WHERE %[4]s),
			COUNT(*) FILTER (WHERE t.created_at >= %[5]s AND t.created_at < %[6]s),
			COUNT(*) FILTER (WHERE t.resolved_at >= %[5]s AND t.resolved_at < %[6]s),
			COUNT(*) FILTER (WHERE sc.cnt > 0 AND %[1]s),
			COALESCE(SUM(sc.cnt) FILTER (WHERE %[1]s), 0)::bigint,
			COUNT(*) FILTER (WHERE sc.cnt > 0 AND %[2]s),
			COALESCE(SUM(sc.cnt) FILTER (WHERE %[2]s), 0)::bigint,
			COUNT(*) FILTER (WHERE sc.cnt > 0 AND %[3]s),
			COALESCE(SUM(sc.cnt) FILTER (WHERE %[3]s), 0)::bigint,
			COUNT(*) FILTER (WHERE sc.cnt > 0 AND %[4]s),
			COALESCE(SUM(sc.cnt) FILTER (WHERE %[4]s), 0)::bigint
		FROM %[7]s t
		CROSS JOIN LATERAL (SELECT COUNT(*) AS cnt FROM %[8]s st WHERE st.ticket_id = t.id) sc
		WHERE %[9]s`,
		createdInPeriod, activeCond, overdueCond, resolvedInPeriod,
		prevFromPh, prevToPh,
		Tables.Tickets, Tables.Subtasks,
		strings.Join(w.clauses, " AND "),
	)

	summary := &models.StatisticsSummary{}
	if err := r.db.QueryRow(ctx, query, w.args...).Scan(
		&summary.Total, &summary.Active, &summary.Overdue, &summary.Resolved,
		&summary.TotalPrev, &summary.ResolvedPrev,
		&summary.Subtasks.Total.WithSubtasks, &summary.Subtasks.Total.TotalSubtasks,
		&summary.Subtasks.Active.WithSubtasks, &summary.Subtasks.Active.TotalSubtasks,
		&summary.Subtasks.Overdue.WithSubtasks, &summary.Subtasks.Overdue.TotalSubtasks,
		&summary.Subtasks.Resolved.WithSubtasks, &summary.Subtasks.Resolved.TotalSubtasks,
	); err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	return summary, nil
}

// GetStatisticsByStatus возвращает распределение видимых заявок по статусу за
// период по created_at (заявки, созданные в окне, по их текущему статусу).
func (r *StatisticsRepo) GetStatisticsByStatus(ctx context.Context, q models.StatisticsQuery) ([]*models.StatusBucket, error) {
	from, to := q.Filter.From, q.Filter.To
	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)

	query := fmt.Sprintf(`SELECT t.status, COUNT(*) AS count
		FROM %s t
		WHERE %s AND t.created_at >= %s AND t.created_at < %s
		GROUP BY t.status
		ORDER BY t.status`,
		Tables.Tickets, strings.Join(w.clauses, " AND "), fromPh, toPh,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.StatusBucket{}
	for rows.Next() {
		bucket := &models.StatusBucket{}
		if err := rows.Scan(&bucket.Status, &bucket.Count); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// statisticsDimension описывает разрез агрегата: JOIN и колонки группировки.
type statisticsDimension struct {
	join  string
	id    string
	name  string
	group string
}

var statisticsDimensions = map[string]statisticsDimension{
	"category": {
		join:  fmt.Sprintf("LEFT JOIN %s c ON t.category_id = c.id", Tables.Categories),
		id:    "COALESCE(c.id::text, '')",
		name:  "COALESCE(c.name, 'Без категории')",
		group: "c.id, c.name",
	},
	"group": {
		join:  fmt.Sprintf("LEFT JOIN %s g ON t.group_id = g.id", Tables.Groups),
		id:    "COALESCE(g.id::text, '')",
		name:  "COALESCE(g.name, 'Без группы')",
		group: "g.id, g.name",
	},
	"site": {
		join:  fmt.Sprintf("LEFT JOIN %s s ON t.site_id = s.id", Tables.Sites),
		id:    "COALESCE(s.id::text, '')",
		name:  "COALESCE(s.name, 'Без площадки')",
		group: "s.id, s.name",
	},
}

// GetStatisticsByDimension возвращает разрез видимых заявок, созданных за период,
// по измерению (category|group|site), отсортированный по убыванию количества.
func (r *StatisticsRepo) GetStatisticsByDimension(ctx context.Context, q models.StatisticsQuery) ([]*models.StatisticsBucket, error) {
	d, ok := statisticsDimensions[q.Dim]
	if !ok {
		return nil, fmt.Errorf("unknown statistics dimension: %q", q.Dim)
	}

	from, to := q.Filter.From, q.Filter.To
	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	w.idx++
	limitPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, limit)

	query := fmt.Sprintf(`SELECT %s AS id, %s AS name, COUNT(*) AS count
		FROM %s t
		%s
		WHERE %s AND t.created_at >= %s AND t.created_at < %s
		GROUP BY %s
		ORDER BY count DESC, name ASC
		LIMIT %s`,
		d.id, d.name, Tables.Tickets, d.join, strings.Join(w.clauses, " AND "), fromPh, toPh, d.group, limitPh,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.StatisticsBucket{}
	for rows.Next() {
		bucket := &models.StatisticsBucket{}
		var idStr string
		if err := rows.Scan(&idStr, &bucket.Name, &bucket.Count); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if id, err := uuid.Parse(idStr); err == nil {
			bucket.ID = id
		}
		data = append(data, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// GetStatisticsByOwner возвращает разрез видимых заявок, созданных за период, по
// заказчику: Total — все заявки за период, Active — из них ещё в активном статусе.
// Заявки без заказчика собираются в бакет «Без заказчика» с пустым ID.
func (r *StatisticsRepo) GetStatisticsByOwner(ctx context.Context, q models.StatisticsQuery) ([]*models.StatisticsBreakdownBucket, error) {
	from, to := q.Filter.From, q.Filter.To
	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)

	// 0 (или отрицательное) — без LIMIT: фронт сворачивает хвост в «Прочие»
	// (топ-N, максимум 20), поэтому строки за пределами топа тоже нужны.
	limitClause := ""
	if q.Limit > 0 {
		w.idx++
		limitPh := fmt.Sprintf("$%d", w.idx)
		w.args = append(w.args, q.Limit)
		limitClause = fmt.Sprintf(" LIMIT %s", limitPh)
	}

	query := fmt.Sprintf(`SELECT
			COALESCE(u.id::text, ''),
			COALESCE(NULLIF(TRIM(COALESCE(u.last_name, '') || ' ' || COALESCE(u.first_name, '')), ''), u.username, 'Без заказчика') AS name,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE t.status IN ('open', 'in_progress', 'pending', 'on_hold')) AS active
		FROM %s t
		LEFT JOIN %s u ON t.owner_id = u.id
		WHERE %s AND t.created_at >= %s AND t.created_at < %s
		GROUP BY u.id, u.last_name, u.first_name, u.username
		ORDER BY total DESC, name ASC%s`,
		Tables.Tickets, Tables.Users, strings.Join(w.clauses, " AND "), fromPh, toPh, limitClause,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.StatisticsBreakdownBucket{}
	for rows.Next() {
		bucket := &models.StatisticsBreakdownBucket{}
		var idStr string
		if err := rows.Scan(&idStr, &bucket.Name, &bucket.Total, &bucket.Active); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if id, err := uuid.Parse(idStr); err == nil {
			bucket.ID = id
		}
		data = append(data, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// GetStatisticsWorkload возвращает нагрузку по исполнителям за период: каждый бакет
// несёт два столбца (active/closed), каждый — stack из задач и подзадач. Исполнитель
// подзадачи считается исполнителем родительской заявки (JOIN по ticket_id).
func (r *StatisticsRepo) GetStatisticsWorkload(ctx context.Context, q models.StatisticsQuery) ([]*models.WorkloadBucket, error) {
	// 1. Инициализируем твой билдер. idx=2 резервирует $1 и $2 под from/to.
	w := &whereBuilder{idx: 2}

	// Базовое условие realm_id
	w.add(fmt.Sprintf("t.realm_id = $%d", w.idx+1))
	w.args = append(w.args, q.Scope.RealmID)
	w.idx++

	// Scope условия
	if !q.Scope.AllRealm {
		disj := make([]string, 0, 3)
		if q.Scope.AssigneeID != nil {
			w.idx++
			disj = append(disj, fmt.Sprintf("t.assignee_id = $%d", w.idx))
			w.args = append(w.args, *q.Scope.AssigneeID)
		}
		if q.Scope.RequesterID != nil {
			w.idx++
			creator := w.idx
			w.idx++
			owner := w.idx
			disj = append(disj, fmt.Sprintf("(t.creator_id = $%d OR t.owner_id = $%d)", creator, owner))
			w.args = append(w.args, *q.Scope.RequesterID, *q.Scope.RequesterID)
		}
		if len(q.Scope.GroupIDs) > 0 {
			w.args = append(w.args, toAny(q.Scope.GroupIDs)...)
			disj = append(disj, "t.group_id IN ("+w.place(len(q.Scope.GroupIDs))+")")
		}
		if len(disj) == 0 {
			w.add("FALSE")
		} else {
			w.add("(" + strings.Join(disj, " OR ") + ")")
		}
	}

	// Refinements
	inList(w, "t.assignee_id", q.Filter.AssigneeIDs)
	inList(w, "t.category_id", q.Filter.CategoryIDs)
	inList(w, "t.group_id", q.Filter.GroupIDs)
	inList(w, "t.site_id", q.Filter.SiteIDs)

	scopeCond := strings.Join(w.clauses, " AND ")

	// 2. Подготовка плейсхолдеров для статусов
	activeStatuses := []models.TicketStatus{"open", "in_progress", "pending", "on_hold"}
	closedStatuses := []models.TicketStatus{"resolved", "closed", "cancelled"}

	w.args = append(w.args, toAny(activeStatuses)...)
	activePh := w.place(len(activeStatuses))

	w.args = append(w.args, toAny(closedStatuses)...)
	closedPh := w.place(len(closedStatuses))

	// Limit
	limitClause := ""
	if q.Limit > 0 {
		w.idx++
		limitClause = fmt.Sprintf("LIMIT $%d", w.idx)
		w.args = append(w.args, q.Limit)
	}

	// 3. Оптимизированный запрос с ПОДЗАПРОСОМ для финальной агрегации
	// Это полностью устраняет конфликты GROUP BY и ORDER BY в PostgreSQL
	query := fmt.Sprintf(`
		WITH TicketStats AS (
			SELECT 
				t.assignee_id AS user_id,
				COUNT(*) FILTER (WHERE t.status IN (%s)) AS active_tasks,
				COUNT(*) FILTER (WHERE t.status IN (%s)) AS closed_tasks,
				0::bigint AS active_subtasks,
				0::bigint AS closed_subtasks
			FROM %s t
			WHERE t.created_at >= $1 AND t.created_at < $2
			  AND %s
			GROUP BY t.assignee_id
		),
		SubtaskStats AS (
			SELECT 
				t.assignee_id AS user_id,
				0::bigint AS active_tasks,
				0::bigint AS closed_tasks,
				COUNT(*) FILTER (WHERE st.status IN (%s)) AS active_subtasks,
				COUNT(*) FILTER (WHERE st.status IN (%s)) AS closed_subtasks
			FROM %s st
			JOIN %s t ON t.id = st.ticket_id
			WHERE t.created_at >= $1 AND t.created_at < $2
			  AND %s
			GROUP BY t.assignee_id
		),
		CombinedStats AS (
			SELECT * FROM TicketStats
			UNION ALL
			SELECT * FROM SubtaskStats
		)
		SELECT 
			user_id::text,
			name,
			active_tasks,
			closed_tasks,
			active_subtasks,
			closed_subtasks
		FROM (
			SELECT 
				u.id AS user_id,
				COALESCE(NULLIF(TRIM(COALESCE(u.last_name, '') || ' ' || COALESCE(u.first_name, '')), ''), u.username) AS name,
				SUM(cs.active_tasks)::bigint AS active_tasks,
				SUM(cs.closed_tasks)::bigint AS closed_tasks,
				SUM(cs.active_subtasks)::bigint AS active_subtasks,
				SUM(cs.closed_subtasks)::bigint AS closed_subtasks
			FROM %s u
			JOIN CombinedStats cs ON cs.user_id = u.id
			GROUP BY u.id, u.last_name, u.first_name, u.username
		) AS aggregated
		WHERE (active_tasks + closed_tasks + active_subtasks + closed_subtasks) > 0
		ORDER BY 
			(active_tasks + closed_tasks + active_subtasks + closed_subtasks) DESC,
			(active_tasks + active_subtasks) DESC,
			name ASC
		%s
	`,
		// TicketStats args
		activePh, closedPh, Tables.Tickets, scopeCond,
		// SubtaskStats args
		activePh, closedPh, Tables.Subtasks, Tables.Tickets, scopeCond,
		// Final aggregation
		Tables.Users,
		limitClause,
	)

	// 4. Собираем финальный массив аргументов: [from, to] + [все аргументы из билдера]
	args := []any{q.Filter.From, q.Filter.To}
	args = append(args, w.args...)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := make([]*models.WorkloadBucket, 0)
	for rows.Next() {
		bucket := &models.WorkloadBucket{}
		var idStr string
		if err := rows.Scan(&idStr, &bucket.Name, &bucket.ActiveTasks, &bucket.ClosedTasks, &bucket.ActiveSubtasks, &bucket.ClosedSubtasks); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		if id, err := uuid.Parse(idStr); err == nil {
			bucket.UserID = id
		}
		data = append(data, bucket)
	}

	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, nil
}

// GetStatisticsTrend возвращает динамику «создано vs решено» по бакетам
// гранулярности (day|week|month) за период [from, to).
func (r *StatisticsRepo) GetStatisticsTrend(ctx context.Context, q models.StatisticsQuery) ([]*models.TrendPoint, error) {
	from, to := q.Filter.From, q.Filter.To

	gran := "day"
	switch q.Granularity {
	case "week", "month":
		gran = q.Granularity
	}

	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)
	where := strings.Join(w.clauses, " AND ")

	query := fmt.Sprintf(`SELECT bucket, created, resolved
		FROM (
			SELECT to_char(bucket, 'YYYY-MM-DD') AS bucket,
				COUNT(*) FILTER (WHERE kind = 'created') AS created,
				COUNT(*) FILTER (WHERE kind = 'resolved') AS resolved
			FROM (
				SELECT date_trunc('%s', t.created_at) AS bucket, 'created' AS kind
				FROM %s t
				WHERE %s AND t.created_at >= %s AND t.created_at < %s
				UNION ALL
				SELECT date_trunc('%s', t.resolved_at) AS bucket, 'resolved' AS kind
				FROM %s t
				WHERE %s AND t.resolved_at >= %s AND t.resolved_at < %s
			) series
			GROUP BY bucket
		) points
		ORDER BY bucket`,
		gran, Tables.Tickets, where, fromPh, toPh,
		gran, Tables.Tickets, where, fromPh, toPh,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.TrendPoint{}
	for rows.Next() {
		point := &models.TrendPoint{}
		if err := rows.Scan(&point.Date, &point.Created, &point.Resolved); err != nil {
			return nil, MapError(fmt.Errorf("scan row error: %w", err))
		}
		data = append(data, point)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, nil
}

// statisticsDrilldownActive/Closed — статусные наборы колец диаграммы. «Активные»
// здесь, в отличие от activeStatuses, НЕ включают resolved: в статистике он
// считается неактивным (см. GetStatisticsWorkload/ByOwner), поэтому список
// drill-down обязан повторять именно это деление.
var (
	statisticsDrilldownActive = []models.TicketStatus{
		models.StatusOpen, models.StatusInProgress, models.StatusPending, models.StatusOnHold,
	}
	statisticsDrilldownClosed = []models.TicketStatus{
		models.StatusResolved, models.StatusClosed, models.StatusCancelled,
	}
)

// GetStatisticsTickets возвращает заявки, стоящие за сектором двухкольцевой
// диаграммы («Нагрузка исполнителей» / «Задачи от заказчиков»). Срез/уточнения и
// период берутся те же, что у агрегатов (statisticsWhere + created_at), плюс
// человек (исполнитель или заказчик) и статусная группа кольца — поэтому список
// совпадает с числом в секции.
func (r *StatisticsRepo) GetStatisticsTickets(ctx context.Context, q models.StatisticsTicketsQuery) ([]*models.Ticket, int, error) {
	if q.Filter == nil {
		return nil, 0, MapError(fmt.Errorf("statistics tickets: nil filter"))
	}

	w := &whereBuilder{}
	w.statisticsWhere(models.StatisticsQuery{Scope: q.Scope, Filter: q.Filter})

	fromPh, toPh := w.rangeArgs(q.Filter.From, q.Filter.To)
	w.add(fmt.Sprintf("t.created_at >= %s AND t.created_at < %s", fromPh, toPh))

	// Добавляем WHERE условие в зависимости от измерения
	switch q.Dimension {
	case "category":
		if q.BucketID != nil {
			w.idx++
			w.add(fmt.Sprintf("c.id = $%d", w.idx))
			w.args = append(w.args, *q.BucketID)
		}
	case "site":
		if q.BucketID != nil {
			w.idx++
			w.add(fmt.Sprintf("s.id = $%d", w.idx))
			w.args = append(w.args, *q.BucketID)
		}
	case "group":
		if q.BucketID != nil {
			w.idx++
			w.add(fmt.Sprintf("g.id = $%d", w.idx))
			w.args = append(w.args, *q.BucketID)
		}
	case "owner":
		if q.OwnerNone {
			w.add("t.owner_id IS NULL")
		} else {
			eq(w, "t.owner_id", q.PersonID)
		}
	default: // assignee
		eq(w, "t.assignee_id", q.PersonID)
	}

	switch q.StatusGroup {
	case "active":
		w.statuses(nil, statisticsDrilldownActive)
	case "closed":
		w.statuses(nil, statisticsDrilldownClosed)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	w.idx++
	limitPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, limit)

	w.idx++
	offsetPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, offset)

	// Убрали лишний %s в начале шаблона (было `%s %s WHERE ...`)
	query := fmt.Sprintf(
		`%s WHERE %s ORDER BY t.created_at DESC LIMIT %s OFFSET %s`,
		ticketSelectBase(),
		strings.Join(w.clauses, " AND "),
		limitPh,
		offsetPh,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.Ticket{}
	total := 0
	for rows.Next() {
		ticket, rowTotal, err := scanTicketRow(rows)
		if err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		total = rowTotal
		data = append(data, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}

	return data, total, nil
}

// GetKpiTickets возвращает заявки для KPI-карточки. CardKey задаёт тип выборки:
// total — созданные за период (created_at), active — активные на текущий момент,
// overdue — просроченные (due_date < NOW()), resolved — решённые за период (resolved_at).
func (r *StatisticsRepo) GetKpiTickets(ctx context.Context, q models.StatisticsKpiQuery) ([]*models.Ticket, int, error) {
	if q.Filter == nil {
		return nil, 0, MapError(fmt.Errorf("kpi tickets: nil filter"))
	}

	w := &whereBuilder{}
	w.statisticsWhere(models.StatisticsQuery{Scope: q.Scope, Filter: q.Filter})

	switch q.CardKey {
	case models.KpiTotal:
		fromPh, toPh := w.rangeArgs(q.Filter.From, q.Filter.To)
		w.add(fmt.Sprintf("t.created_at >= %s AND t.created_at < %s", fromPh, toPh))
	case models.KpiActive:
		w.statuses(nil, []models.TicketStatus{
			models.StatusOpen, models.StatusInProgress, models.StatusPending, models.StatusOnHold,
		})
	case models.KpiOverdue:
		w.add("t.due_date IS NOT NULL AND t.due_date < NOW()")
		w.statuses(nil, []models.TicketStatus{
			models.StatusOpen, models.StatusInProgress, models.StatusPending, models.StatusOnHold,
		})
	case models.KpiResolved:
		fromPh, toPh := w.rangeArgs(q.Filter.From, q.Filter.To)
		w.add(fmt.Sprintf("t.resolved_at >= %s AND t.resolved_at < %s", fromPh, toPh))
	default:
		return nil, 0, MapError(fmt.Errorf("unknown kpi card key: %q", q.CardKey))
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	w.idx++
	limitPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, limit)
	w.idx++
	offsetPh := fmt.Sprintf("$%d", w.idx)
	w.args = append(w.args, offset)

	query := fmt.Sprintf(`%s WHERE %s ORDER BY t.created_at DESC LIMIT %s OFFSET %s`,
		ticketSelectBase(), strings.Join(w.clauses, " AND "), limitPh, offsetPh,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, 0, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.Ticket{}
	total := 0
	for rows.Next() {
		ticket, rowTotal, err := scanTicketRow(rows)
		if err != nil {
			return nil, 0, MapError(fmt.Errorf("scan row error: %w", err))
		}
		total = rowTotal
		data = append(data, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, MapError(fmt.Errorf("rows iteration error: %w", err))
	}
	return data, total, nil
}
