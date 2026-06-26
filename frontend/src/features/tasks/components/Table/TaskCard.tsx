import { Box, Typography, Stack } from '@mui/material'

import type { ITask } from '../../types/task'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { TaskAssignmentChip } from '../TaskAssignmentChip'
import { TaskProgressBar } from '../TaskProgressBar'

interface Props {
	task: ITask
	onClick: (task: ITask) => void
}

export const TaskCard = ({ task, onClick }: Props) => {
	const priorityColor = PRIORITY_MAP[task.priority]?.barColor ?? '#6b7280'

	const subtaskProgress = task.subtasks
		? {
				done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
				total: task.subtasks.length,
			}
		: null

	// const createdDate = new Date(task.createdAt).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })

	const formatDate = (dateStr: string) =>
		new Date(dateStr).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })

	const attachmentsCount = task.attachments?.length ?? 0
	const commentsCount = (task as { comments?: unknown[] }).comments?.length ?? 0

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

			<Box sx={{ p: 2.5, pl: 3.5 }}>
				<Stack direction='row' spacing={1} useFlexGap sx={{ alignItems: 'center', mb: 0.5, flexWrap: 'wrap' }}>
					<Typography sx={{ fontSize: '0.75rem', fontFamily: 'mono', color: '#4f5562', fontWeight: 500 }}>
						№{task.ticketNumber ?? task.id.slice(0, 4)}
					</Typography>

					<TaskStatusBadge status={task.status} sx={{ px: 1, py: 0.5, fontSize: '0.625rem' }} />
					<TaskPriorityBadge priority={task.priority} sx={{ px: 1, py: 0.5, fontSize: '0.625rem' }} />

					{/* <Typography sx={{ fontSize: '0.75rem', color: '#41454a' }}>{createdDate}</Typography> */}
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
					<TaskAssignmentChip assignee={task.assignee} group={task.group} />
				</Box>

				{subtaskProgress && subtaskProgress.total > 0 && (
					<Box sx={{ mb: 1.5 }}>
						<TaskProgressBar done={subtaskProgress.done} total={subtaskProgress.total} />
					</Box>
				)}

				<Stack
					direction='row'
					spacing={2}
					sx={{ alignItems: 'center', pt: 1.5, borderTop: '1px solid #f3f4f6' }}
				>
					<Typography sx={{ fontSize: '0.75rem', color: task.closedAt ? '#059669' : '#9ca3af' }}>
						{task.closedAt ? 'Закрыта' : 'Срок'}:{' '}
						{task.closedAt ? formatDate(task.closedAt) : task.dueDate ? formatDate(task.dueDate) : '—'}
					</Typography>

					{attachmentsCount > 0 && (
						<Typography
							sx={{
								fontSize: '0.75rem',
								color: '#9ca3af',
								display: 'flex',
								alignItems: 'center',
								gap: 0.5,
							}}
						>
							📎 {attachmentsCount}
						</Typography>
					)}

					{commentsCount > 0 && (
						<Typography
							sx={{
								fontSize: '0.75rem',
								color: '#9ca3af',
								display: 'flex',
								alignItems: 'center',
								gap: 0.5,
							}}
						>
							💬 {commentsCount}
						</Typography>
					)}
				</Stack>
			</Box>
		</Box>
	)
}
