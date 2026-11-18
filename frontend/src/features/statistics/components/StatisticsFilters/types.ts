import type { IStatisticsRefinements } from '../../types/statistics'

export interface Option {
	id: string
	label: string
}

export interface Props {
	value: IStatisticsRefinements
	onChange: (patch: Partial<IStatisticsRefinements>) => void
	onReset: () => void
	isManager: boolean
}

export interface FilterSelectProps {
	options: Option[]
	values: string[]
	onChange: (ids: string[]) => void
}

export interface PopoverProps {
	open: boolean
	anchorEl: HTMLElement | null
	onClose: () => void
	initial: IStatisticsRefinements
	onApply: (patch: Partial<IStatisticsRefinements>) => void
	isManager: boolean
	assigneeOptions: Option[]
	categoryOptions: Option[]
	groupOptions: Option[]
	siteOptions: Option[]
}

export interface FilterPopoverBodyProps {
	initial: IStatisticsRefinements
	onClose: () => void
	isManager: boolean
	assigneeOptions: Option[]
	categoryOptions: Option[]
	groupOptions: Option[]
	siteOptions: Option[]
	onApply: (patch: Partial<IStatisticsRefinements>) => void
}
