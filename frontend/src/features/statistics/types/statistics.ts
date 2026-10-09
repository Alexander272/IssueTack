export interface IStatisticsSummary {
	total: number
	active: number
	overdue: number
	resolved: number
	totalPrev: number
	resolvedPrev: number
}

export interface IStatisticsBucket {
	id: string
	name: string
	count: number
}

export interface IStatusBucket {
	status: string
	count: number
}

export interface IWorkloadBucket {
	userId: string
	name: string
	active: number
	total: number
}

// IByOwnerBucket — разрез по заказчику. active/total нужны для двухкольцевой
// диаграммы: внутреннее кольцо — total, внешнее — активные и остальные.
export interface IByOwnerBucket {
	id: string
	name: string
	total: number
	active: number
}

export interface ITrendPoint {
	date: string
	created: number
	resolved: number
}

export interface ITicketStatistics {
	summary: IStatisticsSummary
	trend: ITrendPoint[]
	byStatus: IStatusBucket[]
	byCategory: IStatisticsBucket[]
	byGroup: IStatisticsBucket[]
	bySite: IStatisticsBucket[]
	byOwner: IByOwnerBucket[]
	workload: IWorkloadBucket[]
}

export type StatisticsGranularity = 'day' | 'week' | 'month'

// Разрез drill-down статистики: кого показать — исполнителя или заказчика.
export type StatisticsDrilldownDimension = 'assignee' | 'owner'

// Кольцо диаграммы: пусто — внутреннее (все заявки), active — внешнее активное,
// closed — внешнее закрытое. Значения совпадают с параметром `ring` API.
export type StatisticsDrilldownRing = '' | 'active' | 'closed'

export interface IStatisticsFilter {
	from: string
	to: string
	granularity?: StatisticsGranularity
	assigneeId?: string[]
	categoryId?: string[]
	groupId?: string[]
	siteId?: string[]
}

// IStatisticsTicketsFilter — запрос списка заявок сектора диаграммы: тот же
// период и уточнения, что у агрегатов, плюс разрез (dim), человек (id) и кольцо.
export interface IStatisticsTicketsFilter extends IStatisticsFilter {
	dim: StatisticsDrilldownDimension
	id: string
	ring?: Exclude<StatisticsDrilldownRing, ''>
	limit?: number
	offset?: number
}

// IStatisticsRefinements — уточнения выборки без периода: то, чем управляет панель
// фильтров и что хранится в URL. Пустой массив означает «фильтр не задан».
export interface IStatisticsRefinements {
	assigneeId: string[]
	categoryId: string[]
	groupId: string[]
	siteId: string[]
}

export const emptyRefinements = (): IStatisticsRefinements => ({
	assigneeId: [],
	categoryId: [],
	groupId: [],
	siteId: [],
})