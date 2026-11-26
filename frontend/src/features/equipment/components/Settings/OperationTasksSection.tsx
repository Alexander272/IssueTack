import { useState, type FC } from 'react'
import { Box, Button, Chip, IconButton, Tooltip, Typography } from '@mui/material'
import {
	InfoIcon,
	PencilIcon,
	PlayIcon,
	PlusIcon,
	PauseIcon,
	Trash2Icon,
	TruckIcon,
	Undo2Icon,
} from 'lucide-mui'

import type {
	EquipmentOperation,
	IEquipmentOperationTask,
} from '../../types/equipment'
import type { Priority } from '@/features/tasks/types/task'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import {
	useGetOperationTasksQuery,
	useUpdateOperationTaskMutation,
	useDeleteOperationTaskMutation,
} from '../../equipmentApiSlice'
import { OperationTaskDialog } from '../Dialogs/OperationTaskDialog'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { useAppSelector } from '@/hooks/redux'
import { getRealm } from '@/features/realms/realmSlice'
import { pluralize } from '@/utils/plural'

const OPERATIONS: { value: EquipmentOperation; label: string; Icon: typeof TruckIcon }[] = [
	{ value: 'assign', label: 'Выдача', Icon: TruckIcon },
	{ value: 'unassign', label: 'Возврат', Icon: Undo2Icon },
]

