import dayjs from 'dayjs'
import { Box, Typography } from '@mui/material'

import { getDisplayName as getDisplayNameBase } from '@/utils/avatar'

import type { IActivityLog } from '../../types/activity'
import { ACTION_META, GROUPLESS_HIDDEN_FIELDS, MULTI_META, MULTI_META_SUBTASK, fieldMeta, type FieldMeta } from './activityMeta'
import { ChangeRow } from './ChangeRow'

interface VisibleChange {
	tag: string
	oldValue: string
	newValue?: string
}

const getDisplayName = (log: IActivityLog): string => {
	if (log.actor?.firstName || log.actor?.lastName) return getDisplayNameBase(log.actor)
	return log.changedByName || getDisplayNameBase(log.actor)
}

const visibleChanges = (log: IActivityLog, hasNoGroups: boolean): VisibleChange[] => {
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

export const TimelineEntry = ({ log, hasNoGroups, isLast }: { log: IActivityLog; hasNoGroups: boolean; isLast: boolean }) => {
	const changes = visibleChanges(log, hasNoGroups)
	const isSubtask = log.entityType === 'subtask'

	const actionMeta =
		log.action === 'created'
			? ACTION_META.created[isSubtask ? 'subtask' : 'ticket']
			: log.action === 'deleted'
				? ACTION_META.deleted[isSubtask ? 'subtask' : 'ticket']
				: null

	let Icon: FieldMeta['icon']
	let iconColor: string
	let phrase: string
	if (actionMeta) {
		Icon = actionMeta.icon
		iconColor = actionMeta.color
		phrase = actionMeta.phrase
	} else if (changes.length === 1) {
		const meta = fieldMeta(changes[0].tag)
		Icon = meta.icon
		iconColor = meta.color
		phrase = isSubtask ? meta.phrase + ' подзадачи' : meta.phrase
	} else {
		Icon = isSubtask ? MULTI_META_SUBTASK.icon : MULTI_META.icon
		iconColor = isSubtask ? MULTI_META_SUBTASK.color : MULTI_META.color
		phrase = isSubtask ? MULTI_META_SUBTASK.phrase : MULTI_META.phrase
	}

	const showLabels = log.action === 'updated' && changes.length > 1

	return (
		<Box sx={{ display: 'flex', gap: 1.25 }}>
			<Box
				sx={{
					width: 20,
					flexShrink: 0,
					display: 'flex',
					flexDirection: 'column',
					alignItems: 'center',
					alignSelf: 'stretch',
				}}
			>
				<Box
					sx={{
						width: 20,
						height: 20,
						borderRadius: '50%',
						flexShrink: 0,
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						bgcolor: iconColor,
						zIndex: 1,
					}}
				>
					<Icon sx={{ fontSize: 12, color: '#fff' }} />
				</Box>
				{!isLast && <Box sx={{ flex: 1, width: '2px', bgcolor: '#e5e7eb', mt: 0.5 }} />}
			</Box>

			<Box sx={{ flex: 1, minWidth: 0, pb: isLast ? 0 : 1.5 }}>
				<Box sx={{ display: 'flex', alignItems: 'baseline', gap: 0.75, flexWrap: 'wrap', mb: 0.75 }}>
					<Typography sx={{ fontSize: '0.8125rem', fontWeight: 600, color: '#1f2937' }}>
						{getDisplayName(log)}
					</Typography>
					<Typography sx={{ fontSize: '0.8125rem', color: '#6b7280' }}>{phrase}</Typography>
					<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', ml: 'auto', flexShrink: 0 }}>
						{dayjs(log.createdAt).format('HH:mm')}
					</Typography>
				</Box>

				{log.action === 'created' ? (
					<Typography sx={{ fontSize: '0.8125rem', color: '#374151', wordBreak: 'break-word' }}>
						{log.entity ? `"${log.entity}"` : ''}
					</Typography>
				) : (
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
						{changes.map(change => (
							<ChangeRow key={change.tag} change={change} showLabel={showLabels} />
						))}
					</Box>
				)}
			</Box>
		</Box>
	)
}