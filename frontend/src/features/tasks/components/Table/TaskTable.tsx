import {
	Table,
	TableBody,
	TableContainer,
	TableHead,
	TableRow,
	TableCell,
	Paper,
	Box,
	useTheme,
	useMediaQuery,
} from '@mui/material'
import { ArrowDownAZ, ArrowUpAZ } from 'lucide-mui'

import type { ITask } from '../../types/task'
import { TaskTableEmpty } from './TaskTableEmpty'
import { TaskTableGrouped } from './TaskTableGrouped'
import { TaskCardList } from './TaskCardList'
import { TaskRow } from './TaskRow'
import { TASK_COLUMNS, type TaskColumn, type TaskListViewState } from './taskView'

interface Props {
	tasks: ITask[]
	view: TaskListViewState
	onTaskClick: (task: ITask) => void
}

export const TaskTable = ({ tasks, view, onTaskClick }: Props) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('md'))

	const columns: TaskColumn[] = TASK_COLUMNS.filter(col => view.columns.has(col.field))
	const tableMinWidth = columns.reduce((sum, col) => sum + (col.width ?? 240), 0)

	const currentBase = view.sort.replace(/_(asc|desc)$/, '')
	const isDesc = view.sort.endsWith('_desc')

	const handleSort = (field: string) => {
		if (currentBase === field) {
			view.onSortChange(`${field}_${isDesc ? 'asc' : 'desc'}`)
		} else {
			view.onSortChange(`${field}_asc`)
		}
	}

	if (isMobile) {
		return <TaskCardList tasks={tasks} view={view} onTaskClick={onTaskClick} />
	}

	const headRow = (
		<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
			{columns.map(col => {
				const cellSx: Record<string, unknown> = {
					color: 'text.secondary',
					fontSize: '0.875rem',
				}

				if (col.width) {
					cellSx.width = col.width
					cellSx.minWidth = col.width
				}

				if (!col.sortable) {
					return (
						<TableCell key={col.field} sx={cellSx} align='right'>
							{col.label}
						</TableCell>
					)
				}

				const isActive = currentBase === col.field
				return (
					<TableCell
						key={col.field}
						sx={{
							...cellSx,
							cursor: 'pointer',
							userSelect: 'none',
							'&:hover': { color: 'text.primary' },
						}}
						onClick={() => handleSort(col.field)}
					>
						{col.label}
						{isActive && (
							<Box component='span' sx={{ ml: 0.5 }}>
								{isDesc ? (
									<ArrowDownAZ sx={{ fontSize: 16, mb: -0.5 }} />
								) : (
									<ArrowUpAZ sx={{ fontSize: 16, mb: -0.5 }} />
								)}
							</Box>
						)}
					</TableCell>
				)
			})}
		</TableRow>
	)

	const columnsCount = columns.length

	const body = !tasks.length ? (
		<TaskTableEmpty columnsCount={columnsCount} />
	) : view.groupEnabled ? (
		<TaskTableGrouped tasks={tasks} view={view} onTaskClick={onTaskClick} columnsCount={columnsCount} />
	) : (
		tasks.map(task => <TaskRow key={task.id} task={task} view={view} onClick={onTaskClick} />)
	)

	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: 3, border: '1px solid #f3f4f6', overflow: 'hidden', overflowX: 'auto' }}
		>
			<Table sx={{ tableLayout: 'fixed', minWidth: tableMinWidth }}>
				<TableHead>{headRow}</TableHead>
				<TableBody>{body}</TableBody>
			</Table>
		</TableContainer>
	)
}
