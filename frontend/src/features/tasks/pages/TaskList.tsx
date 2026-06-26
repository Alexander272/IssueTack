import { useState, useCallback, useMemo, useEffect } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'

import { Pagination } from '@/components/Pagination/Pagination'

import type { GroupByField } from '../constants/taskMaps'
import type { FilterValues } from '../components/filters'
import type { ITask, ITaskFilter, TicketStatus } from '../types/task'
import { useGetTasksQuery, useUpdateTaskMutation } from '../tasksApiSlice'
import { useUpdateSubtaskMutation } from '../modules/subtasks/subtasksApiSlice'
import { TaskFilters } from '../components/filters'
import { TaskTable } from '../components/Table'
import { TaskDetailModal } from '../components/TaskDetailModal'
import { TaskCreateModal } from '../components/TaskCreateModal'
import { PlusIcon } from '@/components/Icons/PlusIcon'

type Mode = 'created' | 'assigned'

type Props = {
	mode: Mode
}

const DEFAULT_FILTERS: FilterValues = {
	sort: 'dueDate_asc',
	search: '',
	groupBy: 'site',
	groupEnabled: true,

	ticketNumber: undefined,
	ownerId: undefined,
	siteIds: undefined,
	dueDateFrom: undefined,
	dueDateTo: undefined,
	priorities: undefined,
	assigneeId: undefined,
	statuses: undefined,
}

const PAGE_TITLE: Record<Mode, string> = {
	created: 'Заявки',
	assigned: 'Мои задачи',
}

const PAGE_DESC: Record<Mode, string> = {
	created: 'Созданные вами заявки',
	assigned: 'Задачи, назначенные лично или группам',
}

export const TaskList = ({ mode }: Props) => {
	const { palette } = useTheme()
	const rowsPerPage = 20

	const STORAGE_KEY = `taskFilters_${mode}`

	const loadFilters = (): FilterValues => {
		try {
			const saved = localStorage.getItem(STORAGE_KEY)
			if (saved) return { ...DEFAULT_FILTERS, ...JSON.parse(saved) }
		} catch {
			/* ignore */
		}
		return DEFAULT_FILTERS
	}

	const [filters, setFilters] = useState<FilterValues>(loadFilters)
	const [selectedTask, setSelectedTask] = useState<ITask | null>(null)
	const [createOpen, setCreateOpen] = useState(false)
	const [page, setPage] = useState(0)

	useEffect(() => {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(filters))
	}, [filters, STORAGE_KEY])

	const [updateTask] = useUpdateTaskMutation()
	const [updateSubtask] = useUpdateSubtaskMutation()

	const queryFilter: ITaskFilter = useMemo(() => ({
		number: filters.ticketNumber ? Number(filters.ticketNumber) : undefined,
		ownerId: filters.ownerId ?? undefined,
		assigneeId: filters.assigneeId ?? undefined,
		siteIds: filters.siteIds?.length ? filters.siteIds : undefined,
		priorities: filters.priorities?.length ? filters.priorities : undefined,
		statuses: filters.statuses?.length ? filters.statuses : undefined,
		dueDateFrom: filters.dueDateFrom || undefined,
		dueDateTo: filters.dueDateTo || undefined,
		search: filters.search || undefined,
		sort: filters.sort,
		mode,
		limit: filters.groupEnabled ? undefined : rowsPerPage,
		offset: filters.groupEnabled ? undefined : page * rowsPerPage,
	}), [filters, mode, page])

	const { data, isFetching } = useGetTasksQuery(queryFilter)

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const totalPages = filters.groupEnabled ? 1 : Math.ceil(total / rowsPerPage) || 1

	const handleFilterChange = useCallback((patch: Partial<FilterValues>) => {
		setFilters(prev => ({ ...prev, ...patch }))
		setPage(0)
	}, [])

	const handleReset = useCallback(() => {
		setFilters(DEFAULT_FILTERS)
		setPage(0)
		localStorage.removeItem(STORAGE_KEY)
	}, [STORAGE_KEY])

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
			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 0 },
					mb: 4,
				}}
			>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						{PAGE_TITLE[mode]}
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						{PAGE_DESC[mode]}
					</Typography>
				</Box>
				{mode === 'created' && (
					<Button
						variant='outlined'
						sx={{
							borderRadius: '8px',
							textTransform: 'none',
							background: '#fff',
							width: { xs: '100%', sm: 'auto' },
						}}
						onClick={() => setCreateOpen(true)}
					>
						<PlusIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
						Создать заявку
					</Button>
				)}
			</Box>

			<TaskFilters filters={filters} onChange={handleFilterChange} onReset={handleReset} />

			{isFetching && !data ? (
				<Box sx={{ textAlign: 'center', py: 6, color: '#6b7280' }}>Загрузка...</Box>
			) : (
				<>
					<TaskTable
						tasks={tasks}
						groupBy={filters.groupBy as GroupByField}
						groupEnabled={filters.groupEnabled}
						onTaskClick={handleTaskClick}
						sort={filters.sort}
						onSortChange={sort => handleFilterChange({ sort })}
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
							{filters.groupEnabled ? `Всего: ${total} задач` : `Показано ${tasks.length} из ${total} задач`}
						</Typography>
						{!filters.groupEnabled && (
							<Pagination page={page + 1} totalPages={totalPages} onClick={p => setPage(p - 1)} />
						)}
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

			<TaskCreateModal open={createOpen} onClose={() => setCreateOpen(false)} />
		</Box>
	)
}
