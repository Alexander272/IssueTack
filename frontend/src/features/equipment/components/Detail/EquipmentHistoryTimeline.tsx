import { type ElementType, type FC, useMemo, useState } from 'react'
import { Box, Chip, Stack, Typography } from '@mui/material'
import {
	ArchiveRestoreIcon,
	ClockIcon,
	MapPinIcon,
	PackageMinusIcon,
	PencilIcon,
	PlusIcon,
	RefreshCwIcon,
	Trash2Icon,
	Undo2Icon,
	UserPlusIcon,
	WrenchIcon,
} from 'lucide-mui'

import type { EquipmentEventType, IEquipment } from '../../types/equipment'
import { getEquipmentEventLabel } from '../../types/equipment'
import {
	useGetEquipmentAssignmentsQuery,
	useGetEquipmentCompositionHistoryQuery,
	useGetEquipmentEventsQuery,
} from '../../equipmentApiSlice'

type Group = 'composition' | 'status' | 'transfer'

type Entry = {
	id: string
	date: string
	title: string
	subtitle?: string
	Icon: ElementType
	color: string
	group: Group
}

const GROUPS: { value: Group | 'all'; label: string }[] = [
	{ value: 'all', label: 'Все события' },
	{ value: 'composition', label: 'Состав' },
	{ value: 'status', label: 'Статус' },
	{ value: 'transfer', label: 'Перемещения' },
]

const EVENT_META: Record<EquipmentEventType, { Icon: ElementType; color: string; group: Group }> = {
	created: { Icon: PlusIcon, color: '#2563eb', group: 'status' },
	updated: { Icon: PencilIcon, color: '#6b7280', group: 'status' },
	assigned: { Icon: UserPlusIcon, color: '#16a34a', group: 'transfer' },
	unassigned: { Icon: ArchiveRestoreIcon, color: '#2563eb', group: 'transfer' },
	moved: { Icon: MapPinIcon, color: '#2563eb', group: 'transfer' },
	status_changed: { Icon: RefreshCwIcon, color: '#ea580c', group: 'status' },
	component_added: { Icon: PlusIcon, color: '#7c3aed', group: 'composition' },
	component_removed: { Icon: PackageMinusIcon, color: '#dc2626', group: 'composition' },
	component_replaced: { Icon: Undo2Icon, color: '#7c3aed', group: 'composition' },
	repaired: { Icon: WrenchIcon, color: '#ea580c', group: 'status' },
	written_off: { Icon: Trash2Icon, color: '#dc2626', group: 'status' },
}

const fmtDateTime = (d: string | null) =>
	d
		? new Date(d).toLocaleString('ru-RU', {
				day: '2-digit',
				month: '2-digit',
				year: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
			})
		: '—'

type Props = {
	unit: IEquipment
}

