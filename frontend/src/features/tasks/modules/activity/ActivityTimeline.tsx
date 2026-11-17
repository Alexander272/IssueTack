import dayjs from 'dayjs'
import { Box, Typography } from '@mui/material'

import { useAppSelector } from '@/hooks/redux'
import { getHasNoGroups } from '@/features/user/userSlice'

import type { IActivityLog } from '../../types/activity'
import { GROUPLESS_HIDDEN_FIELDS } from './activityMeta'
import { TimelineEntry } from './TimelineEntry'

const dayLabel = (day: string): string => {
	const d = dayjs(day)
	const diff = d.startOf('day').diff(dayjs().startOf('day'), 'day')
	if (diff === 0) return 'Сегодня'
	if (diff === -1) return 'Вчера'
	return d.format('D MMMM YYYY')
}

const visibleChanges = (log: IActivityLog, hasNoGroups: boolean): { tag: string; oldValue: string; newValue?: string }[] => {
	if (log.action === 'created') return []

	const visible = ([tag]: [string, string]) => !(hasNoGroups && GROUPLESS_HIDDEN_FIELDS.has(tag))

	if (log.action === 'deleted') {
		return Object.entries(log.oldValues ?? {})
			.filter(visible)
			.map(([tag, oldValue]) => ({ tag, oldValue }))
	}

	const olds = log.oldValues ?? {}
	return Object.entries(log.newValues ?? {})
		.filter(visible)
		.map(([tag, newValue]) => ({ tag, oldValue: olds[tag] ?? '', newValue }))
}

export const ActivityTimeline = ({ logs }: { logs: IActivityLog[] }) => {
	const hasNoGroups = useAppSelector(getHasNoGroups)

	const groups: { day: string; items: IActivityLog[] }[] = []
	for (const log of logs) {
		if (log.action !== 'created' && visibleChanges(log, hasNoGroups).length === 0) continue
		const day = dayjs(log.createdAt).format('YYYY-MM-DD')
		const last = groups[groups.length - 1]
		if (last && last.day === day) last.items.push(log)
		else groups.push({ day, items: [log] })
	}

	if (groups.length === 0) {
		return <Typography sx={{ color: '#9ca3af', fontSize: '0.875rem', textAlign: 'center', py: 3 }}>История пуста</Typography>
	}

	return (
		<Box sx={{ px: 2.5, py: 2 }}>
			{groups.map((group, index) => (
				<Box key={group.day} sx={{ mt: index === 0 ? 0 : 1.5 }}>
					<Typography
						sx={{
							position: 'sticky',
							top: 0,
							zIndex: 2,
							bgcolor: '#fff',
							pt: 0.5,
							pb: 0.75,
							mb: 1.25,
							fontSize: '0.6875rem',
							fontWeight: 600,
							color: '#9ca3af',
							textTransform: 'uppercase',
							letterSpacing: '0.05em',
							borderBottom: '1px solid #e5e7eb',
						}}
					>
						{dayLabel(group.day)}
					</Typography>
					{group.items.map((log, index) => (
						<TimelineEntry key={log.id} log={log} hasNoGroups={hasNoGroups} isLast={index === group.items.length - 1} />
					))}
				</Box>
			))}
		</Box>
	)
}