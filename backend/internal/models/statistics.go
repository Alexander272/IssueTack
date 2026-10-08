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
type StatisticsSummary struct {
	Total        int `json:"total"`
	Active       int `json:"active"`
	Overdue      int `json:"overdue"`
	Resolved     int `json:"resolved"`
	TotalPrev    int `json:"totalPrev"`
	ResolvedPrev int `json:"resolvedPrev"`
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

// WorkloadBucket — нагрузка на исполнителя за период.
type WorkloadBucket struct {
	UserID uuid.UUID `json:"userId"`
	Name   string    `json:"name"`
	Active int       `json:"active"`
	Total  int       `json:"total"`
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
