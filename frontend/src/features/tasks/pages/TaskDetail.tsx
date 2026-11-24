import { useState } from 'react'
import { Box, Grid } from '@mui/material'
import { useParams } from 'react-router'

import type { ITask } from '../types/task'
import { useGetTaskByIdQuery, useUpdateTaskMutation } from '../tasksApiSlice'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { useAppSelector } from '@/hooks/redux'
import { getIsManager, getUserId } from '@/features/user/userSlice'
import { useTicketControls } from '../hooks/useTicketControls'
import { Header } from '../components/Detail/Header'
import { InfoBar } from '../components/Detail/InfoBar'
import { Description } from '../components/Detail/Description'
import { Subtasks } from '../components/Detail/Subtasks'
import { Attachments } from '../components/Detail/Attachments'
import { Comments } from '../components/Detail/Comments'
import { Participants } from '../components/Detail/Participants'
import { Meta } from '../components/Detail/Meta'
import { Notifications } from '../components/Detail/Notifications'
import { Duplicates } from '../components/Detail/Duplicates'
import { DuplicateBanner } from '../components/Detail/DuplicateBanner'
import { TaskEditModal } from '../components/TaskEditModal'
import DetailNotFound from './TaskDetailEmpty'

interface DetailContentProps {
	task: ITask
	controls: ReturnType<typeof useTicketControls>
	isManager: boolean
	userId?: string
}

/**
 * Всё состояние детали живёт здесь и монтируется с key={task.id}. Роутер при
 * переходе /tasks/:id → /tasks/:id2 переиспользует компонент маршрута, поэтому
 * без key открытые модалки/поповеры и черновик комментария «переехали» бы на
 * другую заявку (черновик — ещё и отправился бы с её id).
 */
const DetailContent = ({ task, controls, isManager, userId }: DetailContentProps) => {
	const [editOpen, setEditOpen] = useState(false)
	const [updateTask] = useUpdateTaskMutation()

	// Один обработчик на оба входа: кнопку в InfoBar (когда подкатегории нет)
	// и пункт меню «Ещё» (когда она уже выбрана). В Update уходит только id
	// и categoryModifierId, остальные поля не задеваются.
	const handleSetModifier = async (categoryModifierId: string | null) => {
		try {
			await updateTask({ id: task.id, categoryModifierId })
		} catch {
			// handled by toast in apiSlice
		}
	}

	return (
		<>
			<Box sx={{ flexGrow: 1, overflow: 'auto', bgcolor: '#f9fafb' }}>
				<Box sx={{ maxWidth: 'xl', mx: 'auto', p: { xs: 2, sm: 3 } }}>
					<Box
						sx={{
							gap: 2,
							p: 2.5,
							bgcolor: 'white',
							borderRadius: '12px',
							border: '1px solid #e5e7eb',
							mb: 2,
						}}
					>
						<Header
							task={task}
							onEdit={controls.canEdit ? () => setEditOpen(true) : undefined}
							onSetModifier={handleSetModifier}
						/>

						<InfoBar task={task} onSetModifier={handleSetModifier} />
					</Box>

					{task.duplicateOf && <DuplicateBanner task={task} />}

					<Grid container spacing={2} sx={{ mt: 2 }}>
						<Grid size={{ xs: 12, lg: 9 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Description
								description={task.description}
								onEdit={controls.canEdit ? () => setEditOpen(true) : undefined}
							/>
							<Subtasks
								subtasks={task.subtasks}
								taskId={task.id}
								canWork={controls.canWork}
								canCreateSubtask={controls.canCreateSubtask}
								canDeleteSubtask={controls.canDeleteSubtask}
								inactive={controls.isInactive}
								userId={userId}
								canManage={controls.canManage}
							/>
							<Attachments
								attachments={task.attachments}
								canAttach={controls.canAttachToTicket}
								taskId={task.id}
							/>
							<Comments taskId={task.id} isInactive={controls.isInactive} canWork={controls.canWork} />
						</Grid>

						<Grid size={{ xs: 12, lg: 3 }} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Participants task={task} />
							<Duplicates task={task} />
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

export const TaskDetailPage = () => {
	const { id } = useParams<{ id: string }>()
	const { data, isLoading } = useGetTaskByIdQuery(id!)

	const isManager = useAppSelector(getIsManager)
	const userId = useAppSelector(getUserId)

	// Все права детали считаются одной формулой из access-флагов бэкенда; хуки
	// вызываются до ранних возвратов, чтобы порядок хуков не менялся.
	const controls = useTicketControls(data?.data)

	if (isLoading) return <BoxFallback />
	if (!data?.data) return <DetailNotFound />

	return (
		<DetailContent
			key={data.data.id}
			task={data.data}
			controls={controls}
			isManager={isManager}
			userId={userId ?? undefined}
		/>
	)
}
