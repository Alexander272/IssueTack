import { useState } from 'react'
import { TableCell, TableRow, Typography, Box } from '@mui/material'
import { BottomArrowIcon } from '@/components/Icons/BottomArrowIcon'

import type { ITask } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'
import { TaskRow } from './TaskRow'
import { getGroupValue } from './getGroupValue'

interface Props {
	tasks: ITask[]
	groupBy: GroupByField
	onTaskClick: (task: ITask) => void
	columnsCount: number
}

export const TaskTableGrouped = ({ tasks, groupBy, onTaskClick, columnsCount }: Props) => {
	const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())

	const toggleGroup = (key: string) => {
		setCollapsedGroups(prev => {
			const next = new Set(prev)
			if (next.has(key)) next.delete(key)
			else next.add(key)
			return next
		})
	}

	const groups: Record<string, ITask[]> = {}
	tasks.forEach(task => {
		const key = getGroupValue(task, groupBy)
		if (!groups[key]) groups[key] = []
		groups[key].push(task)
	})

	const sortedKeys = Object.keys(groups).sort((a, b) => a.localeCompare(b))

	return (
		<>
			{sortedKeys.flatMap(key => {
				const groupTasks = groups[key]
				const isCollapsed = collapsedGroups.has(key)
				return [
					<TableRow
						key={`group-${key}`}
						sx={{ bgcolor: '#f9fafb', cursor: 'pointer', '&:hover': { bgcolor: '#f3f4f6' } }}
						onClick={() => toggleGroup(key)}
					>
						<TableCell colSpan={columnsCount} sx={{ px: 3, py: 2 }}>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
								<BottomArrowIcon
									sx={{
										fontSize: 10,
										fill: '#6b7280',
										transform: isCollapsed ? 'rotate(-90deg)' : 'rotate(0deg)',
										transition: 'transform 0.2s',
										flexShrink: 0,
									}}
								/>
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
					...(isCollapsed
						? []
						: groupTasks.map(task => <TaskRow key={task.id} task={task} onClick={onTaskClick} />)),
				]
			})}
		</>
	)
}
