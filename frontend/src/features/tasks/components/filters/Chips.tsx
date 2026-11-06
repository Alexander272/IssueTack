import { useMemo, type FC } from 'react'
import { Box } from '@mui/material'

import type { FilterValues } from './types'
import type { Priority } from '../../types/task'
import { useAppSelector } from '@/hooks/redux'
import { STATUS_OPTIONS, PRIORITY_MAP } from '../../constants/taskMaps'
import { getHasNoGroups } from '@/features/user/userSlice'
import { FilterChip } from '@/components/FilterChip'

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
	equipmentLabel?: string
	equipmentOptions?: Option[]
}

// Явно типизируем value, чтобы избежать неявного string
const PRIORITY_OPTIONS: { value: Priority; label: string }[] = Object.entries(PRIORITY_MAP).map(([value, info]) => ({
	value: value as Priority,
	label: info.label,
}))

export const Chips: FC<Props> = ({
	filters,
	onChange,
	siteOptions,
	ownerOptions,
	assigneeOptions,
	equipmentLabel,
	equipmentOptions,
}) => {
	const hasNoGroups = useAppSelector(getHasNoGroups)

	const activeChips = useMemo(() => {
		const chips: { key: keyof FilterValues | string; prefix: string; values: string[]; onClear: () => void }[] = []

		if (filters.ticketNumber) {
			chips.push({
				key: 'ticketNumber',
				prefix: '№',
				values: [filters.ticketNumber],
				onClear: () => onChange({ ticketNumber: undefined }),
			})
		}

		if (filters.equipmentId) {
			// Используем опциональную цепочку вместо (arr ?? []).find
			const label =
				equipmentOptions?.find(e => e.id === filters.equipmentId)?.label ??
				equipmentLabel ??
				filters.equipmentId
			chips.push({
				key: 'equipmentId',
				prefix: 'Оборудование',
				values: [label],
				onClear: () => onChange({ equipmentId: undefined }),
			})
		}

		if (filters.ownerId) {
			const label = ownerOptions.find(u => u.id === filters.ownerId)?.label ?? filters.ownerId
			chips.push({
				key: 'ownerId',
				prefix: 'Заказчик',
				values: [label],
				onClear: () => onChange({ ownerId: undefined }),
			})
		}

		if (filters.siteIds?.length) {
			// TypeScript знает, что siteIds существует и имеет длину > 0 благодаря проверке выше
			const labels = filters.siteIds.map(id => siteOptions.find(s => s.id === id)?.label ?? id)
			chips.push({
				key: 'siteIds',
				prefix: 'Площадка',
				values: labels,
				onClear: () => onChange({ siteIds: undefined }),
			})
		}

		if (!hasNoGroups && (filters.dueDateFrom || filters.dueDateTo)) {
			chips.push({
				key: 'dueDate',
				prefix: 'Срок',
				values: [`${filters.dueDateFrom || '…'} — ${filters.dueDateTo || '…'}`],
				onClear: () => onChange({ dueDateFrom: undefined, dueDateTo: undefined }),
			})
		}

		if (!hasNoGroups && filters.priorities?.length) {
			const labels = filters.priorities.map(p => PRIORITY_OPTIONS.find(o => o.value === p)?.label ?? p)
			chips.push({
				key: 'priorities',
				prefix: 'Приоритет',
				values: labels,
				onClear: () => onChange({ priorities: undefined }),
			})
		}

		if (filters.assigneeId) {
			const label = assigneeOptions.find(u => u.id === filters.assigneeId)?.label ?? filters.assigneeId
			chips.push({
				key: 'assigneeId',
				prefix: 'Назначено',
				values: [label],
				onClear: () => onChange({ assigneeId: undefined }),
			})
		}

		if (filters.statuses?.length) {
			const labels = filters.statuses.map(s => STATUS_OPTIONS.find(o => o.value === s)?.label ?? s)
			chips.push({
				key: 'statuses',
				prefix: 'Статус',
				values: labels,
				onClear: () => onChange({ statuses: undefined }),
			})
		}

		return chips
	}, [filters, onChange, ownerOptions, assigneeOptions, siteOptions, equipmentOptions, equipmentLabel, hasNoGroups])

	if (activeChips.length === 0) return null

	return (
		<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
			{activeChips.map(chip => (
				<FilterChip key={chip.key} prefix={chip.prefix} values={chip.values} onClear={chip.onClear} />
			))}
		</Box>
	)
}
