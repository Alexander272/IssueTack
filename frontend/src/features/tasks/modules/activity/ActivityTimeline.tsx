import dayjs from 'dayjs'
import { Box, Typography } from '@mui/material'
import { ArrowRight } from 'lucide-mui'

import { useAppSelector } from '@/hooks/redux'
import { getHasNoGroups } from '@/features/user/userSlice'
import { getDisplayName as getDisplayNameBase } from '@/utils/avatar'

import type { IActivityLog } from '../../types/activity'
import { PRIORITY_MAP, STATUS_MAP } from '../../constants/taskMaps'
import type { Priority, TicketStatus } from '../../types/task'
import { ACTION_META, GROUPLESS_HIDDEN_FIELDS, MULTI_META, fieldMeta, type FieldMeta } from './activityMeta'

interface VisibleChange {
	tag: string
	oldValue: string
	newValue?: string
}

const getDisplayName = (log: IActivityLog): string => {
	if (log.actor?.firstName || log.actor?.lastName) return getDisplayNameBase(log.actor)
	return log.changedByName || getDisplayNameBase(log.actor)
}

// Изменения записи, оставшиеся после фильтрации скрытых для безгруппового полей.
// Порядок — как в newValues. Для удаления значения лежат только в oldValues.
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

// Значение отсутствия: undefined — ключа нет, '' — фронтовый дефолт пустой старой
// стороны (и пустое описание), 'none' — каноничное «не задано». Легаси-артефакты
// "<nil>"/"nil" нормализованы в БД миграцией-бэкфиллом, здесь не нужны.
const isNone = (value: string | undefined): boolean => value === undefined || value === '' || value === 'none'

const formatDate = (value: string): string => {
	if (isNone(value)) return '—'
	const d = dayjs(value)
	if (!d.isValid()) return value
	const hasTime = d.hour() !== 0 || d.minute() !== 0
	return hasTime ? d.format('D MMMM YYYY, HH:mm') : d.format('D MMMM YYYY')
}

const chipSx = (bgColor: string, textColor: string) => ({
	display: 'inline-flex',
	alignItems: 'center',
	gap: 0.5,
	px: 1,
	py: 0.25,
	borderRadius: '999px',
	fontSize: '0.75rem',
	fontWeight: 500,
	bgcolor: bgColor,
	color: textColor,
	lineHeight: 1.4,
})

const StatusChip = ({ value }: { value: string }) => {
	const info = STATUS_MAP[value as TicketStatus]
	if (!info) return <>{isNone(value) ? '—' : value}</>

	return (
		<Box component='span' sx={chipSx(info.bgColor, info.textColor)}>
			<info.icon sx={{ fontSize: 13, color: info.textColor }} />
			{info.label}
		</Box>
	)
}

const PriorityChip = ({ value }: { value: string }) => {
	const info = PRIORITY_MAP[value as Priority]
	if (!info) return <>{isNone(value) ? '—' : value}</>

	return <Box component='span' sx={chipSx(info.bgColor, info.textColor)}>{info.label}</Box>
}

// Одно значение в строке изменения. Чипы статуса/приоритета цветные с обеих сторон;
// у заголовка/описания и даты старое серое и зачёркнутое, у «имени» — обычное тёмное,
// новое везде темнее и жирнее.
const ChangeValue = ({ meta, value, side }: { meta: FieldMeta; value: string | undefined; side: 'old' | 'new' }) => {
	if (meta.kind === 'status') return <StatusChip value={value ?? ''} />
	if (meta.kind === 'priority') return <PriorityChip value={value ?? ''} />

	const text = meta.kind === 'date' ? formatDate(value ?? '') : isNone(value) ? '—' : value
	const isOld = side === 'old'
	const isPlain = meta.kind === 'plain'

	return (
		<Typography
			component='span'
			sx={{
				fontSize: '0.8125rem',
				// У «имени» старое не приглушаем — контраст только по жирности (как в макете),
				// у заголовка/описания и даты старое серое и зачёркнутое.
				color: isOld ? (isPlain ? '#374151' : '#9ca3af') : '#1f2937',
				fontWeight: isOld ? 400 : isPlain ? 600 : 500,
				textDecoration: isOld && (meta.kind === 'diff' || meta.kind === 'date') ? 'line-through' : 'none',
				wordBreak: 'break-word',
			}}
		>
			{text}
		</Typography>
	)
}

const ChangeRow = ({ change, showLabel }: { change: VisibleChange; showLabel: boolean }) => {
	const meta = fieldMeta(change.tag)

	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
			{showLabel && (
				<Typography component='span' sx={{ fontSize: '0.8125rem', color: '#6b7280', fontWeight: 600 }}>
					{meta.label}:
				</Typography>
			)}
			{change.newValue !== undefined ? (
				<>
					<ChangeValue meta={meta} value={change.oldValue} side='old' />
					<ArrowRight sx={{ fontSize: 13, color: '#9ca3af', flexShrink: 0 }} />
					<ChangeValue meta={meta} value={change.newValue} side='new' />
				</>
			) : (
				<ChangeValue meta={meta} value={change.oldValue} side='new' />
			)}
		</Box>
	)
}

const TimelineEntry = ({
	log,
	hasNoGroups,
	isLast,
}: {
	log: IActivityLog
	hasNoGroups: boolean
	isLast: boolean
}) => {
	const changes = visibleChanges(log, hasNoGroups)

	const actionMeta = log.action === 'created' ? ACTION_META.created : log.action === 'deleted' ? ACTION_META.deleted : null

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
		phrase = meta.phrase
	} else {
		Icon = MULTI_META.icon
		iconColor = MULTI_META.color
		phrase = MULTI_META.phrase
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
						{log.entity ? `«${log.entity}»` : ''}
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

const dayLabel = (day: string): string => {
	const d = dayjs(day)
	const diff = d.startOf('day').diff(dayjs().startOf('day'), 'day')
	if (diff === 0) return 'Сегодня'
	if (diff === -1) return 'Вчера'
	return d.format('D MMMM YYYY')
}

export const ActivityTimeline = ({ logs }: { logs: IActivityLog[] }) => {
	const hasNoGroups = useAppSelector(getHasNoGroups)

	// Логи приходят по убыванию created_at, поэтому группируем подряд идущие дни,
	// не пересортируя. Скрытые для безгруппового записи отсеиваем заранее, чтобы
	// не осталось дня-заголовка без содержимого.
	const groups: { day: string; items: IActivityLog[] }[] = []
	for (const log of logs) {
		if (log.action !== 'created' && visibleChanges(log, hasNoGroups).length === 0) continue
		const day = dayjs(log.createdAt).format('YYYY-MM-DD')
		const last = groups[groups.length - 1]
		if (last && last.day === day) last.items.push(log)
		else groups.push({ day, items: [log] })
	}

	if (groups.length === 0) {
		return (
			<Typography sx={{ color: '#9ca3af', fontSize: '0.875rem', textAlign: 'center', py: 3 }}>
				История пуста
			</Typography>
		)
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
