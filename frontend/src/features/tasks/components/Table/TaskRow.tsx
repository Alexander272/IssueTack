import { useMemo } from 'react'
import { TableRow, TableCell, Typography, Box, Tooltip, type SxProps, type Theme } from '@mui/material'
import dayjs from 'dayjs'

import type { ITask } from '../../types/task'
import {
	ACTIVE_STATUSES,
	DUE_DATE_COLORS,
	DUE_DATE_ICONS,
	DUE_DATE_DEFAULT_ICON,
	SLA_TONES,
} from '../../constants/taskMaps'
import { getDueDateTone, getShortDate, getSmartDate } from '@/utils/date'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { TaskAssignmentChip } from '../TaskAssignmentChip'
import { TaskProgressBar } from '../TaskProgressBar'
import type { TaskColumnField, TaskListViewState } from './taskView'

type SlaInfo = { tone: 'onTime' } | { tone: 'overdue'; daysLate: number } | { tone: 'neutral' } | { tone: 'cancelled' }

function getSlaTooltipTitle(task: ITask, slaInfo: SlaInfo | null): string {
	if (task.status === 'cancelled') {
		return `Отменена ${getShortDate(task.closedAt)}`
	}
	if (!task.dueDate || !task.closedAt) {
		return 'Срок не был задан'
	}

	const base = `Срок: ${getShortDate(task.dueDate)} · Решена: ${getShortDate(task.resolvedAt)}`

	switch (slaInfo?.tone) {
		case 'onTime':
			return `${base} · В срок`
		case 'overdue':
			return `${base} · +${slaInfo.daysLate} дн.`
		default:
			return base
	}
}

interface Props {
	task: ITask
	view: TaskListViewState
	onClick: (task: ITask) => void
	sx?: SxProps<Theme>
}

