import { useState } from 'react'
import { Box, Grid } from '@mui/material'
import { useParams } from 'react-router'

import type { TicketStatus } from '../types/task'
import { useGetTaskByIdQuery, useUpdateTaskMutation, useTakeTaskMutation } from '../tasksApiSlice'
import { useUpdateSubtaskMutation } from '../modules/subtasks/subtasksApiSlice'
import { useCreateCommentMutation } from '../modules/comments/commentsApiSlice'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { useAppSelector } from '@/hooks/redux'
import { getIsManager } from '@/features/user/userSlice'
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
import DetailNotFound from './TaskDetailEmpty'

export const TaskDetailPage = () => {
	const { id } = useParams<{ id: string }>()
	const { data, isLoading } = useGetTaskByIdQuery(id!)
	const [updateTask] = useUpdateTaskMutation()
	const [takeTask] = useTakeTaskMutation()
	const [updateSubtask] = useUpdateSubtaskMutation()
	const [createComment] = useCreateCommentMutation()
	const [editOpen, setEditOpen] = useState(false)
	const isManager = useAppSelector(getIsManager)

	if (isLoading) return <BoxFallback />
	if (!data?.data) return <DetailNotFound />

	const task = data.data
	// Неактивные (замороженные) статусы: решения, закрытые и отменённые заявки
	// недоступны для изменения данных.
	const isInactive = task.status === 'resolved' || task.status === 'closed' || task.status === 'cancelled'
	const canEdit = !isInactive && task.access?.canWrite
	const canUploadAttachments = !isInactive && task.access?.canWork

	const handleStatusChange = async (taskId: string, status: TicketStatus, comment?: string) => {
		try {
			await updateTask({ id: taskId, status })
			if (comment) {
				await createComment({ ticketId: taskId, text: comment, isInternal: false, type: 'status_change' })
			}
		} catch {
			// handled by toast in apiSlice
		}
	}

	const handleTake = async () => {
		try {
			await takeTask(task.id)
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

						<InfoBar task={task} onStatusChange={handleStatusChange} onTake={handleTake} />
					</Box>

					<Grid container spacing={3} sx={{ mt: 2 }}>
						<Grid size={{ xs: 12, lg: 8 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Description
								description={task.description}
								onEdit={canEdit ? () => setEditOpen(true) : undefined}
							/>
							<Subtasks
								subtasks={task.subtasks}
								taskId={task.id}
								canWork={task.access?.canWork}
								onSubtaskStatusChange={handleSubtaskStatusChange}
							/>
							<Attachments
								attachments={task.attachments}
								canWork={canUploadAttachments}
								taskId={task.id}
							/>
							<Comments taskId={task.id} isInactive={isInactive} />
						</Grid>

						<Grid size={{ xs: 12, lg: 4 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Participants task={task} />
							<Meta task={task} />
							{isManager && <Notifications taskId={task.id} />}
						</Grid>
					</Grid>
				</Box>
			</Box>

			<TaskEditModal open={editOpen} onClose={() => setEditOpen(false)} task={task} />
		</>
	)
}