export const EquipmentHistoryTimeline: FC<Props> = ({ unit }) => {
	const [group, setGroup] = useState<Group | 'all'>('all')

	const { data: assignmentsData } = useGetEquipmentAssignmentsQuery(unit.id)
	const { data: eventsData } = useGetEquipmentEventsQuery({ id: unit.id, limit: 100 })
	const { data: compositionData } = useGetEquipmentCompositionHistoryQuery(unit.id, { skip: !unit.isKit })

	const entries = useMemo<Entry[]>(() => {
		const list: Entry[] = []

		;(eventsData?.data ?? []).forEach(e => {
			const meta = EVENT_META[e.eventType]
			if (!meta) return
			list.push({
				id: `event-${e.id}`,
				date: e.createdAt,
				title: getEquipmentEventLabel(e.eventType),
				subtitle: e.comment || undefined,
				Icon: meta.Icon,
				color: meta.color,
				group: meta.group,
			})
		})

		;(assignmentsData?.data ?? []).forEach(a => {
			list.push({
				id: `assign-${a.id}`,
				date: a.assignedAt,
				title: 'Выдано',
				subtitle: [a.holderName, a.location].filter(Boolean).join(' · ') || undefined,
				Icon: UserPlusIcon,
				color: '#16a34a',
				group: 'transfer',
			})
			if (a.unassignedAt) {
				list.push({
					id: `unassign-${a.id}`,
					date: a.unassignedAt,
					title: 'Возвращено на склад',
					subtitle: a.comment || undefined,
					Icon: ArchiveRestoreIcon,
					color: '#2563eb',
					group: 'transfer',
				})
			}
		})

		;(compositionData?.data ?? []).forEach(c => {
			list.push({
				id: `comp-in-${c.id}`,
				date: c.startedAt,
				title: 'Компонент добавлен в комплект',
				subtitle: `#${c.componentId.slice(0, 8)}${c.reason ? ` · ${c.reason}` : ''}`,
				Icon: PlusIcon,
				color: '#7c3aed',
				group: 'composition',
			})
			if (c.endedAt) {
				list.push({
					id: `comp-out-${c.id}`,
					date: c.endedAt,
					title: 'Компонент исключён из комплекта',
					subtitle: `#${c.componentId.slice(0, 8)}${c.reason ? ` · ${c.reason}` : ''}`,
					Icon: PackageMinusIcon,
					color: '#dc2626',
					group: 'composition',
				})
			}
		})

		return list
			.filter(e => group === 'all' || e.group === group)
			.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
	}, [assignmentsData, eventsData, compositionData, group])

	return (
		<Stack spacing={2}>
			<Stack direction='row' spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
				{GROUPS.map(g => (
					<Chip
						key={g.value}
						label={g.label}
						size='small'
						onClick={() => setGroup(g.value)}
						color={group === g.value ? 'primary' : 'default'}
						variant={group === g.value ? 'filled' : 'outlined'}
						sx={{ cursor: 'pointer' }}
					/>
				))}
			</Stack>

			{entries.length === 0 ? (
				<Box sx={{ py: 8, textAlign: 'center', color: '#9ca3af' }}>
					<ClockIcon sx={{ fontSize: 40, mb: 1, opacity: 0.5 }} />
					<Typography variant='body2'>Событий пока нет</Typography>
				</Box>
			) : (
				<Box sx={{ position: 'relative', pl: 5 }}>
					<Box
						sx={{
							position: 'absolute',
							left: 15,
							top: 8,
							bottom: 8,
							width: '1px',
							bgcolor: '#e5e7eb',
						}}
					/>
					<Stack spacing={2}>
						{entries.map(e => (
							<TimelineRow key={e.id} entry={e} />
						))}
					</Stack>
				</Box>
			)}
		</Stack>
	)
}

const TimelineRow: FC<{ entry: Entry }> = ({ entry }) => (
	<Box sx={{ position: 'relative' }}>
		<Box
			sx={{
				position: 'absolute',
				left: -40,
				top: 2,
				width: 32,
				height: 32,
				borderRadius: '50%',
				bgcolor: `${entry.color}1a`,
				color: entry.color,
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				boxShadow: '0 0 0 4px #fff',
				zIndex: 1,
			}}
		>
			<entry.Icon sx={{ fontSize: 15 }} />
		</Box>
		<Box sx={{ bgcolor: '#f9fafb', borderRadius: '10px', p: 1.5 }}>
			<Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
				<Typography sx={{ fontSize: 14, fontWeight: 500, color: '#1f2937' }}>{entry.title}</Typography>
				<Typography sx={{ fontSize: 12, color: '#9ca3af', whiteSpace: 'nowrap' }}>
					{fmtDateTime(entry.date)}
				</Typography>
			</Box>
			{entry.subtitle && (
				<Typography sx={{ fontSize: 12, color: '#6b7280', mt: 0.5 }}>{entry.subtitle}</Typography>
			)}
		</Box>
	</Box>
)
