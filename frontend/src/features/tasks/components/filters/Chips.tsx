import { useMemo, type FC } from 'react'
import { Box, Chip } from '@mui/material'

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
	userOptions: Option[]
}

const PRIORITY_OPTIONS = Object.entries(PRIORITY_MAP).map(([value, info]) => ({
	value,
	label: info.label,
}))

export const Chips: FC<Props> = ({ filters, onChange, siteOptions, userOptions }) => {
	const activeChips = useMemo(() => {
		const chips: { key: string; label: string; onClear: () => void }[] = []

		if (filters.ticketNumber) {
			chips.push({
				key: 'ticketNumber',
				label: `№: ${filters.ticketNumber}`,
				onClear: () => onChange({ ticketNumber: undefined }),
			})
		}
		if (filters.ownerId) {
			const name = userOptions.find(u => u.id === filters.ownerId)?.label
			chips.push({
				key: 'ownerId',
				label: `Заказчик: ${name ?? filters.ownerId}`,
				onClear: () => onChange({ ownerId: undefined }),
			})
		}
		if (filters.siteIds?.length) {
			const names = filters.siteIds.map(id => siteOptions.find(s => s.id === id)?.label ?? id).join(', ')
			chips.push({
				key: 'siteIds',
				label: `Площадка: ${names}`,
				onClear: () => onChange({ siteIds: undefined }),
			})
		}
		if (filters.dueDateFrom || filters.dueDateTo) {
			chips.push({
				key: 'dueDate',
				label: `Срок: ${filters.dueDateFrom || '…'} — ${filters.dueDateTo || '…'}`,
				onClear: () => onChange({ dueDateFrom: undefined, dueDateTo: undefined }),
			})
		}
		if (filters.priorities?.length) {
			const names = filters.priorities.map(p => PRIORITY_OPTIONS.find(o => o.value === p)?.label ?? p).join(', ')
			chips.push({
				key: 'priorities',
				label: `Приоритет: ${names}`,
				onClear: () => onChange({ priorities: undefined }),
			})
		}
		if (filters.assigneeId) {
			const name = userOptions.find(u => u.id === filters.assigneeId)?.label
			chips.push({
				key: 'assigneeId',
				label: `Назначено: ${name ?? filters.assigneeId}`,
				onClear: () => onChange({ assigneeId: undefined }),
			})
		}
		if (filters.statuses?.length) {
			const names = filters.statuses.map(s => STATUS_OPTIONS.find(o => o.value === s)?.label ?? s).join(', ')
			chips.push({
				key: 'statuses',
				label: `Статус: ${names}`,
				onClear: () => onChange({ statuses: undefined }),
			})
		}

		return chips
	}, [filters, onChange, userOptions, siteOptions])

	if (activeChips.length === 0) return null

	return (
		<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
			{activeChips.map(chip => (
				<Chip
					key={chip.key}
					label={chip.label}
					onDelete={chip.onClear}
					size='small'
					sx={{
						bgcolor: '#ddf4ff',
						color: '#0969da',
						fontWeight: 500,
						fontSize: '0.8125rem',
						'& .MuiChip-deleteIcon': { color: '#0969da', fontSize: 16 },
					}}
				/>
			))}
		</Box>
	)
}
