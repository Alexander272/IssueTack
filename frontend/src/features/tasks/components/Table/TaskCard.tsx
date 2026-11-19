import { Box, Typography, Stack } from '@mui/material'
import { Paperclip, MessageSquare } from 'lucide-mui'

import type { ITask } from '../../types/task'
import {
	PRIORITY_MAP,
	ACTIVE_STATUSES,
	DUE_DATE_COLORS,
	DUE_DATE_ICONS,
	DUE_DATE_DEFAULT_ICON,
} from '../../constants/taskMaps'
import { getDueDateTone, getShortDate } from '@/utils/date'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { TaskAssignmentChip } from '../TaskAssignmentChip'
import { TaskProgressBar } from '../TaskProgressBar'
import type { TaskListViewState } from './taskView'

interface Props {
	task: ITask
	view: TaskListViewState
	onClick: (task: ITask) => void
}

export const TaskCard = ({ task, view, onClick }: Props) => {
	const priorityColor = PRIORITY_MAP[task.priority]?.barColor ?? '#6b7280'
	const dueTone = ACTIVE_STATUSES.includes(task.status) ? getDueDateTone(task.dueDate) : null
	const DueDateIcon = dueTone ? DUE_DATE_ICONS[dueTone] : DUE_DATE_DEFAULT_ICON

	const displayId = task.ticketNumber ? `№${task.ticketNumber}` : null

	const subtaskProgress = task.subtasks
		? {
				done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
				total: task.subtasks.length,
			}
		: null

	const attachmentsCount = task.attachments?.length ?? 0
	const commentsCount = task.comments?.length ?? 0

	// Подвал карточки подчиняется тем же колонкам, что и таблица: срок скрыт у
	// безгруппового пользователя и при группировке по сроку, в архиве вместо срока
	// показывается дата закрытия. Раньше это решалось отдельным флагом hasNoGroups.
	const showClosed = Boolean(task.closedAt)
	const showDue = view.columns.has('dueDate') && !task.closedAt

	return (
		<Box
			onClick={() => onClick(task)}
			sx={{
				position: 'relative',
				bgcolor: '#fff',
				borderRadius: 3,
				border: '1px solid #e5e7eb',
				boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
				cursor: 'pointer',
				overflow: 'hidden',
				transition: 'all 0.2s ease',
				'&:hover': {
					transform: 'translateY(-2px)',
					boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)',
				},
			}}
		>
			{!view.hasNoGroups && (
				<Box
					sx={{
						position: 'absolute',
						left: 0,
						top: 0,
						bottom: 0,
						width: 4,
						borderRadius: '12px 0 0 12px',
						bgcolor: priorityColor,
					}}
				/>
			)}

			<Box sx={{ p: 1.5, pl: 2.5 }}>
				<Stack direction='row' spacing={1} useFlexGap sx={{ alignItems: 'center', mb: 0.5, flexWrap: 'wrap' }}>
					{displayId && (
						<Typography sx={{ fontSize: '0.75rem', fontFamily: 'mono', color: '#4f5562', fontWeight: 500 }}>
							{displayId}
						</Typography>
					)}

					<TaskStatusBadge status={task.status} sx={{ px: 1, py: 0.5, fontSize: '0.625rem' }} />
					{!view.hasNoGroups && (
						<TaskPriorityBadge priority={task.priority} sx={{ px: 1, py: 0.5, fontSize: '0.625rem' }} />
					)}
				</Stack>

				<Typography
					sx={{
						fontWeight: 700,
						color: '#111827',
						mb: 1,
						fontSize: '0.875rem',
						display: '-webkit-box',
						WebkitLineClamp: 2,
						WebkitBoxOrient: 'vertical',
						overflow: 'hidden',
					}}
				>
					{task.title}
				</Typography>

				{task.description && (
					<Typography
						sx={{
							fontSize: '0.8125rem',
							color: '#6b7280',
							mb: 1.5,
							display: '-webkit-box',
							WebkitLineClamp: 2,
							WebkitBoxOrient: 'vertical',
							overflow: 'hidden',
						}}
					>
						{task.description}
					</Typography>
				)}

				<Stack direction='row' spacing={1} sx={{ mb: 1.5 }}>
					<Box
						component='span'
						sx={{
							display: 'inline-flex',
							px: 1.25,
							py: 0.25,
							borderRadius: '999px',
							fontSize: '0.7rem',
							fontWeight: 500,
							bgcolor: '#f3e8ff',
							color: '#7c3aed',
						}}
					>
						{task.category.name}
					</Box>
					<Box
						component='span'
						sx={{
							display: 'inline-flex',
							alignItems: 'center',
							px: 1.25,
							py: 0.25,
							borderRadius: '999px',
							fontSize: '0.7rem',
							fontWeight: 500,
							bgcolor: '#f3f4f6',
							color: '#374151',
						}}
					>
						{task.site?.name ?? '—'}
					</Box>
				</Stack>

				<Box sx={{ mb: subtaskProgress && subtaskProgress.total > 0 ? 1.5 : 0 }}>
					<TaskAssignmentChip assignee={task.assignee} />
				</Box>

				{subtaskProgress && subtaskProgress.total > 0 && (
					<Box sx={{ mb: 1.5 }}>
						<TaskProgressBar done={subtaskProgress.done} total={subtaskProgress.total} />
					</Box>
				)}

				<Stack
					direction='row'
					spacing={2}
					sx={{ alignItems: 'center', pt: 0.5, mt: 1, borderTop: '1px solid #f3f4f6' }}
				>
					{(showClosed || showDue) && (
						<Box
							sx={{
								display: 'inline-flex',
								alignItems: 'center',
								gap: 0.5,
								fontSize: '0.75rem',
								color: showClosed ? '#059669' : dueTone ? DUE_DATE_COLORS[dueTone] : '#9ca3af',
								fontWeight: showClosed || (dueTone && dueTone !== 'soon') ? 600 : 400,
							}}
						>
							{!showClosed && <DueDateIcon sx={{ fontSize: 14 }} />}
							<Typography component='span' sx={{ fontSize: '0.75rem', lineHeight: 1 }}>
								{showClosed ? 'Закрыта' : 'Срок'}:{' '}
								{showClosed ? getShortDate(task.closedAt) : getShortDate(task.dueDate)}
							</Typography>
						</Box>
					)}

					{attachmentsCount > 0 && (
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: '#9ca3af' }}>
							<Paperclip sx={{ fontSize: 14 }} />
							<Typography component='span' sx={{ fontSize: '0.75rem', lineHeight: 1 }}>
								{attachmentsCount}
							</Typography>
						</Box>
					)}

					{commentsCount > 0 && (
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: '#9ca3af' }}>
							<MessageSquare sx={{ fontSize: 14 }} />
							<Typography component='span' sx={{ fontSize: '0.75rem', lineHeight: 1 }}>
								{commentsCount}
							</Typography>
						</Box>
					)}
				</Stack>
			</Box>
		</Box>
	)
}
