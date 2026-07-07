import { Box, Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown, Check, Tag, Building2, MessageSquare } from 'lucide-mui'
import { useState } from 'react'

import type { ITask, TicketStatus } from '../../types/task'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { STATUS_MAP } from '../../constants/taskMaps'

interface Props {
	task: ITask
	onStatusChange: (taskId: string, status: TicketStatus) => void
}

export const InfoBar = ({ task, onStatusChange }: Props) => {
	const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

	const changeStatus = (status: TicketStatus) => {
		onStatusChange(task.id, status)
		setAnchorEl(null)
	}

	return (
		<Box
			sx={{
				display: 'flex',
				flexWrap: 'wrap',
				alignItems: 'center',
				justifyContent: 'space-between',
				gap: 2,
				pt: 2,
			}}
		>
			<Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1.5 }}>
				<TaskStatusBadge status={task.status} sx={{ height: 33 }} />
				<TaskPriorityBadge priority={task.priority} />

				{task.category && (
					<Box
						sx={{
							display: 'inline-flex',
							alignItems: 'center',
							gap: 0.5,
							px: 1.5,
							py: 0.5,
							height: 33,
							borderRadius: '999px',
							fontSize: '0.75rem',
							fontWeight: 500,
							bgcolor: '#f3e8ff',
							color: '#6b21a8',
						}}
					>
						<Tag sx={{ fontSize: 14 }} />
						{task.category.name}
					</Box>
				)}

				{task.site && (
					<Box
						sx={{
							display: 'inline-flex',
							alignItems: 'center',
							gap: 0.5,
							px: 1.5,
							py: 0.5,
							height: 33,
							borderRadius: '999px',
							fontSize: '0.75rem',
							fontWeight: 500,
							bgcolor: '#f3f4f6',
							color: '#374151',
						}}
					>
						<Building2 sx={{ fontSize: 14 }} />
						{task.site.name}
					</Box>
				)}
			</Box>

			<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5 }}>
				{/* <Button
					variant='outlined'
					size='small'
					sx={{
						borderRadius: '8px',
						textTransform: 'none',
						fontSize: '0.8125rem',
						color: '#374151',
						borderColor: '#d1d5db',
						gap: 0.5,
						'&:hover': { bgcolor: '#f9fafb', borderColor: '#9ca3af' },
					}}
				>
					<MessageSquare sx={{ fontSize: 14 }} />
					Комментарий
				</Button> */}

				<Button
					variant='outlined'
					onClick={e => setAnchorEl(e.currentTarget)}
					sx={{
						textTransform: 'none',
						boxShadow: 'none',
						'&:hover': { boxShadow: 'none' },
					}}
					endIcon={<ChevronDown sx={{ fontSize: 14 }} />}
				>
					Изменить статус
				</Button>

				<Menu
					anchorEl={anchorEl}
					open={Boolean(anchorEl)}
					onClose={() => setAnchorEl(null)}
					slotProps={{
						paper: {
							sx: { borderRadius: '12px', mt: 0.5, minWidth: 200 },
						},
					}}
				>
					{(Object.entries(STATUS_MAP) as [TicketStatus, (typeof STATUS_MAP)[TicketStatus]][]).map(
						([value, info]) => (
							<MenuItem
								key={value}
								selected={value === task.status}
								onClick={() => changeStatus(value)}
								sx={{ fontSize: '0.875rem', gap: 1.5 }}
							>
								<info.icon sx={{ fontSize: 16, color: info.textColor }} />
								{info.label}
								{value === task.status && (
									<Box sx={{ ml: 'auto', color: 'primary.main', display: 'flex' }}>
										<Check sx={{ fontSize: 14 }} />
									</Box>
								)}
							</MenuItem>
						),
					)}
				</Menu>
			</Box>
		</Box>
	)
}
