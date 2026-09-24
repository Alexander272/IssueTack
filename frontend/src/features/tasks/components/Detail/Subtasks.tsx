import { useState } from 'react'
import { Button, Box, FormControl, IconButton, MenuItem, Select, Typography, Tooltip } from '@mui/material'
import type { SvgIconProps } from '@mui/material'
import type { FC } from 'react'
import { CheckCircle, Clock, Circle, ListCheck, Plus, Pencil, Trash2, ChevronDown, ChevronUp } from 'lucide-mui'

import type { ISubtask, TicketStatus } from '../../types/task'
import {
	useCreateSubtaskMutation,
	useUpdateSubtaskMutation,
	useDeleteSubtaskMutation,
} from '../../modules/subtasks/subtasksApiSlice'
import { STATUS_MAP, DUE_DATE_COLORS, DUE_DATE_ICONS, DUE_DATE_DEFAULT_ICON } from '../../constants/taskMaps'
import { getDueDateTone, getShortDate } from '@/utils/date'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { SubtaskDialog } from './SubtaskDialog'

interface Props {
	subtasks: ISubtask[] | undefined
	taskId: string
	canWork?: boolean
	/** Может ли пользователь создавать подзадачи (создатель/исполнитель тикета, менеджер группы, админ реалма). */
	canCreate?: boolean
	canDelete?: boolean
	inactive?: boolean
	/** Текущий пользователь. Нужен для правки содержимого: она доступна автору (createdBy) или canManage. */
	userId?: string
	/** Признак «управления» тикетом: менеджер группы или админ реалма. */
	canManage?: boolean
}

const SUBTASK_STATUS_OPTIONS: { value: TicketStatus; label: string; icon: FC<SvgIconProps>; iconColor: string }[] = (
	[
		{ value: 'open', label: 'Открыта' },
		{ value: 'in_progress', label: 'В работе' },
		{ value: 'closed', label: 'Выполнена' },
	] as const
).map(opt => ({ ...opt, icon: STATUS_MAP[opt.value].icon, iconColor: STATUS_MAP[opt.value].textColor }))

const DONE_STATUSES: TicketStatus[] = ['closed', 'resolved']

const DESCRIPTION_CLAMP_THRESHOLD = 110
const isLongDescription = (text: string) => text.length > DESCRIPTION_CLAMP_THRESHOLD

