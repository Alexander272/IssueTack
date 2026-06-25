import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Typography, Box } from '@mui/material'

import type { ITask } from '../types/task'
import type { GroupByField } from '../constants/taskMaps'
import { STATUS_MAP, PRIORITY_MAP } from '../constants/taskMaps'
import { TaskRow } from './TaskRow'

interface Props {
	tasks: ITask[]
	groupBy: GroupByField
	groupEnabled: boolean
	onTaskClick: (task: ITask) => void
}

function getGroupValue(task: ITask, groupBy: GroupByField): string {
	switch (groupBy) {
		case 'category':
			return task.category.name
		case 'status':
			return STATUS_MAP[task.status]?.label ?? task.status
		case 'priority':
			return PRIORITY_MAP[task.priority]?.label ?? task.priority
		case 'assignee': {
			if (task.assignee) return `👤 ${task.assignee.fullName}`
			if (task.group) return `👥 ${task.group.name}`
			return 'Без назначения'
		}
		case 'creator':
			return task.creator?.fullName ?? 'Без заказчика'
		case 'dueDate': {
			if (!task.dueDate) return 'Без срока'
			const date = new Date(task.dueDate)
			const week = Math.ceil(
				(date.getTime() - new Date(date.getFullYear(), 0, 1).getTime()) / (7 * 24 * 60 * 60 * 1000),
			)
			return `${date.getFullYear()} неделя ${week}`
		}
		default:
			return task.category.name
	}
}

export const TaskTable = ({ tasks, groupBy, groupEnabled, onTaskClick }: Props) => {
	if (tasks.length === 0) {
		return (
			<TableContainer
				component={Paper}
				elevation={0}
				sx={{ borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}
			>
				<Table>
					<TableHead>
						<TableRow sx={{ bgcolor: '#f9fafb', borderBottom: '1px solid #e5e7eb' }}>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								ID / Тема
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Заказчик
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Категория
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Назначено
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Статус
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Приоритет
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
							>
								Срок
							</TableCell>
							<TableCell
								sx={{
									px: 3,
									py: 2.5,
									color: '#6b7280',
									fontSize: '0.75rem',
									fontWeight: 600,
									textTransform: 'uppercase',
									letterSpacing: '0.05em',
								}}
								align='right'
							>
								Подзадачи
							</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						<TableRow>
							<TableCell colSpan={8} align='center' sx={{ py: 6, color: '#6b7280' }}>
								Нет задач по выбранным фильтрам
							</TableCell>
						</TableRow>
					</TableBody>
				</Table>
			</TableContainer>
		)
	}

	const headRow = (
		<TableRow sx={{ bgcolor: '#f9fafb', borderBottom: '1px solid #e5e7eb' }}>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				ID / Тема
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Заказчик
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Категория
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Назначено
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Статус
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Приоритет
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
			>
				Срок
			</TableCell>
			<TableCell
				sx={{
					px: 3,
					py: 2.5,
					color: '#6b7280',
					fontSize: '0.75rem',
					fontWeight: 600,
					textTransform: 'uppercase',
					letterSpacing: '0.05em',
				}}
				align='right'
			>
				Подзадачи
			</TableCell>
		</TableRow>
	)

	if (!groupEnabled) {
		return (
			<TableContainer
				component={Paper}
				elevation={0}
				sx={{ borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}
			>
				<Table sx={{ minWidth: 900 }}>
					<TableHead>{headRow}</TableHead>
					<TableBody>
						{tasks.map(task => (
							<TaskRow key={task.id} task={task} onClick={onTaskClick} />
						))}
					</TableBody>
				</Table>
			</TableContainer>
		)
	}

	const groups: Record<string, ITask[]> = {}
	tasks.forEach(task => {
		const key = getGroupValue(task, groupBy)
		if (!groups[key]) groups[key] = []
		groups[key].push(task)
	})

	const sortedKeys = Object.keys(groups).sort((a, b) => a.localeCompare(b))

	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}
		>
			<Table sx={{ minWidth: 900 }}>
				<TableHead>{headRow}</TableHead>
				<TableBody>
					{sortedKeys.flatMap(key => {
						const groupTasks = groups[key]
						return [
							<TableRow key={`group-${key}`} sx={{ bgcolor: '#f3f4f6' }}>
								<TableCell colSpan={8} sx={{ px: 3, py: 2 }}>
									<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
										<Typography
											sx={{
												fontWeight: 700,
												color: '#374151',
												fontSize: '0.75rem',
												textTransform: 'uppercase',
												letterSpacing: '0.05em',
											}}
										>
											{key}
										</Typography>
										<Typography sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 500 }}>
											({groupTasks.length})
										</Typography>
									</Box>
								</TableCell>
							</TableRow>,
							...groupTasks.map(task => <TaskRow key={task.id} task={task} onClick={onTaskClick} />),
						]
					})}
				</TableBody>
			</Table>
		</TableContainer>
	)
}
