import { type FC } from 'react'
import { Box, Button, CircularProgress, Typography } from '@mui/material'

import type { ITask } from '@/features/tasks/types/task'
import { TaskStatusBadge } from '@/features/tasks/components/TaskStatusBadge'
import { TaskPriorityBadge } from '@/features/tasks/components/TaskPriorityBadge'
import { ACTIVE_STATUSES, DUE_DATE_COLORS, DUE_DATE_ICONS, DUE_DATE_DEFAULT_ICON } from '@/features/tasks/constants/taskMaps'
import { getDueDateTone, getShortDate } from '@/utils/date'

interface IStatisticsTicketsList {
	tasks: ITask[]
	isLoading: boolean
	isFetching: boolean
	total: number
	onOpenTask: (task: ITask) => void
	onShowMore: () => void
	/** dimension задаёт отображение второй стороны: 'owner' — показываем исполнителя, 'assignee' — заказчика */
	dimension?: 'owner' | 'assignee'
}

const DUE_LABEL: Record<Exclude<ReturnType<typeof getDueDateTone>, null>, string> = {
	overdue: 'просрочено',
	today: 'сегодня',
	soon: 'скоро',
}

// personLabel показывает вторую сторону заявки: в разрезе исполнителя — заказчика,
// в разрезе заказчика — исполнителя.
const personLabel = (task: ITask, dimension?: 'owner' | 'assignee') => {
	if (!dimension) {
		// Для KPI показываем владельца
		return task.owner ? `${task.owner.lastName} ${task.owner.firstName}` : 'Без заказчика'
	}
	const user = dimension === 'owner' ? task.assignee : task.owner
	if (!user) return dimension === 'owner' ? 'Без исполнителя' : 'Без заказчика'
	return `${user.lastName} ${user.firstName}${user.internalNumber ? ` (${user.internalNumber})` : ''}`
}

export const StatisticsTicketsList: FC<IStatisticsTicketsList> = ({
	tasks,
	isLoading,
	isFetching,
	total,
	onOpenTask,
	onShowMore,
	dimension,
}) => {
	if (isLoading) {
		return (
			<Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
				<CircularProgress size={28} />
			</Box>
		)
	}

	if (tasks.length === 0) {
		return (
			<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
				<Typography variant='body2'>Нет заявок</Typography>
			</Box>
		)
	}

	return (
		<>
			{tasks.map(task => {
				const dueTone = ACTIVE_STATUSES.includes(task.status) ? getDueDateTone(task.dueDate) : null
				const DueDateIcon = dueTone ? DUE_DATE_ICONS[dueTone] : DUE_DATE_DEFAULT_ICON
				const dueDateColor = dueTone ? DUE_DATE_COLORS[dueTone] : '#4b5563'
				const dueBold = dueTone === 'overdue' || dueTone === 'today'

				return (
					<Box
						key={task.id}
						onClick={() => onOpenTask(task)}
						sx={{
							display: 'flex',
							alignItems: 'center',
							gap: 1.5,
							px: { xs: 2, sm: 2.5 },
							py: 1.75,
							cursor: 'pointer',
							borderBottom: '1px solid #f3f4f6',
							'&:hover': { bgcolor: '#f8fafc' },
							'&:hover .ticket-title': { color: 'primary.main' },
							'&:last-child': { borderBottom: 0 },
						}}
					>
						<Box
							sx={{
								display: 'flex',
								flexDirection: 'column',
								alignItems: 'center',
								justifyContent: 'center',
								gap: 0.5,
								flexShrink: 0,
								minWidth: 28,
							}}
						>
							<Typography sx={{ fontWeight: 700, color: '#111827', fontSize: '1.125rem', lineHeight: 1 }}>
								{task.ticketNumber ?? '—'}
							</Typography>
							{dueTone && (
								<Box
									sx={{
										width: 6,
										height: 6,
										borderRadius: '50%',
										bgcolor: DUE_DATE_COLORS[dueTone],
									}}
								/>
							)}
						</Box>

						<Box sx={{ flexGrow: 1, minWidth: 0 }}>
							<Box
								sx={{
									display: 'flex',
									alignItems: 'flex-start',
									justifyContent: 'space-between',
									gap: 1,
									mb: 0.25,
								}}
							>
								<Typography
									className='ticket-title'
									sx={{
										fontWeight: 600,
										color: '#111827',
										fontSize: '0.875rem',
										overflow: 'hidden',
										textOverflow: 'ellipsis',
										whiteSpace: 'nowrap',
										transition: 'color .15s',
									}}
								>
									{task.title}
								</Typography>
							</Box>
							<Typography
								sx={{
									color: '#6b7280',
									fontSize: '0.75rem',
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap',
								}}
							>
								{personLabel(task, dimension)} · {task.site?.name ?? '—'}
							</Typography>
							<Typography sx={{ color: '#9ca3af', fontSize: '0.75rem', mt: 0.25 }}>
								Создана {getShortDate(task.createdAt)}
							</Typography>
						</Box>

						<Box
							sx={{
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
								gap: 1.5,
								flexShrink: 0,
							}}
						>
							<Box sx={{ display: 'flex', gap: 0.75, flexShrink: 0 }}>
								<Box sx={{ display: { xs: 'none', sm: 'flex' } }}>
									<TaskPriorityBadge priority={task.priority} />
								</Box>
								<TaskStatusBadge status={task.status} />
							</Box>

							<Box
								sx={{
									display: { xs: 'none', md: 'flex' },
									flexDirection: 'column',
									alignItems: 'flex-end',
									gap: 0.5,
								}}
							>
								<Box
									sx={{
										display: 'flex',
										alignItems: 'center',
										gap: 0.75,
										color: dueDateColor,
									}}
								>
									<DueDateIcon sx={{ fontSize: 14, color: dueDateColor }} />
									<Typography
										component='span'
										sx={{
											fontSize: '0.8125rem',
											fontWeight: dueBold ? 700 : 400,
											lineHeight: 1,
											color: dueDateColor,
										}}
									>
										{getShortDate(task.dueDate)}
									</Typography>
								</Box>
								{dueTone && (
									<Typography sx={{ fontSize: '0.6875rem', color: dueDateColor, lineHeight: 1 }}>
										{DUE_LABEL[dueTone]}
									</Typography>
								)}
							</Box>
						</Box>
					</Box>
				)
			})}

			{total > tasks.length && (
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						gap: 1,
						px: 2.5,
						py: 1.5,
						bgcolor: '#f9fafb',
					}}
				>
					<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
						Кликните на заявку, чтобы открыть её
					</Typography>
					<Button onClick={onShowMore} disabled={isFetching}>
						Показать ещё
					</Button>
				</Box>
			)}
		</>
	)
}
