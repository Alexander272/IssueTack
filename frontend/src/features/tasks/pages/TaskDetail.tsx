import { useState } from 'react'
import { Box, CircularProgress, Grid, Typography } from '@mui/material'
import { useParams } from 'react-router'

import type { TicketStatus } from '../types/task'
import { useGetTaskByIdQuery, useUpdateTaskMutation } from '../tasksApiSlice'
import { useUpdateSubtaskMutation } from '../modules/subtasks/subtasksApiSlice'
import { Header } from '../components/Detail/Header'
import { InfoBar } from '../components/Detail/InfoBar'
import { Description } from '../components/Detail/Description'
import { Subtasks } from '../components/Detail/Subtasks'
import { Attachments } from '../components/Detail/Attachments'
import { Comments } from '../components/Detail/Comments'
import { Participants } from '../components/Detail/Participants'
import { Meta } from '../components/Detail/Meta'
import { Notifications } from '../components/Detail/Notifications'
import { TaskEditModal } from '../components/TaskEditModal'

export const TaskDetailPage = () => {
	const { id } = useParams<{ id: string }>()
	const { data, isLoading } = useGetTaskByIdQuery(id!)
	const [updateTask] = useUpdateTaskMutation()
	const [updateSubtask] = useUpdateSubtaskMutation()
	const [editOpen, setEditOpen] = useState(false)

	if (isLoading) {
		return (
			<Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
				<CircularProgress />
			</Box>
		)
	}

	if (!data?.data) {
		return (
			<Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
				<Typography sx={{ color: '#9ca3af' }}>Задача не найдена</Typography>
			</Box>
		)
	}

	const task = data.data
	const canEdit = task.status === 'open' && task.access?.canWrite

	const handleStatusChange = async (taskId: string, status: TicketStatus) => {
		try {
			await updateTask({ id: taskId, status })
		} catch {
			// handled by toast in apiSlice
		}
	}

	const handleSubtaskStatusChange = async (taskId: string, subtaskId: string, status: TicketStatus) => {
		try {
			await updateSubtask({ ticketId: taskId, id: subtaskId, status })
		} catch {
			// handled by toast in apiSlice
		}
	}

	return (
		<>
		<Box sx={{ flexGrow: 1, overflow: 'auto', bgcolor: '#f9fafb' }}>
			<Box sx={{ maxWidth: 'xl', mx: 'auto', p: 3 }}>
				<Box
					sx={{
						gap: 2,
						p: 2.5,
						bgcolor: 'white',
						borderRadius: '12px',
						border: '1px solid #e5e7eb',
					}}
				>
					<Header task={task} />

					<InfoBar task={task} onStatusChange={handleStatusChange} />
				</Box>

				<Grid container spacing={3} sx={{ mt: 2 }}>
					<Grid size={{ xs: 12, lg: 8 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						<Description description={task.description} onEdit={canEdit ? () => setEditOpen(true) : undefined} />
						<Subtasks
							subtasks={task.subtasks}
							taskId={task.id}
							canWork={task.access?.canWork}
							onSubtaskStatusChange={handleSubtaskStatusChange}
						/>
						<Attachments attachments={task.attachments} />
						<Comments comments={[]} taskId={task.id} />
					</Grid>

					<Grid size={{ xs: 12, lg: 4 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						<Participants task={task} />
						<Meta task={task} />
						<Notifications />
					</Grid>
				</Grid>
			</Box>
		</Box>

		<TaskEditModal open={editOpen} onClose={() => setEditOpen(false)} task={task} />
		</>
	)
}
