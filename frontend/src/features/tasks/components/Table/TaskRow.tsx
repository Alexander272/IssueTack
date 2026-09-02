import { TableRow, TableCell, Typography, Box, Tooltip, type SxProps, type Theme } from '@mui/material'

import type { ITask } from '../../types/task'
import { ACTIVE_STATUSES, DUE_DATE_COLORS, DUE_DATE_ICONS, DUE_DATE_DEFAULT_ICON } from '../../constants/taskMaps'
import { getDueDateTone, getShortDate } from '@/utils/date'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { TaskAssignmentChip } from '../TaskAssignmentChip'
import { TaskProgressBar } from '../TaskProgressBar'

interface Props {
	task: ITask
	onClick: (task: ITask) => void
	sx?: SxProps<Theme>
}

export const TaskRow = ({ task, onClick, sx }: Props) => {
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
			<TableCell>
				<Typography sx={{ fontWeight: 600, color: '#111827', fontSize: '0.875rem' }}>{displayId}</Typography>
			</TableCell>
			<TableCell>
				<Tooltip title={task.title} placement='top'>
					<Typography
						sx={{
							fontWeight: 600,
							color: '#111827',
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
			<TableCell sx={{ py: 1 }}>
				<Tooltip title={task.site?.name ?? '—'} placement='top'>
					<Box
						sx={{
							display: 'inline-flex',
							px: 1.5,
							py: 1,
							borderRadius: '999px',
							fontSize: '0.75rem',
							fontWeight: 500,
							bgcolor: '#f3f4f6',
							color: '#374151',
							maxWidth: '100%',
							overflow: 'hidden',
							textOverflow: 'ellipsis',
							whiteSpace: 'nowrap',
						}}
					>
						{task.site?.name ?? '—'}
					</Box>
				</Tooltip>
			</TableCell>
			<TableCell>
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
			</TableCell>
			<TableCell sx={{ py: 1 }}>
				<TaskPriorityBadge priority={task.priority} />
			</TableCell>
			<TableCell sx={{ py: 1 }}>
				<TaskAssignmentChip assignee={task.assignee} />
			</TableCell>
			<TableCell sx={{ py: 1 }}>
				<TaskStatusBadge status={task.status} />
			</TableCell>
			<TableCell align='right'>
				{subtaskProgress.total === 0 ? (
					<Typography variant='caption' color='text.secondary'>
						Без подзадач
					</Typography>
				) : (
					<TaskProgressBar done={subtaskProgress.done} total={subtaskProgress.total} />
				)}
			</TableCell>
		</TableRow>
	)
}
