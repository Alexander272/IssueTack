import { Box, Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown, Check, CheckCircle2, RotateCcw, XCircle, Hand, CalendarClock } from 'lucide-mui'
import { useMemo, useState } from 'react'

import type { ITask, TicketStatus } from '../../types/task'
import { STATUS_MAP } from '../../constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { getUserId, getCurrentCapabilities, getHasNoGroups } from '@/features/user/userSlice'
import { useUpdateTaskMutation, useTakeTaskMutation, useGetDueDateRequestsQuery } from '../../tasksApiSlice'
import { useCreateCommentMutation } from '../../modules/comments/commentsApiSlice'
import { TaskStatusBadge } from '../TaskStatusBadge'
import { TaskPriorityBadge } from '../TaskPriorityBadge'
import { StatusChangeDialog } from './StatusChangeDialog'
import { DeadlinePopover } from './DeadlinePopover'
import { ModifierPopover } from './ModifierPopover'
import { DueDateRequestDialog } from './DueDateRequestDialog'
import { DueDateRequestCard } from './DueDateRequestCard'

const ACTIVE_STATUSES: TicketStatus[] = ['open', 'in_progress', 'pending', 'on_hold']
const COMMENT_REQUIRED_STATUSES: TicketStatus[] = ['on_hold', 'pending']
const TERMINAL_STATUSES: TicketStatus[] = ['closed', 'cancelled']

interface Props {
	task: ITask
	/** Смена подкатегории: общий обработчик поднят в TaskDetailPage, чтобы
	 * кнопка здесь и меню «Ещё» в шапке отправляли одно и то же обновление. */
	onSetModifier: (id: string | null) => void
}

