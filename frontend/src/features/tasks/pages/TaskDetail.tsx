import { useState } from 'react'
import { Box, Grid } from '@mui/material'
import { useParams } from 'react-router'

import { useGetTaskByIdQuery } from '../tasksApiSlice'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { useAppSelector } from '@/hooks/redux'
import { getCurrentCapabilities, getIsManager, getUserId } from '@/features/user/userSlice'
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
import { TransferModal } from '../components/Detail/TransferModal'
import DetailNotFound from './TaskDetailEmpty'

export const TaskDetailPage = () => {
	const { id } = useParams<{ id: string }>()
	const { data, isLoading } = useGetTaskByIdQuery(id!)
	const [editOpen, setEditOpen] = useState(false)
	const [transferOpen, setTransferOpen] = useState(false)
	const isManager = useAppSelector(getIsManager)
	const capabilities = useAppSelector(getCurrentCapabilities)
	const userId = useAppSelector(getUserId)

	if (isLoading) return <BoxFallback />
	if (!data?.data) return <DetailNotFound />

	const task = data.data
	// Неактивные (замороженные) статусы: решения, закрытые и отменённые заявки
	// недоступны для изменения данных.
	const isInactive = task.status === 'resolved' || task.status === 'closed' || task.status === 'cancelled'
	const canEdit = !isInactive && task.access?.canEditFields
	const canUploadAttachments = !isInactive && task.access?.canWork
	const canCreate = Boolean(
		task.creator?.id === userId ||
		task.assignee?.id === userId ||
		task.access?.isManager ||
		capabilities.isRealmAdmin,
	)

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
						<Header
							task={task}
							onEdit={canEdit ? () => setEditOpen(true) : undefined}
							onTransfer={() => setTransferOpen(true)}
						/>

						<InfoBar task={task} />
					</Box>

					<Grid container spacing={2} sx={{ mt: 2 }}>
						<Grid size={{ xs: 12, lg: 9 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Description
								description={task.description}
								onEdit={canEdit ? () => setEditOpen(true) : undefined}
							/>
							<Subtasks
								subtasks={task.subtasks}
								taskId={task.id}
								canWork={task.access?.canWork}
								canCreate={canCreate}
								canDelete={
									Boolean(task.access?.canDelete) && Boolean(task.access?.canWork) && !isInactive
								}
								userId={userId ?? undefined}
								canManage={Boolean(task.access?.isManager) || capabilities.isRealmAdmin}
							/>
							<Attachments
								attachments={task.attachments}
								canWork={canUploadAttachments}
								taskId={task.id}
							/>
							<Comments taskId={task.id} isInactive={isInactive} />
						</Grid>

						<Grid size={{ xs: 12, lg: 3 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Participants task={task} />
							<Meta task={task} />
							{isManager && <Notifications taskId={task.id} />}
						</Grid>
					</Grid>
				</Box>
			</Box>

			<TaskEditModal open={editOpen} onClose={() => setEditOpen(false)} task={task} />
			<TransferModal open={transferOpen} onClose={() => setTransferOpen(false)} task={task} />
		</>
	)
}
