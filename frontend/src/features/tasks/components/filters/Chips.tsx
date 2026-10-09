import { useMemo, type FC } from 'react'
import { Box } from '@mui/material'

import { FilterChip } from '@/components/FilterChip'
import type { FilterValues } from './types'
import { STATUS_OPTIONS, PRIORITY_MAP } from '../../constants/taskMaps'

interface Option {
	id: string
	label: string
}

interface Props {
	filters: FilterValues
	onChange: (patch: Partial<FilterValues>) => void
	siteOptions: Option[]
	ownerOptions: Option[]
	assigneeOptions: Option[]
}

const PRIORITY_OPTIONS = Object.entries(PRIORITY_MAP).map(([value, info]) => ({
	value,
	label: info.label,
}))

export const Chips: FC<Props> = ({ filters, onChange, siteOptions, ownerOptions, assigneeOptions }) => {
	const activeChips = useMemo(() => {
		const chips: { key: string; prefix: string; values: string[]; onClear: () => void }[] = []

		if (filters.ticketNumber) {
			chips.push({
				key: 'ticketNumber',
				prefix: '№',
				values: [filters.ticketNumber],
				onClear: () => onChange({ ticketNumber: undefined }),
			})
		}
		if (filters.ownerId) {
			const name = ownerOptions.find(u => u.id === filters.ownerId)?.label
			chips.push({
				key: 'ownerId',
				prefix: 'Заказчик',
				values: [name ?? filters.ownerId],
				onClear: () => onChange({ ownerId: undefined }),
			})
		}
		if (filters.siteIds?.length) {
			const names = filters.siteIds.map(id => siteOptions.find(s => s.id === id)?.label ?? id)
			chips.push({
				key: 'siteIds',
				prefix: 'Площадка',
				values: names,
				onClear: () => onChange({ siteIds: undefined }),
			})
		}
		if (filters.dueDateFrom || filters.dueDateTo) {
			chips.push({
				key: 'dueDate',
				prefix: 'Срок',
				values: [`${filters.dueDateFrom || '…'} — ${filters.dueDateTo || '…'}`],
				onClear: () => onChange({ dueDateFrom: undefined, dueDateTo: undefined }),
			})
		}
		if (filters.priorities?.length) {
			const names = filters.priorities.map(p => PRIORITY_OPTIONS.find(o => o.value === p)?.label ?? p)
			chips.push({
				key: 'priorities',
				prefix: 'Приоритет',
				values: names,
				onClear: () => onChange({ priorities: undefined }),
			})
		}
		if (filters.assigneeId) {
			const name = assigneeOptions.find(u => u.id === filters.assigneeId)?.label
			chips.push({
				key: 'assigneeId',
				prefix: 'Назначено',
				values: [name ?? filters.assigneeId],
				onClear: () => onChange({ assigneeId: undefined }),
			})
		}
		if (filters.statuses?.length) {
			const names = filters.statuses.map(s => STATUS_OPTIONS.find(o => o.value === s)?.label ?? s)
			chips.push({
				key: 'statuses',
				prefix: 'Статус',
				values: names,
				onClear: () => onChange({ statuses: undefined }),
			})
		}

		return chips
	}, [filters, onChange, ownerOptions, assigneeOptions, siteOptions])

	if (activeChips.length === 0) return null

	return (
		<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
			{activeChips.map(chip => (
				<FilterChip key={chip.key} prefix={chip.prefix} values={chip.values} onClear={chip.onClear} />
			))}
		</Box>
	)
}
