import { TableRow, TableCell, Typography, Box, Tooltip, type SxProps, type Theme } from '@mui/material'

import type { ITask } from '../../types/task'
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
	const subtaskProgress = task.subtasks
		? {
				done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
				total: task.subtasks.length,
			}
		: { done: 0, total: 0 }

	const deadlineFormatted = task.dueDate || '—'

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
				<Typography sx={{ fontWeight: 600, color: '#111827', fontSize: '0.875rem' }}>
					{task.ticketNumber ?? task.id.slice(0, 4)}
				</Typography>
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
					<Typography sx={{ fontSize: '0.875rem', color: '#4b5563' }}>{deadlineFormatted}</Typography>
					{task.closedAt && (
						<Typography sx={{ fontSize: '0.75rem', color: '#059669' }}>Закрыта: {task.closedAt}</Typography>
					)}
				</Box>
			</TableCell>
			<TableCell sx={{ py: 1 }}>
				<TaskPriorityBadge priority={task.priority} />
			</TableCell>
			<TableCell sx={{ py: 1 }}>
				<TaskAssignmentChip assignee={task.assignee} group={task.group} />
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
