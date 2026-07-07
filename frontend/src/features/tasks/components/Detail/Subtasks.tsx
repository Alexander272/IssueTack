import { Box, FormControl, MenuItem, Select, Typography } from '@mui/material'
import { CheckCircle, Clock, Circle, ListCheck } from 'lucide-mui'

import type { ISubtask, TicketStatus } from '../../types/task'

interface Props {
	subtasks: ISubtask[] | undefined
	onSubtaskStatusChange: (taskId: string, subtaskId: string, status: TicketStatus) => void
	taskId: string
}

const SUBTASK_STATUS_OPTIONS: { value: TicketStatus; label: string }[] = [
	{ value: 'open', label: 'Открыта' },
	{ value: 'in_progress', label: 'В работе' },
	{ value: 'closed', label: 'Выполнена' },
]

export const Subtasks = ({ subtasks, onSubtaskStatusChange, taskId }: Props) => {
	const done = subtasks ? subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length : 0
	const total = subtasks?.length ?? 0
	const progress = total > 0 ? Math.round((done / total) * 100) : 0

	if (!subtasks || subtasks.length === 0) return null

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2.5, py: 2, borderBottom: '1px solid #e5e7eb' }}>
				<Typography sx={{ fontWeight: 700, color: '#1f2937', fontSize: '0.9375rem', display: 'flex', alignItems: 'center', gap: 1 }}>
					<ListCheck sx={{ fontSize: 16 }} />
					Подзадачи
					<Typography component='span' sx={{ fontSize: '0.75rem', bgcolor: '#e5e7eb', color: '#374151', px: 1, py: 0.25, borderRadius: '999px' }}>
						{done}/{total}
					</Typography>
				</Typography>
			</Box>

			<Box sx={{}}>
				{subtasks.map(sub => (
					<Box
						key={sub.id}
						sx={{
							display: 'flex',
							alignItems: 'center',
							gap: 2,
							px: 2.5,
							py: 1.5,
							borderBottom: '1px solid #f3f4f6',
							'&:last-of-type': { borderBottom: 'none' },
							...(sub.status === 'in_progress' ? { bgcolor: 'rgba(239,246,255,0.5)' } : {}),
						}}
					>
						{sub.status === 'closed' || sub.status === 'resolved' ? (
							<CheckCircle sx={{ fontSize: 18, color: '#10b981', flexShrink: 0 }} />
						) : sub.status === 'in_progress' ? (
							<Clock sx={{ fontSize: 16, color: '#f59e0b', flexShrink: 0 }} />
						) : (
							<Circle sx={{ fontSize: 18, color: '#d1d5db', flexShrink: 0 }} />
						)}

						<Box sx={{ flex: 1, minWidth: 0 }}>
							<Typography
								sx={{
									fontSize: '0.875rem',
									color: sub.status === 'closed' || sub.status === 'resolved' ? '#9ca3af' : '#1f2937',
									...(sub.status === 'closed' || sub.status === 'resolved' ? { textDecoration: 'line-through' } : {}),
								}}
							>
								{sub.title}
							</Typography>
							<Box sx={{ display: 'flex', gap: 1.5, mt: 0.25 }}>
								{sub.assignee && (
									<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
										{sub.assignee.lastName} {sub.assignee.firstName}
									</Typography>
								)}
								{sub.closedAt && (
									<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
										{sub.closedAt}
									</Typography>
								)}
							</Box>
						</Box>

						<FormControl size='small'>
							<Select
								value={sub.status}
								onChange={e => onSubtaskStatusChange(taskId, sub.id, e.target.value as TicketStatus)}
								sx={{
									borderRadius: '999px',
									fontSize: '0.75rem',
									'& .MuiOutlinedInput-notchedOutline': { borderColor: '#d1d5db' },
								}}
							>
								{SUBTASK_STATUS_OPTIONS.map(opt => (
									<MenuItem key={opt.value} value={opt.value}>
										{opt.label}
									</MenuItem>
								))}
							</Select>
						</FormControl>
					</Box>
				))}
			</Box>

			{total > 0 && (
				<Box sx={{ px: 2.5, py: 2, bgcolor: '#f9fafb', borderTop: '1px solid #e5e7eb' }}>
					<Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.75 }}>
						<Typography sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 500 }}>Общий прогресс</Typography>
						<Typography sx={{ fontSize: '0.75rem', color: '#374151', fontWeight: 600 }}>{progress}%</Typography>
					</Box>
					<Box sx={{ width: '100%', height: 8, bgcolor: '#e5e7eb', borderRadius: '999px', overflow: 'hidden' }}>
						<Box sx={{ width: `${progress}%`, height: '100%', bgcolor: 'primary.main', borderRadius: '999px', transition: 'width 0.3s' }} />
					</Box>
				</Box>
			)}
		</Box>
	)
}
