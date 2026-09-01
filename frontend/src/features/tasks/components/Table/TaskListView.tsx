import { Box, Typography } from '@mui/material'

import type { GroupByField } from '../../constants/taskMaps'
import type { ITask } from '../../types/task'
import { Pagination } from '@/components/Pagination/Pagination'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { TaskTable } from './TaskTable'

interface Props {
	tasks: ITask[]
	isFetching: boolean
	total: number
	isArchive: boolean
	groupBy: GroupByField
	groupEnabled: boolean
	onTaskClick: (task: ITask) => void
	sort: string
	onSortChange: (sort: string) => void
	page: number
	totalPages: number
	onPageChange: (page: number) => void
}

export const TaskListView = ({
	tasks,
	isFetching,
	total,
	isArchive,
	groupBy,
	groupEnabled,
	onTaskClick,
	sort,
	onSortChange,
	page,
	totalPages,
	onPageChange,
}: Props) => {
	if (isFetching && !tasks.length) {
		return <BoxFallback />
	}

	return (
		<>
			<TaskTable
				tasks={tasks}
				groupBy={groupBy}
				groupEnabled={groupEnabled}
				onTaskClick={onTaskClick}
				sort={sort}
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
