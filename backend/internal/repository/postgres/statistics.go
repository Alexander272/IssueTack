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

	query := fmt.Sprintf(`SELECT
			COUNT(*) FILTER (WHERE t.created_at >= %s AND t.created_at < %s),
			COUNT(*) FILTER (WHERE t.status IN ('open', 'in_progress', 'pending', 'on_hold')),
			COUNT(*) FILTER (WHERE t.due_date IS NOT NULL AND t.due_date < NOW() AND t.status NOT IN ('closed', 'cancelled')),
			COUNT(*) FILTER (WHERE t.resolved_at >= %s AND t.resolved_at < %s),
			COUNT(*) FILTER (WHERE t.created_at >= %s AND t.created_at < %s),
			COUNT(*) FILTER (WHERE t.resolved_at >= %s AND t.resolved_at < %s)
		FROM %s t
		WHERE %s`,
		fromPh, toPh, fromPh, toPh, prevFromPh, prevToPh, prevFromPh, prevToPh,
		Tables.Tickets, strings.Join(w.clauses, " AND "),
	)

	summary := &models.StatisticsSummary{}
	if err := r.db.QueryRow(ctx, query, w.args...).Scan(
		&summary.Total, &summary.Active, &summary.Overdue, &summary.Resolved,
		&summary.TotalPrev, &summary.ResolvedPrev,
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

// GetStatisticsWorkload возвращает нагрузку по исполнителям за период: total —
// заявки, созданные за период и назначенные исполнителю, active — из них ещё активные.
func (r *StatisticsRepo) GetStatisticsWorkload(ctx context.Context, q models.StatisticsQuery) ([]*models.WorkloadBucket, error) {
	from, to := q.Filter.From, q.Filter.To
	w := &whereBuilder{}
	w.statisticsWhere(q)
	fromPh, toPh := w.rangeArgs(from, to)

	// 0 (или отрицательное) — без LIMIT: сортировка нужна только для порядка строк,
	// обрезать хвост нельзя (см. комментарий в GetStatisticsByOwner).
	limitClause := ""
	if q.Limit > 0 {
		w.idx++
		limitPh := fmt.Sprintf("$%d", w.idx)
		w.args = append(w.args, q.Limit)
		limitClause = fmt.Sprintf(" LIMIT %s", limitPh)
	}

	query := fmt.Sprintf(`SELECT
			u.id::text,
			COALESCE(NULLIF(TRIM(COALESCE(u.last_name, '') || ' ' || COALESCE(u.first_name, '')), ''), u.username) AS name,
			COUNT(*) FILTER (WHERE t.status IN ('open', 'in_progress', 'pending', 'on_hold')) AS active,
			COUNT(*) AS total
		FROM %s t
		JOIN %s u ON t.assignee_id = u.id
		WHERE %s AND t.created_at >= %s AND t.created_at < %s
		GROUP BY u.id, u.last_name, u.first_name, u.username
		ORDER BY total DESC, active DESC, name ASC%s`,
		Tables.Tickets, Tables.Users, strings.Join(w.clauses, " AND "), fromPh, toPh, limitClause,
	)

	rows, err := r.db.Query(ctx, query, w.args...)
	if err != nil {
		return nil, MapError(fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()

	data := []*models.WorkloadBucket{}
	for rows.Next() {
		bucket := &models.WorkloadBucket{}
		var idStr string
		if err := rows.Scan(&idStr, &bucket.Name, &bucket.Active, &bucket.Total); err != nil {
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
