import { useCallback } from 'react'
import { Box, Typography } from '@mui/material'
import { useNavigate } from 'react-router'

import { TASKS_PAGE_SIZE, type GroupByField } from '../../constants/taskMaps'
import type { ITask, ITaskFilter } from '../../types/task'
import { Pagination } from '@/components/Pagination/Pagination'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { TaskTable } from './TaskTable'
import { useTaskListView } from './taskView'
import { useGetTasksQuery } from '../../tasksApiSlice'
import { AppRoutes } from '@/pages/router/routes'

interface Props {
	filters: ITaskFilter
	isArchive: boolean
	groupBy: GroupByField
	groupEnabled: boolean
	sort: string
	onSortChange: (sort: string) => void
	page: number
	onPageChange: (page: number) => void
}

export const TaskListView = ({
	filters,
	isArchive,
	groupBy,
	groupEnabled,
	sort,
	onSortChange,
	page,
	onPageChange,
}: Props) => {
	const { data, isFetching } = useGetTasksQuery(filters)
	const navigate = useNavigate()

	// Один объект представления на всю ветку таблицы/карточек: шапка, строки и
	// карточки больше не получают isArchive/groupBy/флаги по отдельности.
	const view = useTaskListView({ isArchive, groupBy, groupEnabled, sort, onSortChange })

	const handleTaskClick = useCallback((task: ITask) => navigate(`${AppRoutes.Tasks}/${task.id}`), [navigate])

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const totalPages = isArchive ? Math.ceil(total / TASKS_PAGE_SIZE) || 1 : 1

	if (isFetching && !tasks.length) {
		return <BoxFallback />
	}

	return (
		<>
			<TaskTable tasks={tasks} view={view} onTaskClick={handleTaskClick} />

			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					alignItems: 'center',
					px: 1,
					py: 1.5,
					gap: { xs: 1, sm: 0 },
				}}
			>
				<Typography
					sx={{
						flex: 1,
						fontSize: '0.875rem',
						color: '#6b7280',
						textAlign: { xs: 'center', sm: 'left' },
					}}
				>
					{isArchive ? `Показано ${tasks.length} из ${total} задач` : `Всего: ${total} задач`}
				</Typography>
				{isArchive && totalPages > 1 ? (
					<Pagination page={page + 1} totalPages={totalPages} onClick={onPageChange} />
				) : null}
				<Box sx={{ flex: 1, display: { xs: 'none', sm: 'block' } }} />
			</Box>
		</>
	)
}
