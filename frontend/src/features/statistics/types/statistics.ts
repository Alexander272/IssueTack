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
	workload: IWorkloadBucket[]
}

export type StatisticsGranularity = 'day' | 'week' | 'month'

export interface IStatisticsFilter {
	from: string
	to: string
	granularity?: StatisticsGranularity
	assigneeId?: string[]
	categoryId?: string[]
	groupId?: string[]
	siteId?: string[]
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