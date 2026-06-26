import { Table, TableBody, TableContainer, TableHead, TableRow, TableCell, Paper, Box, useTheme, useMediaQuery } from '@mui/material'

import type { ITask } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'
import { TaskTableEmpty } from './TaskTableEmpty'
import { TaskTableBody } from './TaskTableBody'
import { TaskTableGrouped } from './TaskTableGrouped'
import { TaskCardList } from './TaskCardList'

interface Props {
	tasks: ITask[]
	groupBy: GroupByField
	groupEnabled: boolean
	onTaskClick: (task: ITask) => void
	sort: string
	onSortChange: (sort: string) => void
}

type Column = { field: string; label: string; sortable: boolean; width?: number }
const COLUMNS: readonly Column[] = [
	{ field: 'ticketNumber', label: '№', sortable: true, width: 60 },
	{ field: 'title', label: 'Тема', sortable: true },
	{ field: 'owner', label: 'Заказчик', sortable: true, width: 200 },
	{ field: 'site', label: 'Площадка', sortable: true, width: 160 },
	{ field: 'dueDate', label: 'Срок', sortable: true, width: 120 },
	{ field: 'priority', label: 'Приоритет', sortable: true, width: 130 },
	{ field: 'assignee', label: 'Назначено', sortable: true, width: 200 },
	{ field: 'status', label: 'Статус', sortable: true, width: 150 },
	{ field: 'subtasks', label: 'Подзадачи', sortable: false, width: 130 },
]

export const TaskTable = ({ tasks, groupBy, groupEnabled, onTaskClick, sort, onSortChange }: Props) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('md'))

	const currentBase = sort.replace(/_(asc|desc)$/, '')
	const isDesc = sort.endsWith('_desc')

	const handleSort = (field: string) => {
		if (currentBase === field) {
			onSortChange(`${field}_${isDesc ? 'asc' : 'desc'}`)
		} else {
			onSortChange(`${field}_asc`)
		}
	}

	if (isMobile) {
		return <TaskCardList tasks={tasks} groupBy={groupBy} groupEnabled={groupEnabled} onTaskClick={onTaskClick} />
	}

	const headRow = (
		<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
			{COLUMNS.map(col => {
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
							<Box component='span' sx={{ ml: 0.5, fontSize: '0.625rem' }}>
								{isDesc ? '▼' : '▲'}
							</Box>
						)}
					</TableCell>
				)
			})}
		</TableRow>
	)

	const columnsCount = COLUMNS.length

	const body = !tasks.length ? (
		<TaskTableEmpty columnsCount={columnsCount} />
	) : !groupEnabled ? (
		<TaskTableBody tasks={tasks} onTaskClick={onTaskClick} />
	) : (
		<TaskTableGrouped tasks={tasks} groupBy={groupBy} onTaskClick={onTaskClick} columnsCount={columnsCount} />
	)

	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: 3, border: '1px solid #f3f4f6', overflow: 'hidden', overflowX: 'auto' }}
		>
			<Table sx={{ minWidth: tasks.length > 0 ? 900 : undefined }}>
				<TableHead>{headRow}</TableHead>
				<TableBody>{body}</TableBody>
			</Table>
		</TableContainer>
	)
}
