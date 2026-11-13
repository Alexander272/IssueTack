package models

import (
	"time"

	"github.com/google/uuid"
)

// StatisticsFilter — вход запроса статистики заявок. Реалм задаёт область, актор —
// носитель роли, из которой вычисляется срез видимых заявок (StatisticsScope).
//
// AssigneeIDs/CategoryIDs/GroupIDs/SiteIDs — необязательные уточнения выборки.
// Они НЕ расширяют срез доступа: репозиторий добавляет их как дополнительный AND
// поверх StatisticsScope. Поэтому менеджер, задав чужого исполнителя или группу,
// получит пустую выборку, а не чужие заявки.
type StatisticsFilter struct {
	Actor       *Actor
	RealmID     uuid.UUID
	From        time.Time
	To          time.Time
	Granularity string // day|week|month; пусто — авто по длине периода

	AssigneeIDs []uuid.UUID
	CategoryIDs []uuid.UUID
	GroupIDs    []uuid.UUID
	SiteIDs     []uuid.UUID
}

// StatisticsScope — вычисленный срез заявок, видимых актору. Ровно одно измерение
// (AllRealm / GroupIDs / AssigneeID / RequesterID) определяет выборку; AllRealm
// имеет приоритет, дальше — менеджер управляемых групп, затем исполнитель, затем заявитель.
type StatisticsScope struct {
	RealmID     uuid.UUID
	AllRealm    bool
	GroupIDs    []uuid.UUID
	AssigneeID  *uuid.UUID
	RequesterID *uuid.UUID
}

// IsManagerial сообщает, доступны ли актору «управленческие» разрезы статистики:
// разбивка по группам, по заказчикам и нагрузка по исполнителям. True для начальника
// области и менеджеров групп.
func (s StatisticsScope) IsManagerial() bool {
	return s.AllRealm || len(s.GroupIDs) > 0
}

// StatisticsSummary — агрегаты верхнего уровня. Total и Resolved привязаны к периоду
// (created_at / resolved_at в диапазоне), Active и Overdue характеризуют текущее
// состояние видимых заявок и от периода не зависят. TotalPrev и ResolvedPrev —
// те же периодные метрики за предыдущее окно той же длины, для расчёта дельт.
// Subtasks — подзадачи в разрезе KPI-карточки (свои цифры у каждой).
type StatisticsSummary struct {
	Total        int                       `json:"total"`
	Active       int                       `json:"active"`
	Overdue      int                       `json:"overdue"`
	Resolved     int                       `json:"resolved"`
	TotalPrev    int                       `json:"totalPrev"`
	ResolvedPrev int                       `json:"resolvedPrev"`
	Subtasks     StatisticsSummarySubtasks `json:"subtasks"`
}

// StatisticsSubtasks — подзадачи одной KPI-карточки: WithSubtasks — сколько заявок
// карточки имеют хотя бы одну подзадачу, TotalSubtasks — сколько подзадач у этих
// заявок суммарно.
type StatisticsSubtasks struct {
	WithSubtasks  int `json:"withSubtasks"`
	TotalSubtasks int `json:"totalSubtasks"`
}

// StatisticsSummarySubtasks — блок подзадач для каждой KPI-карточки. Цифры не общие:
// у «Просроченные» считаются только просроченные заявки, у «Решено» — решённые за
// период и т.д. Отдельная структура, а не карта, чтобы блок всегда был заполнен и
// фронт не проверял nil.
type StatisticsSummarySubtasks struct {
	Total    StatisticsSubtasks `json:"total"`
	Active   StatisticsSubtasks `json:"active"`
	Overdue  StatisticsSubtasks `json:"overdue"`
	Resolved StatisticsSubtasks `json:"resolved"`
}

// StatisticsQuery — вход запроса к слою данных статистики: вычисленный срез доступа
// (Scope), применённые уточнения фильтра (Filter) и параметры конкретного разреза.
// Один структурный аргумент вместо длинного списка позиционных (scope, filter, dim,
// from, to, limit): период берётся из Filter и не дублируется отдельными полями.
type StatisticsQuery struct {
	Scope       StatisticsScope
	Filter      *StatisticsFilter
	Dim         string // разрез для GetStatisticsByDimension: category|group|site
	Limit       int    // топ-N; у byOwner/workload 0 — без LIMIT в SQL (нужен полный хвост для «Прочие»), у GetStatisticsByDimension 0 заменяется на 20
	Granularity string // разрешённая гранулярность динамики: day|week|month
}

