import { useState, useCallback, useMemo } from 'react'
import { Box, Typography } from '@mui/material'

import { Pagination } from '@/components/Pagination/Pagination'

import { useAppSelector } from '@/hooks/redux'
import { useGetTasksQuery, useUpdateTaskMutation } from '../tasksApiSlice'
import { useUpdateSubtaskMutation } from '../modules/subtasks/subtasksApiSlice'
import type { ITask, TicketStatus } from '../types/task'
import type { GroupByField } from '../constants/taskMaps'
import type { FilterValues } from '../components/TaskFilters'
import { TaskFilters } from '../components/TaskFilters'
import { TaskTable } from '../components/TaskTable'
import { TaskDetailModal } from '../components/TaskDetailModal'

const DEFAULT_FILTERS: FilterValues = {
	queue: 'all',
	status: 'all',
	sort: 'dueDate_asc',
	search: '',
	groupBy: 'category',
	groupEnabled: true,
}

function getSortValue(task: ITask, sortKey: string): string | number {
	if (sortKey === 'dueDate_asc' || sortKey === 'dueDate_desc') return task.dueDate || ''
	if (sortKey === 'closedAt_asc' || sortKey === 'closedAt_desc') return task.closedAt || ''
	if (sortKey === 'priority_asc' || sortKey === 'priority_desc') {
		const w: Record<string, number> = { urgent: 4, high: 3, medium: 2, low: 1 }
		return w[task.priority] || 0
	}
	if (sortKey === 'status') return task.status
	return ''
}

function compareTasks(a: ITask, b: ITask, sortKey: string): number {
	const aVal = getSortValue(a, sortKey)
	const bVal = getSortValue(b, sortKey)

	if (
		sortKey === 'dueDate_asc' ||
		sortKey === 'dueDate_desc' ||
		sortKey === 'closedAt_asc' ||
		sortKey === 'closedAt_desc'
	) {
		if (!aVal && !bVal) return 0
		if (!aVal) return 1
		if (!bVal) return -1
		const cmp = String(aVal).localeCompare(String(bVal))
		return sortKey.endsWith('_desc') ? -cmp : cmp
	}

	if (sortKey === 'priority_asc' || sortKey === 'priority_desc') {
		const diff = Number(aVal) - Number(bVal)
		return sortKey.endsWith('_desc') ? -diff : diff
	}

	return String(aVal).localeCompare(String(bVal))
}

export const TaskList = () => {
	const currentUserId = useAppSelector(state => state.user.id)

	const [filters, setFilters] = useState<FilterValues>(DEFAULT_FILTERS)
	const [selectedTask, setSelectedTask] = useState<ITask | null>(null)
	const [page, setPage] = useState(0)
	const rowsPerPage = 20

	const [updateTask] = useUpdateTaskMutation()
	const [updateSubtask] = useUpdateSubtaskMutation()

	const queryFilter = useMemo(() => {
		const qf: Record<string, string | number | undefined> = {}
		if (filters.status !== 'all') qf.status = filters.status
		return qf
	}, [filters.status])

	const { data, isFetching } = useGetTasksQuery(queryFilter)

	const filteredAndSorted = useMemo(() => {
		let tasks = data?.data ?? []

		if (filters.queue === 'personal' && currentUserId) {
			tasks = tasks.filter(t => t.assignee?.id === currentUserId)
		} else if (filters.queue === 'group1') {
			tasks = tasks.filter(t => t.group?.name === 'IT-поддержка')
		} else if (filters.queue === 'group2') {
			tasks = tasks.filter(t => t.group?.name === 'Сетевая администрация')
		}

		if (filters.search) {
			const q = filters.search.toLowerCase().trim()
			tasks = tasks.filter(t => t.id.toLowerCase().includes(q) || t.title.toLowerCase().includes(q))
		}

		tasks.sort((a, b) => compareTasks(a, b, filters.sort))

		return tasks
	}, [data, filters, currentUserId])

	const paginatedTasks = useMemo(
		() => filteredAndSorted.slice(page * rowsPerPage, (page + 1) * rowsPerPage),
		[filteredAndSorted, page, rowsPerPage],
	)

	const handleFilterChange = useCallback((patch: Partial<FilterValues>) => {
		setFilters(prev => ({ ...prev, ...patch }))
		setPage(0)
	}, [])

	const handleReset = useCallback(() => {
		setFilters(DEFAULT_FILTERS)
		setPage(0)
	}, [])

	const handleTaskClick = useCallback((task: ITask) => {
		setSelectedTask(task)
	}, [])

	const handleStatusChange = useCallback(
		async (taskId: string, status: TicketStatus) => {
			try {
				await updateTask({ id: taskId, status } as any)
				setSelectedTask(prev => (prev?.id === taskId ? { ...prev, status } : prev))
			} catch {
				// handled by toast in apiSlice
			}
		},
		[updateTask],
	)

	const handleSubtaskStatusChange = useCallback(
		async (taskId: string, subtaskId: string, status: TicketStatus) => {
			try {
				await updateSubtask({ ticketId: taskId, id: subtaskId, status } as any)
				setSelectedTask(prev => {
					if (!prev || prev.id !== taskId) return prev
					return {
						...prev,
						subtasks: prev.subtasks?.map(s => (s.id === subtaskId ? { ...s, status } : s)),
					}
				})
			} catch {
				// handled by toast in apiSlice
			}
		},
		[updateSubtask],
	)

	return (
		<Box sx={{ p: 3 }}>
			<Box sx={{ mb: 3 }}>
				<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
					Мое рабочее место
				</Typography>
				<Typography variant='body2' sx={{ color: '#6b7280' }}>
					Задачи, назначенные лично или группам
				</Typography>
			</Box>

			<TaskFilters filters={filters} onChange={handleFilterChange} onReset={handleReset} />

			{isFetching && filteredAndSorted.length === 0 ? (
				<Box sx={{ textAlign: 'center', py: 6, color: '#6b7280' }}>Загрузка...</Box>
			) : (
				<>
					<TaskTable
						tasks={paginatedTasks}
						groupBy={filters.groupBy as GroupByField}
						groupEnabled={filters.groupEnabled}
						onTaskClick={handleTaskClick}
					/>

					<Box
						sx={{
							display: 'flex',
							alignItems: 'center',
							justifyContent: 'space-between',
							px: 1,
							py: 1.5,
						}}
					>
						<Typography sx={{ fontSize: '0.875rem', color: '#6b7280' }}>
							Показано {filteredAndSorted.length} задач
						</Typography>
						<Pagination
							page={page + 1}
							totalPages={Math.ceil(filteredAndSorted.length / rowsPerPage) || 1}
							onClick={p => setPage(p - 1)}
						/>
					</Box>
				</>
			)}

			<TaskDetailModal
				open={!!selectedTask}
				task={selectedTask}
				onClose={() => setSelectedTask(null)}
				onStatusChange={handleStatusChange}
				onSubtaskStatusChange={handleSubtaskStatusChange}
			/>
		</Box>
	)
}