export const Subtasks = ({
	subtasks,
	taskId,
	canWork = true,
	canCreate = false,
	canDelete = false,
	inactive = false,
	userId,
	canManage = false,
}: Props) => {
	const [createSubtask] = useCreateSubtaskMutation()
	const [updateSubtask] = useUpdateSubtaskMutation()
	const [deleteSubtask] = useDeleteSubtaskMutation()
	const [dialogOpen, setDialogOpen] = useState(false)
	const [editSubtask, setEditSubtask] = useState<ISubtask | null>(null)
	const [deleteTarget, setDeleteTarget] = useState<ISubtask | null>(null)
	const [deleting, setDeleting] = useState(false)
	const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

	const list = subtasks ?? []
	const interactive = canWork && !inactive
	const canCreateSub = canCreate && interactive
	const canDeleteSub = canDelete && interactive
	const done = list.filter(s => DONE_STATUSES.includes(s.status)).length
	const total = list.length
	const progress = total > 0 ? Math.round((done / total) * 100) : 0

	if (total === 0 && !interactive) return null

	const handleStatusChange = async (subtaskId: string, status: TicketStatus) => {
		try {
			await updateSubtask({ ticketId: taskId, id: subtaskId, status })
		} catch {
			// handled by toast in apiSlice
		}
	}

	const handleDialogSubmit = async ({ title, description }: { title: string; description: string }) => {
		try {
			if (editSubtask) {
				await updateSubtask({ ticketId: taskId, id: editSubtask.id, title, description })
			} else {
				const nextOrder = list.reduce((max, s) => Math.max(max, s.sortOrder), -1) + 1
				await createSubtask({
					ticketId: taskId,
					title,
					description,
					status: 'open',
					priority: 'medium',
					sortOrder: nextOrder,
				})
			}
		} catch {
			// handled by toast in apiSlice
		}
	}

	const openCreate = () => {
		setEditSubtask(null)
		setDialogOpen(true)
	}

	const openEdit = (sub: ISubtask) => {
		setEditSubtask(sub)
		setDialogOpen(true)
	}

	const handleDelete = async () => {
		if (!deleteTarget) return
		setDeleting(true)
		try {
			await deleteSubtask({ ticketId: taskId, subtaskId: deleteTarget.id })
			setDeleteTarget(null)
		} catch {
			// handled by toast in apiSlice
		} finally {
			setDeleting(false)
		}
	}

	const toggleExpanded = (subtaskId: string) => {
		setExpandedIds(prev => {
			const next = new Set(prev)
			if (next.has(subtaskId)) {
				next.delete(subtaskId)
			} else {
				next.add(subtaskId)
			}
			return next
		})
	}

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					px: 2.5,
					py: 2,
					borderBottom: '1px solid #e5e7eb',
				}}
			>
				<Typography
					sx={{
						fontWeight: 700,
						color: '#1f2937',
						fontSize: '0.9375rem',
						display: 'flex',
						alignItems: 'center',
						gap: 1,
					}}
				>
					<ListCheck sx={{ fontSize: 16 }} />
					Подзадачи
					<Typography
						component='span'
						sx={{
							fontSize: '0.75rem',
							bgcolor: '#e5e7eb',
							color: '#374151',
							px: 1,
							py: 0.25,
							borderRadius: '999px',
						}}
					>
						{done}/{total}
					</Typography>
				</Typography>

				{canCreateSub && (
					<Box
						onClick={openCreate}
						sx={{
							display: 'inline-flex',
							alignItems: 'center',
							gap: 0.5,
							cursor: 'pointer',
							color: 'primary.main',
							fontSize: '0.8125rem',
							fontWeight: 600,
							px: 1,
							py: 0.5,
							borderRadius: '8px',
							'&:hover': { bgcolor: 'rgba(25,118,210,0.08)' },
						}}
					>
						<Plus sx={{ fontSize: 18 }} />
						Добавить
					</Box>
				)}
			</Box>

			{total === 0 ? (
				<Box sx={{ px: 2.5, py: 3, textAlign: 'center' }}>
					<Typography sx={{ fontSize: '0.8125rem', color: '#9ca3af' }}>
						{interactive ? 'Нет подзадач — добавьте первую' : 'Нет подзадач'}
					</Typography>
				</Box>
			) : (
				<Box>
					{list.map(sub => {
						const isDone = DONE_STATUSES.includes(sub.status)
						const tone = getDueDateTone(sub.dueDate)
						const DueDateIcon = tone ? DUE_DATE_ICONS[tone] : DUE_DATE_DEFAULT_ICON
						return (
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
								{isDone ? (
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
											color: isDone ? '#9ca3af' : '#1f2937',
											...(isDone ? { textDecoration: 'line-through' } : {}),
										}}
									>
										{sub.title}
									</Typography>
									{sub.description && (
										<Box>
											<Typography
												sx={{
													fontSize: '0.75rem',
													color: '#6b7280',
													whiteSpace: 'pre-wrap',
													...(isLongDescription(sub.description) && !expandedIds.has(sub.id)
														? {
																overflow: 'hidden',
																textOverflow: 'ellipsis',
																display: '-webkit-box',
																WebkitLineClamp: 2,
																WebkitBoxOrient: 'vertical',
															}
														: {}),
												}}
											>
												{sub.description}
											</Typography>
											{isLongDescription(sub.description) && (
												<Button
													onClick={() => toggleExpanded(sub.id)}
													size='small'
													sx={{
														textTransform: 'none',
														fontSize: '0.7rem',
														color: '#9ca3af',
														px: 0,
														py: 0,
														minHeight: 0,
														gap: 0.25,
														'&:hover': { color: 'primary.main', bgcolor: 'transparent' },
													}}
												>
													{expandedIds.has(sub.id) ? (
														<>
															<ChevronUp sx={{ fontSize: 13 }} />
															Скрыть
														</>
													) : (
														<>
															<ChevronDown sx={{ fontSize: 13 }} />
															Показать ещё
														</>
													)}
												</Button>
											)}
										</Box>
									)}
									<Box
										sx={{
											display: 'flex',
											flexWrap: 'wrap',
											alignItems: 'center',
											gap: 1,
											mt: 0.5,
										}}
									>
										{sub.assignee && (
											<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
												{sub.assignee.lastName} {sub.assignee.firstName}
											</Typography>
										)}
										{!isDone && sub.dueDate && (
											<Box
												sx={{
													display: 'inline-flex',
													alignItems: 'center',
													gap: 0.5,
													fontSize: '0.75rem',
													fontWeight: 500,
													color: tone ? DUE_DATE_COLORS[tone] : '#374151',
												}}
											>
												<DueDateIcon
													sx={{
														fontSize: 13,
														color: tone ? DUE_DATE_COLORS[tone] : '#6b7280',
													}}
												/>
												{getShortDate(sub.dueDate)}
											</Box>
										)}
										{isDone && sub.closedAt && (
											<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
												Выполнена · {getShortDate(sub.closedAt)}
											</Typography>
										)}
									</Box>
								</Box>

								{interactive && (
									<Box sx={{ display: 'flex', alignItems: 'center' }}>
										{(sub.createdBy === userId || canManage) && (
											<Tooltip title='Изменить'>
												<IconButton size='large' onClick={() => openEdit(sub)}>
													<Pencil sx={{ fontSize: 16, color: '#6b7280' }} />
												</IconButton>
											</Tooltip>
										)}
										{canDeleteSub && (
											<Tooltip title='Удалить'>
												<IconButton size='large' onClick={() => setDeleteTarget(sub)}>
													<Trash2 sx={{ fontSize: 16, color: '#ef5350' }} />
												</IconButton>
											</Tooltip>
										)}
									</Box>
								)}

								<FormControl>
									<Select
										value={sub.status}
										disabled={!interactive}
										onChange={e => handleStatusChange(sub.id, e.target.value as TicketStatus)}
										renderValue={value => {
											const opt = SUBTASK_STATUS_OPTIONS.find(o => o.value === value)
											if (!opt) return String(value)
											const Icon = opt.icon
											return (
												<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
													<Icon sx={{ fontSize: 14, color: opt.iconColor }} />
													{opt.label}
												</Box>
											)
										}}
										sx={{
											fontSize: '0.85rem',
											transition: '0.15s background-color ease-in-out',
											'&:hover': { backgroundColor: '#f5f5f5' },
											'& .MuiOutlinedInput-notchedOutline': {
												border: 'none',
											},
											// '& .MuiOutlinedInput-notchedOutline': { borderColor: '#d1d5db' },
										}}
									>
										{SUBTASK_STATUS_OPTIONS.map(opt => (
											<MenuItem key={opt.value} value={opt.value} sx={{ gap: 1.25 }}>
												<opt.icon sx={{ fontSize: 15, color: opt.iconColor }} />
												{opt.label}
											</MenuItem>
										))}
									</Select>
								</FormControl>
							</Box>
						)
					})}
				</Box>
			)}

			{total > 0 && (
				<Box sx={{ px: 2.5, py: 2, bgcolor: '#f9fafb', borderTop: '1px solid #e5e7eb' }}>
					<Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.75 }}>
						<Typography sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 500 }}>
							Общий прогресс
						</Typography>
						<Typography sx={{ fontSize: '0.75rem', color: '#374151', fontWeight: 600 }}>
							{progress}%
						</Typography>
					</Box>
					<Box
						sx={{ width: '100%', height: 8, bgcolor: '#e5e7eb', borderRadius: '999px', overflow: 'hidden' }}
					>
						<Box
							sx={{
								width: `${progress}%`,
								height: '100%',
								bgcolor: 'primary.main',
								borderRadius: '999px',
								transition: 'width 0.3s',
							}}
						/>
					</Box>
				</Box>
			)}

			<SubtaskDialog
				key={dialogOpen ? (editSubtask?.id ?? 'create') : 'idle'}
				open={dialogOpen}
				editSubtask={editSubtask}
				onSubmit={handleDialogSubmit}
				onClose={() => setDialogOpen(false)}
			/>
			<ConfirmDialog
				open={Boolean(deleteTarget)}
				title='Удалить подзадачу?'
				message={deleteTarget ? `«${deleteTarget.title}» будет удалена без возможности восстановления.` : ''}
				confirmLabel='Удалить'
				confirmColor='error'
				loading={deleting}
				onConfirm={handleDelete}
				onCancel={() => setDeleteTarget(null)}
			/>
		</Box>
	)
}
