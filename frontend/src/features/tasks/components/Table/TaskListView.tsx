import { Box, Typography } from '@mui/material'

import type { GroupByField } from '../../constants/taskMaps'
import type { ITaskFilter } from '../../types/task'
import { Pagination } from '@/components/Pagination/Pagination'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { TaskTable } from './TaskTable'
import { useGetTasksQuery } from '../../tasksApiSlice'

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
	const rowsPerPage = 20

	const { data, isFetching } = useGetTasksQuery(filters)

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const totalPages = isArchive ? Math.ceil(total / rowsPerPage) || 1 : 1

	if (isFetching && !tasks.length) {
		return <BoxFallback />
	}

	return (
		<>
			<TaskTable
				tasks={tasks}
				groupBy={groupBy}
				groupEnabled={groupEnabled}
				sort={sort}
				isArchive={isArchive}
				onSortChange={onSortChange}
			/>

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
