import type { TicketStatus, Priority } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'

export interface FilterValues {
	sort: string
	search: string
	groupBy: GroupByField
	groupEnabled: boolean

	ticketNumber?: string
	ownerId?: string
	siteIds?: string[]
	dueDateFrom?: string
	dueDateTo?: string
	priorities?: Priority[]
	assigneeId?: string
	statuses?: TicketStatus[]
}

export interface TaskFiltersProps {
	filters: FilterValues
	onChange: (patch: Partial<FilterValues>) => void
	onReset: () => void
}
