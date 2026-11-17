import dayjs from 'dayjs'
import { Box, Typography } from '@mui/material'
import { ArrowRight } from 'lucide-mui'

import { PRIORITY_MAP, STATUS_MAP } from '../../constants/taskMaps'
import type { Priority, TicketStatus } from '../../types/task'
import { fieldMeta, type FieldMeta } from './activityMeta'

interface VisibleChange {
	tag: string
	oldValue: string
	newValue?: string
}

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

const ChangeValue = ({ meta, value, side }: { meta: FieldMeta; value: string | undefined; side: 'old' | 'new' }) => {
	if (meta.kind === 'status') return <StatusChip value={value ?? ''} />
	if (meta.kind === 'priority') return <PriorityChip value={value ?? ''} />

	const isOld = side === 'old'
	const isPlain = meta.kind === 'plain'

	if (meta.kind === 'date' && value && value.includes('\n')) {
		const lines = value.split('\n')
		return (
			<Box component='span' sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
				<Typography
					component='span'
					sx={{
						fontSize: '0.8125rem',
						color: isOld ? '#9ca3af' : '#1f2937',
						fontWeight: isOld ? 400 : 500,
						textDecoration: isOld ? 'line-through' : 'none',
					}}
				>
					{formatDate(lines[0])}
				</Typography>
				{lines.slice(1).map((line, i) => (
					<Typography key={i} component='span' sx={{ fontSize: '0.75rem', color: '#6b7280', fontStyle: 'italic' }}>
						{line}
					</Typography>
				))}
			</Box>
		)
	}

	const text = meta.kind === 'date' ? formatDate(value ?? '') : isNone(value) ? '—' : value

	return (
		<Typography
			component='span'
			sx={{
				fontSize: '0.8125rem',
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

export const ChangeRow = ({ change, showLabel }: { change: VisibleChange; showLabel: boolean }) => {
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