export const InfoBar = ({ task, onSetModifier }: Props) => {
	const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
	const [pendingStatus, setPendingStatus] = useState<TicketStatus | null>(null)
	const [updateTask] = useUpdateTaskMutation()
	const [takeTask] = useTakeTaskMutation()
	const [createComment] = useCreateCommentMutation()
	const { data: dueDateRequestsData } = useGetDueDateRequestsQuery(task.id, {
		skip: !task.access,
	})
	const dueDateRequest = dueDateRequestsData?.data ?? task.dueDateRequest ?? null
	const currentUserId = useAppSelector(getUserId)
	const capabilities = useAppSelector(getCurrentCapabilities)
	const hasNoGroups = useAppSelector(getHasNoGroups)
	const isAssignee = currentUserId != null && currentUserId === task.assignee?.id
	const isManager = Boolean(task.access?.isManager) || capabilities.isRealmAdmin
	// Меню «Изменить статус» — только «рабочим» и менеджерам: исполнителю (заявку
	// назначили явно, он ведёт её по статусам) и менеджеру группы/админу реалма.
	// Автор и владелец меню не видят, даже если автор состоит в группе: они ведут
	// заявку решениями владельца (Отменить/Подтвердить/Вернуть).
	const isStatusManager = isAssignee || isManager
	// Закрытая/отменённая заявка терминальна — меню «Изменить статус» не показываем.
	const isTerminal = task.status === 'closed' || task.status === 'cancelled'
	const isActive = ACTIVE_STATUSES.includes(task.status)
	const canUseMenu = !isTerminal && isStatusManager
	const canTake = Boolean(task.access?.canTake)
	const allowedStatuses = task.access?.allowedStatuses
	const canRequestDueDate = Boolean(task.access?.canRequestDueDate)
	const canDecideDueDate = Boolean(task.access?.canDecideDueDate)
	// «Терминальные» статусы (Закрыта/Отменена) — решения менеджера. Чистому
	// исполнителю их не показываем (бэкенд всё равно не даст — canChangeStatus /
	// isCreatorOrManager), чтобы в меню не висели серые недоступные пункты.
	const canSeeTerminalStatuses = isManager
	// Решения владельца приходят готовыми флагами с бэкенда (AccessFlags): той же
	// формулой считается и плагин Mattermost, поэтому веб не дублирует предикат
	// «владелец + статус» и не может с ним разъехаться.
	const canCancel = Boolean(task.access?.canCancel)
	const canConfirm = Boolean(task.access?.canConfirm)
	const canReopen = Boolean(task.access?.canReopen)

	const handleStatusChange = async (status: TicketStatus, comment?: string, dueDate?: string | null) => {
		try {
			const updateData: { status: TicketStatus; dueDate?: string | null } = { status }
			if (dueDate !== undefined) {
				updateData.dueDate = dueDate
			}
			await updateTask({ id: task.id, ...updateData })
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

	const handleDialogSubmit = (comment: string, dueDate?: string | null) => {
		if (pendingStatus) {
			handleStatusChange(pendingStatus, comment, dueDate)
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

	const [dueDateRequestDialogOpen, setDueDateRequestDialogOpen] = useState(false)

	const statusOptions = useMemo(() => {
		return (Object.keys(STATUS_MAP) as TicketStatus[])
			.filter(status => {
				if (canSeeTerminalStatuses || status === task.status) return true
				return !TERMINAL_STATUSES.includes(status)
			})
			.map(status => {
				const isCurrent = status === task.status
				const isAllowed = !allowedStatuses || allowedStatuses.includes(status)

				return {
					value: status,
					info: STATUS_MAP[status],
					isCurrent,
					isAllowed,
					isDisabled: !isAllowed && !isCurrent,
				}
			})
	}, [canSeeTerminalStatuses, task.status, allowedStatuses])

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
				{/* Дубль на детали обозначается баннером сверху; бейдж «дубль #N» остаётся только в списках. */}
				<TaskStatusBadge status={task.status} sx={{ height: 33 }} />
				{!hasNoGroups && <TaskPriorityBadge priority={task.priority} />}
			</Box>

			<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5 }}>
				{isAssignee && canRequestDueDate && isActive && (
					<Button
						variant='outlined'
						onClick={() => setDueDateRequestDialogOpen(true)}
						startIcon={<CalendarClock sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Запросить перенос
					</Button>
				)}

				{!hasNoGroups && isManager && isActive && !task.dueDate && (
					<DeadlinePopover onSetDueDate={handleSetDueDate} />
				)}

				{/* Пока подкатегории нет — заметная кнопка в строке действий. Когда она
				    выбрана, имя уже показано в Meta, а менять её можно из меню «Ещё» в шапке. */}
				{task.access?.canPickModifier && !task.categoryModifier && (
					<ModifierPopover task={task} onSelect={onSetModifier} />
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

				{canConfirm && (
					<Button
						variant='contained'
						onClick={() => changeStatus('closed')}
						startIcon={<CheckCircle2 sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Подтвердить решение
					</Button>
				)}

				{canReopen && (
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
						{statusOptions.map(({ value, info, isCurrent, isAllowed, isDisabled }) => {
							const Icon = info.icon

							return (
								<MenuItem
									key={value}
									disabled={isDisabled}
									selected={isCurrent}
									onClick={() => isAllowed && changeStatus(value)}
									sx={{ fontSize: '0.875rem', gap: 1.5 }}
								>
									<Icon sx={{ fontSize: 16, color: info.textColor }} />
									{info.label}
									{isCurrent && <Check sx={{ fontSize: 14, ml: 'auto', color: 'primary.main' }} />}
								</MenuItem>
							)
						})}
					</Menu>
				)}
			</Box>

			{canDecideDueDate && dueDateRequest?.status === 'pending' && (
				<DueDateRequestCard task={task} request={dueDateRequest} />
			)}

			{pendingStatus && (
				<StatusChangeDialog
					open
					statusLabel={STATUS_MAP[pendingStatus].label}
					showDueDateField={pendingStatus === 'in_progress' && task.status === 'resolved' && isManager}
					currentDueDate={task.dueDate ?? null}
					onSubmit={(text, dueDate) => handleDialogSubmit(text, dueDate)}
					onCancel={() => setPendingStatus(null)}
				/>
			)}

			<DueDateRequestDialog
				open={dueDateRequestDialogOpen}
				task={task}
				onClose={() => setDueDateRequestDialogOpen(false)}
			/>
		</Box>
	)
}
