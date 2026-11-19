import { useState } from 'react'
import { Box, Typography } from '@mui/material'
import { ChevronDownIcon } from 'lucide-mui'

import type { ITask } from '../../types/task'
import { TaskCard } from './TaskCard'
import type { TaskListViewState } from './taskView'
import { useGroupedTasks } from './useGroupedTasks'

interface Props {
	tasks: ITask[]
	view: TaskListViewState
	onTaskClick: (task: ITask) => void
}

export const TaskCardList = ({ tasks, view, onTaskClick }: Props) => {
	const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())
	const { groups, sortedKeys } = useGroupedTasks(tasks, view)

	const toggleGroup = (key: string) => {
		setCollapsedGroups(prev => {
			const next = new Set(prev)
			if (next.has(key)) next.delete(key)
			else next.add(key)
			return next
		})
	}

	if (!view.groupEnabled) {
		return (
			<Box
				sx={{
					display: 'grid',
					gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
					gap: 2.5,
				}}
			>
				{tasks.map(task => (
					<TaskCard key={task.id} task={task} view={view} onClick={onTaskClick} />
				))}
			</Box>
		)
	}

	return (
		<>
			{sortedKeys.map(key => {
				const groupTasks = groups[key]
				const isCollapsed = collapsedGroups.has(key)
				return (
					<Box key={key} sx={{ mb: 3 }}>
						<Box
							onClick={() => toggleGroup(key)}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1,
								mb: 1,
								cursor: 'pointer',
								bgcolor: '#f9fafb',
								borderRadius: 2,
								px: 2,
								py: 1.5,
								'&:hover': { bgcolor: '#f3f4f6' },
							}}
						>
							<ChevronDownIcon
								sx={{
									fontSize: 10,
									color: '#6b7280',
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

						{!isCollapsed && (
							<Box
								sx={{
									display: 'grid',
									gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
									gap: 2.5,
								}}
							>
								{groupTasks.map(task => (
									<TaskCard key={task.id} task={task} view={view} onClick={onTaskClick} />
								))}
							</Box>
						)}
					</Box>
				)
			})}
		</>
	)
}