export const TaskRow = ({ task, view, onClick, sx }: Props) => {
	const show = (field: TaskColumnField) => view.columns.has(field)
	const displayId = task.ticketNumber || '—'
	const dueTone = ACTIVE_STATUSES.includes(task.status) ? getDueDateTone(task.dueDate) : null
	const DueDateIcon = dueTone ? DUE_DATE_ICONS[dueTone] : DUE_DATE_DEFAULT_ICON
	const dueDateColor = dueTone ? DUE_DATE_COLORS[dueTone] : '#4b5563'
	const subtaskProgress = task.subtasks
		? {
				done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
				total: task.subtasks.length,
			}
		: { done: 0, total: 0 }

	const isTerminal = !ACTIVE_STATUSES.includes(task.status)

	// Создатель не исполнитель — заявка от «чужого» клиента: название выделяем primary цветом.
	const creatorIsNotAssignee = task.assignee != null && task.assignee.id !== task.creator.id

	const slaInfo = useMemo(() => {
		if (!view.isArchive || !isTerminal) return null
		if (task.status === 'cancelled') {
			return { tone: 'cancelled' as const }
		}

		// На случай, если заявка закрылась без этапа решения, можно оставить фоллбэк на closedAt
		const targetDate = task.resolvedAt ?? task.closedAt
		if (!task.dueDate || !targetDate) {
			return { tone: 'neutral' as const }
		}
		const resolved = dayjs(targetDate)
		const due = dayjs(task.dueDate)

		if (!resolved.isValid() || !due.isValid()) {
			return { tone: 'neutral' as const }
		}
		const diffMs = resolved.valueOf() - due.valueOf()
		if (diffMs <= 0) {
			return { tone: 'onTime' as const }
		}
		const diffDays = resolved.startOf('day').diff(due.startOf('day'), 'day')
		return { tone: 'overdue' as const, daysLate: diffDays }
	}, [view.isArchive, isTerminal, task.status, task.dueDate, task.resolvedAt, task.closedAt])

	const sla = SLA_TONES[slaInfo?.tone ?? 'neutral']
	const SlaIcon = sla.icon

	return (
		<TableRow
			hover
			onClick={() => onClick(task)}
			sx={{
				cursor: 'pointer',
				borderBottom: '1px solid #f3f4f6',
				'&:hover': { bgcolor: '#fafafa' },
				'&:last-child td': { border: 0 },
				...sx,
			}}
		>
			{show('ticketNumber') && (
				<TableCell>
					<Typography sx={{ fontWeight: 600, color: '#111827', fontSize: '0.875rem' }}>
						{displayId}
					</Typography>
				</TableCell>
			)}
			{show('title') && (
				<TableCell>
					<Tooltip title={task.title} placement='top'>
						<Typography
							sx={{
								fontWeight: 600,
								color: creatorIsNotAssignee ? '#3300ff' : '#111827',
								fontSize: '0.875rem',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{task.title}
						</Typography>
					</Tooltip>
				</TableCell>
			)}

			{show('owner') && (
				<TableCell>
					<Tooltip
						title={`${task.owner?.lastName} ${task.owner?.firstName}${task.owner?.internalNumber ? ` (${task.owner.internalNumber})` : ''}`}
						placement='top'
					>
						<Typography
							sx={{
								fontSize: '0.875rem',
								color: '#6b7280',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{task.owner?.lastName} {task.owner?.firstName}{' '}
							{task.owner?.internalNumber ? `(${task.owner?.internalNumber})` : null}
						</Typography>
					</Tooltip>
				</TableCell>
			)}

			{show('site') && (
				<TableCell sx={{ py: 1 }}>
					<Tooltip title={task.site?.name ?? '—'} placement='top'>
						<Box
							sx={{
								display: 'inline-flex',
								alignItems: 'center',
								px: 1.5,
								py: 1,
								borderRadius: '999px',
								fontSize: '0.75rem',
								fontWeight: 500,
								bgcolor: '#f3f4f6',
								color: '#374151',
								maxWidth: '100%',
								minWidth: 0,
							}}
						>
							<Typography
								component='span'
								sx={{
									display: 'block',
									minWidth: 0,
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap',
									fontSize: '0.75rem',
									fontWeight: 500,
									lineHeight: 1,
								}}
							>
								{task.site?.name ?? '—'}
							</Typography>
						</Box>
					</Tooltip>
				</TableCell>
			)}

			{show('createdAt') && (
				<TableCell sx={{ py: 1 }}>
					<Tooltip title={getSmartDate(task.createdAt) ?? '—'} placement='top'>
						<Box sx={{ display: 'flex', flexDirection: 'column' }}>
							<Box
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									gap: 0.5,
								}}
							>
								<DUE_DATE_DEFAULT_ICON sx={{ fontSize: 14 }} />
								<Typography
									component='span'
									sx={{
										fontSize: '0.875rem',
										fontWeight: 400,
										lineHeight: 1,
									}}
								>
									{getShortDate(task.createdAt)}
								</Typography>
							</Box>
						</Box>
					</Tooltip>
				</TableCell>
			)}

			{(show('dueDate') || show('closedAt')) && (
				<TableCell>
					{!view.isArchive && (
						<Box sx={{ display: 'flex', flexDirection: 'column' }}>
							<Box
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									gap: 0.5,
									color: dueDateColor,
								}}
							>
								<DueDateIcon sx={{ fontSize: 14, color: dueDateColor }} />
								<Typography
									component='span'
									sx={{
										fontSize: '0.875rem',
										fontWeight: dueTone && dueTone !== 'soon' ? 700 : 400,
										lineHeight: 1,
									}}
								>
									{getShortDate(task.dueDate)}
								</Typography>
							</Box>
							{task.closedAt && (
								<Typography sx={{ fontSize: '0.75rem', color: '#059669' }}>
									Закрыта: {getShortDate(task.closedAt)}
								</Typography>
							)}
						</Box>
					)}
					{view.isArchive && (
						<Tooltip title={getSlaTooltipTitle(task, slaInfo)} placement='top'>
							<Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
								<SlaIcon sx={{ fontSize: 14, color: sla.iconColor }} />
								<Typography
									component='span'
									sx={{
										fontSize: '0.875rem',
										fontWeight: sla.bold ? 700 : 400,
										color: sla.textColor,
										lineHeight: 1,
									}}
								>
									{getShortDate(task.closedAt)}
								</Typography>
							</Box>
						</Tooltip>
					)}
				</TableCell>
			)}

			{show('priority') && (
				<TableCell sx={{ py: 1 }}>
					<TaskPriorityBadge priority={task.priority} />
				</TableCell>
			)}

			{show('assignee') && (
				<TableCell sx={{ py: 1 }}>
					<TaskAssignmentChip assignee={task.assignee} />
				</TableCell>
			)}

			{show('status') && (
				<TableCell sx={{ py: 1 }}>
					<TaskStatusBadge status={task.status} duplicateOfNumber={task.duplicateOfNumber} />
				</TableCell>
			)}

			{show('subtasks') && (
				<TableCell align='right'>
					<TaskProgressBar done={subtaskProgress.done} total={subtaskProgress.total} />
				</TableCell>
			)}
		</TableRow>
	)
}
