import { Box, Typography } from '@mui/material'

import type { IActivityLog } from '../../types/activity'
import { STATUS_MAP } from '../../constants/taskMaps'
import type { TicketStatus } from '../../types/task'

const ACTION_LABELS: Record<string, string> = {
	created: 'Создал',
	updated: 'Изменил',
	deleted: 'Удалил',
}

const FIELD_LABELS: Record<string, string> = {
	title_changed: 'Заголовок',
	description_changed: 'Описание',
	status_changed: 'Статус',
	priority_changed: 'Приоритет',
	assigned: 'Исполнитель',
	assign_changed: 'Исполнитель',
	owner_changed: 'Владелец',
	group_changed: 'Группа',
	group_assigned: 'Группа',
	due_date_changed: 'Срок',
	site_changed: 'Площадка',
	category_changed: 'Категория',
	closed: 'Закрытие',
}

const AVATAR_COLORS = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#ef4444', '#6366f1', '#14b8a6']

const hashCode = (str: string): number => {
	let hash = 0
	for (let i = 0; i < str.length; i++) {
		hash = str.charCodeAt(i) + ((hash << 5) - hash)
	}
	return hash
}

const getAvatarColor = (name: string): string => {
	return AVATAR_COLORS[Math.abs(hashCode(name)) % AVATAR_COLORS.length]
}

const getDisplayName = (log: IActivityLog): string => {
	if (log.actor?.firstName || log.actor?.lastName) {
		return `${log.actor.firstName ?? ''} ${log.actor.lastName ?? ''}`.trim()
	}
	return log.changedByName
}

const getInitials = (log: IActivityLog): string => {
	if (log.actor?.firstName || log.actor?.lastName) {
		const first = log.actor.firstName?.[0] ?? ''
		const last = log.actor.lastName?.[0] ?? ''
		if (first || last) return `${first}${last}`.toUpperCase()
	}
	const name = log.changedByName
	const parts = name.trim().split(/\s+/)
	if (parts.length >= 2) return `${parts[0][0]}${parts[1][0]}`.toUpperCase()
	return name.slice(0, 2).toUpperCase()
}

const StatusValue = ({ value }: { value: string }) => {
	const statusInfo = STATUS_MAP[value as TicketStatus]
	if (!statusInfo) return <>{value}</>

	return (
		<Box component='span' sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5, verticalAlign: 'middle' }}>
			<statusInfo.icon sx={{ fontSize: 14, color: statusInfo.textColor }} />
			{statusInfo.label}
		</Box>
	)
}

const DiffLine = ({
	fieldKey,
	label,
	oldValue,
	newValue,
}: {
	fieldKey: string
	label: string
	oldValue: string
	newValue?: string
}) => {
	const isStatus = fieldKey === 'status_changed'
	const oldValueEl = oldValue === 'none' ? '—' : isStatus ? <StatusValue value={oldValue} /> : oldValue
	const newValueEl = newValue === 'none' ? '—' : isStatus && newValue ? <StatusValue value={newValue} /> : newValue

	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.375 }}>
			<Typography sx={{ fontSize: '0.8125rem', color: '#374151', whiteSpace: 'nowrap', fontWeight: 600 }}>
				{label}:
			</Typography>
			{newValue !== undefined ? (
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
					<Typography
						component='span'
						sx={{ fontSize: '0.8125rem', textDecoration: 'line-through', color: '#5c626c' }}
					>
						{oldValueEl}
					</Typography>
					<Typography component='span' sx={{ fontSize: '0.8125rem', color: '#6b7280' }}>
						→
					</Typography>
					<Typography component='span' sx={{ fontSize: '0.8125rem', color: '#374151' }}>
						{newValueEl}
					</Typography>
				</Box>
			) : (
				<Typography sx={{ fontSize: '0.8125rem', color: '#374151' }}>{oldValueEl}</Typography>
			)}
		</Box>
	)
}

const ActivityDiff = ({ log }: { log: IActivityLog }) => {
	if (log.action === 'created' && log.newValues) {
		return (
			<Box>
				{Object.entries(log.newValues).map(([key, value]) => (
					<DiffLine key={key} fieldKey={key} label={FIELD_LABELS[key] ?? key} oldValue={value} />
				))}
			</Box>
		)
	}

	if (log.action === 'deleted' && log.oldValues) {
		return (
			<Box>
				{Object.entries(log.oldValues).map(([key, value]) => (
					<DiffLine key={key} fieldKey={key} label={FIELD_LABELS[key] ?? key} oldValue={value} />
				))}
			</Box>
		)
	}

	if (log.action === 'updated' && log.oldValues && log.newValues) {
		const fields = Object.keys(log.newValues)
		return (
			<Box>
				{fields.map(key => (
					<DiffLine
						key={key}
						fieldKey={key}
						label={FIELD_LABELS[key] ?? key}
						oldValue={log.oldValues![key] ?? ''}
						newValue={log.newValues![key] ?? ''}
					/>
				))}
			</Box>
		)
	}

	return null
}

export const ActivityEntry = ({ log }: { log: IActivityLog }) => {
	const displayName = getDisplayName(log)
	const initials = getInitials(log)
	const color = getAvatarColor(displayName)

	return (
		<Box sx={{ display: 'flex', gap: 1.5 }}>
			<Box
				sx={{
					width: 36,
					height: 36,
					borderRadius: '50%',
					flexShrink: 0,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					fontSize: '0.75rem',
					fontWeight: 700,
					color: 'white',
					bgcolor: color,
				}}
			>
				{initials}
			</Box>

			<Box sx={{ flex: 1, minWidth: 0 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.25 }}>
					<Typography sx={{ fontSize: '0.8125rem', fontWeight: 600, color: '#1f2937' }}>
						{displayName}
					</Typography>
					<Typography
						sx={{
							fontSize: '0.6875rem',
							px: 1,
							py: 0.125,
							borderRadius: '4px',
							fontWeight: 500,
							bgcolor: '#f3f4f6',
							color: '#374151',
						}}
					>
						{ACTION_LABELS[log.action] ?? log.action}
					</Typography>
					<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', ml: 'auto', flexShrink: 0 }}>
						{new Date(log.createdAt).toLocaleDateString('ru-RU', {
							day: 'numeric',
							month: 'short',
							hour: '2-digit',
							minute: '2-digit',
						})}
					</Typography>
				</Box>

				{log.action === 'updated' ? (
					<Box sx={{ bgcolor: '#f3f4f6', borderRadius: '8px', p: 1.5 }}>
						<ActivityDiff log={log} />
					</Box>
				) : log.action === 'deleted' ? (
					<Box sx={{ bgcolor: '#fef2f2', borderRadius: '8px', p: 1.5 }}>
						<ActivityDiff log={log} />
					</Box>
				) : (
					<Box sx={{ fontSize: '0.8125rem', color: '#6b7280' }}>
						{log.entity && (
							<Typography component='span' sx={{ fontSize: '0.8125rem', color: '#374151' }}>
								{log.entity}
							</Typography>
						)}
					</Box>
				)}
			</Box>
		</Box>
	)
}
