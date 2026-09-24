import { Box, Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown, Check, Tag, Building2, CheckCircle2, RotateCcw, XCircle, Hand } from 'lucide-mui'
import { useState } from 'react'

import type { ITask, TicketStatus } from '../../types/task'
import { STATUS_MAP } from '../../constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { getUserId, getCurrentCapabilities } from '@/features/user/userSlice'
import { useUpdateTaskMutation, useTakeTaskMutation } from '../../tasksApiSlice'
import { useCreateCommentMutation } from '../../modules/comments/commentsApiSlice'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { StatusChangeDialog } from './StatusChangeDialog'
import { DeadlinePopover } from './DeadlinePopover'

const ACTIVE_STATUSES: TicketStatus[] = ['open', 'in_progress', 'pending', 'on_hold']
const COMMENT_REQUIRED_STATUSES: TicketStatus[] = ['on_hold', 'pending']

interface Props {
	task: ITask
}

export const InfoBar = ({ task }: Props) => {
	const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
	const [pendingStatus, setPendingStatus] = useState<TicketStatus | null>(null)
	const [updateTask] = useUpdateTaskMutation()
	const [takeTask] = useTakeTaskMutation()
	const [createComment] = useCreateCommentMutation()
	const currentUserId = useAppSelector(getUserId)
	const capabilities = useAppSelector(getCurrentCapabilities)
	const isOwner = currentUserId != null && currentUserId === task.owner?.id
	// Меню «Изменить статус» доступно только автору, исполнителю, менеджеру группы
	// или админу реалма. Права Casbin write/canWork сами по себе его не открывают.
	const isCreator = currentUserId != null && currentUserId === task.creator?.id
	const isAssignee = currentUserId != null && currentUserId === task.assignee?.id
	const isManager = Boolean(task.access?.isManager) || capabilities.isRealmAdmin
	const isStatusManager = isCreator || isAssignee || isManager
	// Закрытая/отменённая заявка терминальна — меню «Изменить статус» не показываем.
	const isTerminal = task.status === 'closed' || task.status === 'cancelled'
	const isActive = ACTIVE_STATUSES.includes(task.status)
	const canUseMenu = !isTerminal && isStatusManager
	const canTake = Boolean(task.access?.canTake)
	const allowedStatuses = task.access?.allowedStatuses
	const canAccept = isOwner && task.status === 'resolved'
	const canReturn = isOwner && task.status === 'resolved'
	const canCancel = isOwner && ACTIVE_STATUSES.includes(task.status)

	const handleStatusChange = async (status: TicketStatus, comment?: string) => {
		try {
			await updateTask({ id: task.id, status })
			if (comment) {
				await createComment({ ticketId: task.id, text: comment, isInternal: false, type: 'status_change' })
			}
		} catch {
			// handled by toast in apiSlice
		}
	}

	const changeStatus = (status: TicketStatus) => {
		setAnchorEl(null)
		// Комментарий при переходе в «В работе» требуется только при возврате из «Решена»,
		// во всех остальных случаях — нет. Для «Ожидание»/«Отложена» — всегда.
		const requiresComment =
			status === 'in_progress' ? task.status === 'resolved' : COMMENT_REQUIRED_STATUSES.includes(status)
		if (requiresComment) {
			setPendingStatus(status)
		} else {
			handleStatusChange(status)
		}
	}

	const handleDialogSubmit = (comment: string) => {
		if (pendingStatus) {
			handleStatusChange(pendingStatus, comment)
			setPendingStatus(null)
		}
	}

	const handleTake = async () => {
		try {
			await takeTask(task.id)
		} catch {
			// handled by toast in apiSlice
		}
	}

	const handleSetDueDate = async (iso: string) => {
		try {
			await updateTask({ id: task.id, dueDate: iso })
		} catch {
			// handled by toast in apiSlice
		}
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
				{(task.access?.isManager || capabilities.isRealmAdmin) && isActive && !task.dueDate && (
					<DeadlinePopover onSetDueDate={handleSetDueDate} />
				)}

				{canCancel && (
					<Button
						variant='outlined'
						color='error'
						onClick={() => changeStatus('cancelled')}
						startIcon={<XCircle sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Отменить заявку
					</Button>
				)}

				{canAccept && (
					<Button
						variant='contained'
						onClick={() => changeStatus('closed')}
						startIcon={<CheckCircle2 sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Подтвердить решение
					</Button>
				)}

				{canReturn && (
					<Button
						variant='outlined'
						onClick={() => changeStatus('in_progress')}
						startIcon={<RotateCcw sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Вернуть в работу
					</Button>
				)}

				{canTake && (
					<Button
						variant='outlined'
						onClick={handleTake}
						startIcon={<Hand sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Взять в работу
					</Button>
				)}

				{!canTake && canUseMenu && (
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
				)}

				{!canTake && canUseMenu && (
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
							([value, info]) => {
								const isAllowed = !allowedStatuses || allowedStatuses.includes(value)
								const isCurrent = value === task.status
								return (
									<MenuItem
										key={value}
										disabled={!isAllowed && !isCurrent}
										selected={isCurrent}
										onClick={() => isAllowed && changeStatus(value)}
										sx={{ fontSize: '0.875rem', gap: 1.5 }}
									>
										<info.icon sx={{ fontSize: 16, color: info.textColor }} />
										{info.label}
										{isCurrent && (
											<Box sx={{ ml: 'auto', color: 'primary.main', display: 'flex' }}>
												<Check sx={{ fontSize: 14 }} />
											</Box>
										)}
									</MenuItem>
								)
							},
						)}
					</Menu>
				)}
			</Box>

			{pendingStatus && (
				<StatusChangeDialog
					open
					statusLabel={STATUS_MAP[pendingStatus].label}
					onSubmit={handleDialogSubmit}
					onCancel={() => setPendingStatus(null)}
				/>
			)}
		</Box>
	)
}