// Секция настроек «Автоматические задачи»: правила, которые создают заявки после
// выдачи и возврата оборудования. Площадка заявки всегда берётся из единицы.
export const OperationTasksSection: FC = () => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const realm = useAppSelector(getRealm)

	const { data } = useGetOperationTasksQuery(
		{ realmId: realm?.id ?? '' },
		{ skip: !realm?.id },
	)
	const [updateTask] = useUpdateOperationTaskMutation()
	const [remove] = useDeleteOperationTaskMutation()

	const [dialogOpen, setDialogOpen] = useState(false)
	const [editTask, setEditTask] = useState<IEquipmentOperationTask | null>(null)
	const [deleteTarget, setDeleteTarget] = useState<IEquipmentOperationTask | null>(null)

	const tasks = data?.data ?? []

	const toggle = async (task: IEquipmentOperationTask) => {
		try {
			await updateTask({
				id: task.id,
				realmId: task.realmId,
				name: task.name,
				operation: task.operation,
				action: task.action,
				title: task.title,
				description: task.description,
				categoryId: task.categoryId,
				groupId: task.groupId,
				priority: task.priority as Priority,
				enabled: !task.enabled,
			}).unwrap()
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	const handleDelete = async () => {
		if (!deleteTarget?.id) return
		try {
			await remove(deleteTarget.id).unwrap()
			setDeleteTarget(null)
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	return (
		<>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					gap: 2,
					mb: 2,
					flexWrap: 'wrap',
				}}
			>
				<Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, maxWidth: 640 }}>
					<InfoIcon sx={{ fontSize: 16, color: '#6b7280', mt: 0.2 }} />
					<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
						Пока правил нет, после выдачи и возврата создаётся заявка по общим настройкам.
						Как только появится хотя бы одно правило, используются только правила: можно
						отключить все, чтобы заявки не создавались вообще.
					</Typography>
				</Box>
				{canWrite && (
					<Button
						variant='outlined'
						onClick={() => {
							setEditTask(null)
							setDialogOpen(true)
						}}
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						sx={{
							textTransform: 'none',
							fontWeight: 500,
							borderRadius: '8px',
							bgcolor: '#fff',
							gap: 1,
						}}
					>
						Добавить правило
					</Button>
				)}
			</Box>

			{tasks.length === 0 ? (
				<Box
					sx={{
						display: 'flex',
						flexDirection: 'column',
						alignItems: 'center',
						gap: 1,
						py: 4,
						px: 2,
						mt: 4,
						bgcolor: '#fff',
						border: '1px dashed #d1d5db',
						borderRadius: '12px',
						textAlign: 'center',
					}}
				>
					<TruckIcon sx={{ fontSize: 32, color: '#9ca3af' }} />
					<Typography variant='body2' sx={{ fontWeight: 600, color: '#374151' }}>
						Правил автоматических задач пока нет
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						Добавьте правило, чтобы настроить категорию и группу заявок после операций с оборудованием
					</Typography>
				</Box>
			) : (
				<Box>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
						<Typography
							variant='subtitle2'
							sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}
						>
							Правила
						</Typography>
						<Typography variant='caption' sx={{ color: '#6b7280' }}>
							· {tasks.length} {pluralize(tasks.length, ['правило', 'правила', 'правил'])}
						</Typography>
					</Box>

					<Box
						sx={{
							bgcolor: '#fff',
							border: '1px solid #e5e7eb',
							borderRadius: '12px',
							boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)',
							overflow: 'hidden',
						}}
					>
						{OPERATIONS.map(({ value, label, Icon }) => {
							const operationTasks = tasks.filter(task => task.operation === value)
							if (operationTasks.length === 0) return null
							return (
								<Box key={value}>
									<Box
										sx={{
											px: 2,
											py: 1.5,
											bgcolor: '#f9fafb',
											borderBottom: '1px solid #e5e7eb',
											display: 'flex',
											alignItems: 'center',
											gap: 1,
										}}
									>
										<Icon sx={{ fontSize: 18, color: '#6b7280' }} />
										<Typography variant='subtitle2' sx={{ fontWeight: 600, color: '#374151' }}>
											{label}
										</Typography>
										<Chip
											label={operationTasks.length}
											size='small'
											sx={{ bgcolor: '#e5e7eb', color: '#6b7280', fontWeight: 500 }}
										/>
									</Box>
									{operationTasks.map(task => (
										<Box
											key={task.id}
											sx={{
												py: 1.5,
												px: 2,
												borderBottom: '1px solid #f3f4f6',
												'&:last-child': { borderBottom: 'none' },
												'&:hover': { bgcolor: '#f9fafb' },
												display: 'flex',
												alignItems: 'center',
												gap: 1.5,
											}}
										>
											<Box sx={{ flexGrow: 1, minWidth: 0 }}>
												<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
													<Typography variant='subtitle1' sx={{ fontWeight: 600, color: '#1f2937' }}>
														{task.name}
													</Typography>
													<Chip
														label={task.enabled ? 'Активно' : 'Отключено'}
														size='small'
														sx={{
															bgcolor: task.enabled ? '#dcfce7' : '#f3f4f6',
															color: task.enabled ? '#166534' : '#6b7280',
															border: '1px solid',
															borderColor: task.enabled ? '#86efac' : '#d1d5db',
															fontWeight: 600,
															height: 20,
														}}
													/>
												</Box>
												<Typography variant='body2' sx={{ color: '#6b7280' }}>
													{task.title || 'Заголовок по умолчанию'}
												</Typography>
												<Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 0.5 }}>
													<Typography variant='caption' sx={{ color: '#9ca3af' }}>
														Категория: {task.categoryName || 'Не указана'}
													</Typography>
													<Typography variant='caption' sx={{ color: '#9ca3af' }}>
														Группа: {task.groupName || 'Не указана'}
													</Typography>
													<Typography variant='caption' sx={{ color: '#9ca3af' }}>
														Приоритет: {PRIORITY_MAP[task.priority as Priority]?.label ?? task.priority}
													</Typography>
												</Box>
											</Box>

											{canWrite && (
												<Box sx={{ display: 'flex', gap: 1, flexShrink: 0 }}>
													<Tooltip title={task.enabled ? 'Отключить' : 'Включить'}>
														<IconButton
															onClick={() => toggle(task)}
															sx={{
																color: '#6b7280',
																'&:hover': { color: '#2563eb', bgcolor: '#eff6ff' },
															}}
														>
															{task.enabled ? (
																<PauseIcon sx={{ fontSize: 16 }} />
															) : (
																<PlayIcon sx={{ fontSize: 16 }} />
															)}
														</IconButton>
													</Tooltip>
													<Tooltip title='Редактировать'>
														<IconButton
															onClick={() => {
																setEditTask(task)
																setDialogOpen(true)
															}}
															sx={{
																color: '#6b7280',
																'&:hover': { color: '#2563eb', bgcolor: '#eff6ff' },
															}}
														>
															<PencilIcon sx={{ fontSize: 16 }} />
														</IconButton>
													</Tooltip>
													<Tooltip title='Удалить'>
														<IconButton
															onClick={() => setDeleteTarget(task)}
															sx={{
																color: '#6b7280',
																'&:hover': { color: '#dc2626', bgcolor: '#fef2f2' },
															}}
														>
															<Trash2Icon sx={{ fontSize: 16 }} />
														</IconButton>
													</Tooltip>
												</Box>
											)}
										</Box>
									))}
								</Box>
							)
						})}
					</Box>
				</Box>
			)}

			<OperationTaskDialog
				key={editTask?.id ?? 'new'}
				open={dialogOpen}
				onClose={() => {
					setEditTask(null)
					setDialogOpen(false)
				}}
				taskId={editTask?.id ?? null}
			/>

			<ConfirmDialog
				open={Boolean(deleteTarget)}
				title='Удаление правила автоматической задачи'
				message={`Удалить правило «${deleteTarget?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteTarget(null)}
			/>
		</>
	)
}