// WithDimension возвращает копию запроса с заданным разрезом: разрезы отличаются
// только полем Dim, поэтому остальные поля (срез, фильтр, лимит, гранулярность)
// переиспользуются как есть.
func (q StatisticsQuery) WithDimension(dim string) StatisticsQuery {
	q.Dim = dim
	return q
}

// StatisticsTicketsQuery — вход drill-down статистики: клик по сектору
// двухкольцевой диаграммы («Нагрузка исполнителей» / «Задачи от заказчиков»)
// или по столбцу графика категорий/площадок/групп.
// Scope и Filter те же, что у агрегатов, поэтому список совпадает с числом в
// секции — тот же период и те же уточнения фильтра. Dimension задаёт поле
// группировки: assignee/owner/category/site/group.
type StatisticsTicketsQuery struct {
	Scope     StatisticsScope
	Filter    *StatisticsFilter
	Dimension string // assignee|owner|category|site|group
	PersonID  *uuid.UUID
	OwnerNone bool // owner_id IS NULL — бакет «Без заказчика»
	BucketID  *uuid.UUID // category_id/site_id/group_id — для не-person измерений
	// StatusGroup — кольцо диаграммы: "" (внутреннее — все заявки), active
	// (open/in_progress/pending/on_hold), closed (resolved/closed/cancelled).
	StatusGroup string
	Limit       int
	Offset      int
}

// StatisticsBucket — одна строка разреза (категория, группа или площадка).
type StatisticsBucket struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Count int       `json:"count"`
}

// StatusBucket — строка разреза по статусу заявки. В отличие от StatisticsBucket
// идентификатор здесь строковый (сам статус), а не uuid.
type StatusBucket struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// StatisticsBreakdownBucket — строка разреза, у которой известно и общее число
// заявок (Total), и число ещё активных (Active): два нужны для двухкольцевой
// диаграммы (внутреннее кольцо — Total, внешнее — активные/неактивные).
// ID пустой, когда сущность отсутствует (заявка без заказчика).
type StatisticsBreakdownBucket struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Total  int       `json:"total"`
	Active int       `json:"active"`
}

// WorkloadBucket — нагрузка на исполнителя за период: два столбца (active/closed),
// каждый — stack из задач и подзадач. Исполнитель подзадачи = исполнитель родительской заявки.
type WorkloadBucket struct {
	UserID         uuid.UUID `json:"userId"`
	Name           string    `json:"name"`
	ActiveTasks    int       `json:"activeTasks"`
	ActiveSubtasks int       `json:"activeSubtasks"`
	ClosedTasks    int       `json:"closedTasks"`
	ClosedSubtasks int       `json:"closedSubtasks"`
}

// TrendPoint — точка динамики заявок: создано и решено за бакет.
type TrendPoint struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

// TicketStatistics — ответ страницы статистики. Все срезы всегда не-nil: фронт
// считает их массивами, а nil сериализуется в JSON null (см. AGENTS.md).
type TicketStatistics struct {
	Summary    StatisticsSummary            `json:"summary"`
	Trend      []*TrendPoint                `json:"trend"`
	ByStatus   []*StatusBucket              `json:"byStatus"`
	ByCategory []*StatisticsBucket          `json:"byCategory"`
	ByGroup    []*StatisticsBucket          `json:"byGroup"`
	BySite     []*StatisticsBucket          `json:"bySite"`
	ByOwner    []*StatisticsBreakdownBucket `json:"byOwner"`
	Workload   []*WorkloadBucket            `json:"workload"`
}

// KpiCardKey — тип KPI-карточки: total (новые за период), active (активные на конец периода),
// overdue (просроченные), resolved (решённые за период).
type KpiCardKey string

const (
	KpiTotal    KpiCardKey = "total"
	KpiActive   KpiCardKey = "active"
	KpiOverdue  KpiCardKey = "overdue"
	KpiResolved KpiCardKey = "resolved"
)

// StatisticsKpiQuery — вход запроса списка заявок для KPI-карточки.
// Scope и Filter те же, что у агрегатов, CardKey задаёт тип выборки.
type StatisticsKpiQuery struct {
	Scope   StatisticsScope
	Filter  *StatisticsFilter
	CardKey KpiCardKey
	Limit   int
	Offset  int
}
