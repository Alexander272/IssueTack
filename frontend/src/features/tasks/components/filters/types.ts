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
	// equipmentId — фильтр по оборудованию: deep-link (?equipmentId=…) или выбор в
	// фильтре (Popover/чип). Deep-link задаёт из URL, выбор — из UI.
	equipmentId?: string
}

export interface TaskFiltersProps {
	filters: FilterValues
	onChange: (patch: Partial<FilterValues>) => void
	onReset: () => void
	hideGrouping?: boolean
	// statusOptions — допустимые статусы текущей вкладки (см. statusOptionsFor);
	// без него Popover предлагает все статусы, в том числе дающие пустой список.
	statusOptions: TicketStatus[]
	hideFilters?: boolean
	// equipmentLabel — название единицы для чипа фильтра по оборудованию.
	equipmentLabel?: string
	// equipmentOptions — справочник единиц для выбора фильтра по оборудованию.
	equipmentOptions?: { id: string; label: string }[]
}
