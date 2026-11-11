import { useState, type FC } from 'react'
import { Box, Button, Typography, Chip, IconButton, Tooltip } from '@mui/material'
import { PlusIcon, PencilIcon, Trash2Icon, CopyIcon, PlayIcon, PauseIcon, ForwardIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentTypePeriodicTask } from '../../types/equipment'
import {
	useGetPeriodicTaskTemplatesQuery,
	useDeletePeriodicTaskTemplateMutation,
} from '../../equipmentApiSlice'
import { PeriodicTaskDialog } from '../Dialogs/PeriodicTaskDialog'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { useAppSelector } from '@/hooks/redux'
import { getRealm } from '@/features/realms/realmSlice'
import { EquipmentTypeIcon } from '../EquipmentTypeIcon'

// Секция настроек «Периодические задачи»: шаблоны периодических задач для типов оборудования.
export const PeriodicTasksSection: FC = () => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const realm = useAppSelector(getRealm)

	const { data } = useGetPeriodicTaskTemplatesQuery({ realmId: realm?.id ?? '' }, { skip: !realm?.id })
	const [remove] = useDeletePeriodicTaskTemplateMutation()

	const [dialogOpen, setDialogOpen] = useState(false)
	const [editTemplate, setEditTemplate] = useState<IEquipmentTypePeriodicTask | null>(null)
	const [deleteTarget, setDeleteTarget] = useState<IEquipmentTypePeriodicTask | null>(null)

	const templates = data?.data ?? []

	// Группировка по типу оборудования
	const groupedByType = templates.reduce<Record<string, IEquipmentTypePeriodicTask[]>>((acc, t) => {
		const key = t.equipmentTypeId
		if (!acc[key]) acc[key] = []
		acc[key].push(t)
		return acc
	}, {})

	const openCreate = () => {
		setEditTemplate(null)
		setDialogOpen(true)
	}

	const openEdit = (template: IEquipmentTypePeriodicTask) => {
		setEditTemplate(template)
		setDialogOpen(true)
	}

	const handleDelete = async () => {
		if (!deleteTarget?.id) return
		try {
			await remove(deleteTarget.id).unwrap()
			toast.success('Шаблон удалён')
			setDeleteTarget(null)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при удалении', { autoClose: false })
		}
	}

	// Человекочитаемая подпись периода
	const periodLabel = (t: IEquipmentTypePeriodicTask): string => {
		const parts: string[] = []
		if (t.periodYears > 0) parts.push(`${t.periodYears} г`)
		if (t.periodMonths > 0) parts.push(`${t.periodMonths} мес`)
		if (t.periodDays > 0) parts.push(`${t.periodDays} д`)
		return parts.length > 0 ? parts.join(' ') : 'Без периода'
	}

	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'flex-end',
					mb: { xs: 2, sm: -3 },
				}}
			>
				{canWrite && (
					<Button
						variant='outlined'
						onClick={openCreate}
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						sx={{
							textTransform: 'none',
							fontWeight: 500,
							borderRadius: '8px',
							gap: 1,
							bgcolor: '#fff',
							width: { xs: '100%', sm: 'auto' },
						}}
					>
						Добавить шаблон
					</Button>
				)}
			</Box>

			{templates.length === 0 ? (
				<Box
					sx={{
						display: 'flex',
						flexDirection: 'column',
						alignItems: 'center',
						gap: 1,
						py: 2,
						px: 2,
						mt: 5,
						bgcolor: '#fff',
						border: '1px dashed #d1d5db',
						borderRadius: '12px',
						textAlign: 'center',
					}}
				>
					<PlayIcon sx={{ fontSize: 32, color: '#9ca3af' }} />
					<Typography variant='body2' sx={{ fontWeight: 600, color: '#374151' }}>
						Шаблонов периодических задач пока нет
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						Шаблон определяет, какие заявки создавать автоматически для оборудования выбранного типа
					</Typography>
				</Box>
			) : (
				<Box>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
						<Typography
							variant='subtitle2'
							sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}
						>
							Шаблоны
						</Typography>
						<Typography variant='caption' sx={{ color: '#6b7280' }}>
							· {templates.length} {pluralize(templates.length, ['шаблон', 'шаблона', 'шаблонов'])}
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
						{Object.entries(groupedByType).map(([typeId, typeTemplates]) => {
							const typeName = typeTemplates[0]?.equipmentTypeName || 'Без типа'
							return (
								<Box key={typeId}>
									<Box
										sx={{
											px: 3,
											py: 1.5,
											bgcolor: '#f9fafb',
											borderBottom: '1px solid #e5e7eb',
											display: 'flex',
											alignItems: 'center',
											gap: 2,
										}}
									>
										<EquipmentTypeIcon
											isKit={false}
											typeName={typeName}
											icon=''
											sx={{ fontSize: 18 }}
										/>
										<Typography variant='subtitle2' sx={{ fontWeight: 600, color: '#374151' }}>
											{typeName}
										</Typography>
										<Chip
											label={`${typeTemplates.length} ${pluralize(typeTemplates.length, ['шаблон', 'шаблона', 'шаблонов'])}`}
											size='small'
											sx={{ bgcolor: '#e5e7eb', color: '#6b7280', fontWeight: 500 }}
										/>
									</Box>
									{typeTemplates.map(template => (
										<Box
											key={template.id}
											sx={{
												p: 3,
												borderBottom: '1px solid #f3f4f6',
												'&:last-child': { borderBottom: 'none' },
												'&:hover': { bgcolor: '#f9fafb' },
												transition: 'background-color 0.15s',
											}}
										>
											<Box sx={{ display: 'flex', gap: 3, alignItems: 'flex-start' }}>
												{/* Иконка статуса */}
												<Box
													sx={{
														width: 40,
														height: 40,
														borderRadius: '8px',
														bgcolor: template.enabled ? '#dbeafe' : '#f3f4f6',
														display: 'flex',
														alignItems: 'center',
														justifyContent: 'center',
														flexShrink: 0,
													}}
												>
													{template.enabled ? (
														<PlayIcon sx={{ fontSize: 18, color: '#2563eb' }} />
													) : (
														<PauseIcon sx={{ fontSize: 18, color: '#6b7280' }} />
													)}
												</Box>

												{/* Контент */}
												<Box sx={{ flexGrow: 1, minWidth: 0 }}>
													<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap', mb: 0.5 }}>
														<Typography variant='subtitle1' sx={{ fontWeight: 600, color: '#1f2937' }}>
															{template.title}
														</Typography>
														<Chip
															label={template.enabled ? 'Активен' : 'Неактивен'}
															size='small'
															sx={{
																bgcolor: template.enabled ? '#dcfce7' : '#f3f4f6',
																color: template.enabled ? '#166534' : '#6b7280',
																border: '1px solid',
																borderColor: template.enabled ? '#86efac' : '#d1d5db',
																fontWeight: 600,
																height: 20,
															}}
														/>
													</Box>

													{template.description && (
														<Typography variant='body2' sx={{ color: '#6b7280', mb: 1.5 }}>
															{template.description}
														</Typography>
													)}

													<Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap', alignItems: 'center' }}>
														<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#6b7280' }}>
															<PlayIcon sx={{ fontSize: 14 }} />
															<Typography variant='caption' sx={{ color: '#6b7280' }}>
																Период: <Typography component='span' sx={{ fontWeight: 600, color: '#374151' }}>
																	{periodLabel(template)}
																</Typography>
															</Typography>
														</Box>
														<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#6b7280' }}>
															<CopyIcon sx={{ fontSize: 14 }} />
															<Typography variant='caption' sx={{ color: '#6b7280' }}>
																Категория: <Typography component='span' sx={{ fontWeight: 600, color: '#374151' }}>
																	{template.categoryName || 'Не указана'}
																</Typography>
															</Typography>
														</Box>
														{template.siteName && (
															<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: '#6b7280' }}>
																<ForwardIcon sx={{ fontSize: 14 }} />
																<Typography variant='caption' sx={{ color: '#6b7280' }}>
																	Площадка: <Typography component='span' sx={{ fontWeight: 600, color: '#374151' }}>
																		{template.siteName}
																	</Typography>
																</Typography>
															</Box>
														)}
													</Box>
												</Box>

												{/* Действия */}
												{canWrite && (
													<Box sx={{ display: 'flex', gap: 1, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
														<Tooltip title='Редактировать'>
															<IconButton
																size='small'
																onClick={() => openEdit(template)}
																sx={{ color: '#6b7280', '&:hover': { color: '#2563eb', bgcolor: '#eff6ff' } }}
															>
																<PencilIcon sx={{ fontSize: 16 }} />
															</IconButton>
														</Tooltip>
														<Tooltip title='Удалить'>
															<IconButton
																size='small'
																onClick={() => setDeleteTarget(template)}
																sx={{ color: '#6b7280', '&:hover': { color: '#dc2626', bgcolor: '#fef2f2' } }}
															>
																<Trash2Icon sx={{ fontSize: 16 }} />
															</IconButton>
														</Tooltip>
													</Box>
												)}
											</Box>
										</Box>
									))}
								</Box>
							)
						})}
					</Box>
				</Box>
			)}

			<PeriodicTaskDialog
				key={editTemplate?.id ?? 'new'}
				open={dialogOpen}
				onClose={() => {
					setEditTemplate(null)
					setDialogOpen(false)
				}}
				templateId={editTemplate?.id ?? null}
				realmId={realm?.id ?? ''}
				equipmentTypeId={editTemplate?.equipmentTypeId ?? ''}
			/>

			<ConfirmDialog
				open={Boolean(deleteTarget)}
				title='Удаление шаблона периодической задачи'
				message={`Удалить шаблон «${deleteTarget?.title}»? Все связанные экземпляры будут удалены.`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteTarget(null)}
			/>
		</Box>
	)
}