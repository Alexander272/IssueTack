import { type FC, useState } from 'react'
import { Box, Stack, Typography, IconButton, Tooltip, Button, Switch, FormControlLabel } from '@mui/material'
import { Edit, Delete, PlayCircle, PlusIcon } from 'lucide-mui'

import {
	useGetPeriodicTaskTemplatesQuery,
	useDeletePeriodicTaskTemplateMutation,
	useRunPeriodicTaskMutation,
	useGetPeriodicTaskInstancesQuery,
} from '../../equipmentApiSlice'
import { PeriodicTaskDialog } from '../Dialogs/PeriodicTaskDialog'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IEquipment, IEquipmentTypePeriodicTask, IManualPeriodicTaskRunDTO } from '../../types/equipment'

type Props = {
	unit: IEquipment
}

// periodLabel собирает человекочитаемую подпись периода из годов/месяцев/дней.
const periodLabel = (t: IEquipmentTypePeriodicTask): string => {
	const parts: string[] = []
	if (t.periodYears > 0) parts.push(`${t.periodYears} г`)
	if (t.periodMonths > 0) parts.push(`${t.periodMonths} мес`)
	if (t.periodDays > 0) parts.push(`${t.periodDays} д`)
	return parts.length > 0 ? parts.join(' ') : 'Без периода'
}

export const PeriodicTasksTab: FC<Props> = ({ unit }) => {
	// shiftByTemplate — ручной запуск: какие шаблоны сдвигают следующий старт.
	const [shiftByTemplate, setShiftByTemplate] = useState<Record<string, boolean>>({})
	const [dialogOpen, setDialogOpen] = useState(false)
	const [editingTemplateId, setEditingTemplateId] = useState<string | null>(null)
	const [deletingTemplateId, setDeletingTemplateId] = useState<string | null>(null)

	const { data: templatesResponse } = useGetPeriodicTaskTemplatesQuery(
		{ realmId: unit.realmId, equipmentTypeId: unit.typeId ?? undefined },
		{ skip: !unit.typeId },
	)
	const safeTemplates = Array.isArray(templatesResponse?.data) ? templatesResponse.data : []

	const { data: instancesResponse } = useGetPeriodicTaskInstancesQuery({
		realmId: unit.realmId,
		equipmentId: unit.id,
		status: 'pending',
	})
	const nextPending = instancesResponse?.data?.[0]

	const [deleteTemplate] = useDeletePeriodicTaskTemplateMutation()
	const [runTask] = useRunPeriodicTaskMutation()

	const handleCreate = () => {
		setEditingTemplateId(null)
		setDialogOpen(true)
	}

	const handleEdit = (id: string) => {
		setEditingTemplateId(id)
		setDialogOpen(true)
	}

	const handleDelete = async (id: string) => {
		try {
			await deleteTemplate(id).unwrap()
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	const handleRun = async (templateId: string, shiftNext: boolean) => {
		try {
			const dto: IManualPeriodicTaskRunDTO = {
				realmId: unit.realmId,
				templateId,
				equipmentId: unit.id,
				shiftNext,
			}
			await runTask(dto).unwrap()
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	return (
		<Box>
			{safeTemplates.length === 0 ? (
				<Typography variant='body2'>Нет шаблонов для периодических задач</Typography>
			) : (
				<Stack spacing={2}>
					<Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
						<Button variant='contained' startIcon={<PlusIcon sx={{ fontSize: 16 }} />} onClick={handleCreate}>
							Создать
						</Button>
					</Box>
					{nextPending && (
						<Typography variant='body2' color='textSecondary'>
							Следующий старт: {nextPending.scheduledAt}
						</Typography>
					)}
					{safeTemplates.map(template => (
						<Box
							key={template.id}
							sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
						>
							<Box>
								<Typography variant='subtitle1' sx={{ fontWeight: 600 }}>
									{template.title}
								</Typography>
								<Typography variant='body2' color='textSecondary' sx={{ mt: 0.5 }}>
									Период: {periodLabel(template)}
								</Typography>
								<Typography variant='body2' color='textSecondary' sx={{ mt: 0.5 }}>
									{template.description && <>{template.description}</>}
								</Typography>
							</Box>
							<Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
								<FormControlLabel
									control={
										<Switch
											checked={!!shiftByTemplate[template.id]}
											onChange={e => setShiftByTemplate(prev => ({ ...prev, [template.id]: e.target.checked }))}
											color='primary'
											size='small'
										/>
									}
									label='Сдвинуть следующий старт'
								/>
								<Tooltip title='Запустить вручную'>
									<IconButton size='small' onClick={() => handleRun(template.id, !!shiftByTemplate[template.id])} color='primary'>
										<PlayCircle />
									</IconButton>
								</Tooltip>
								<Tooltip title='Редактировать'>
									<IconButton size='small' onClick={() => handleEdit(template.id)} color='primary'>
										<Edit />
									</IconButton>
								</Tooltip>
								<Tooltip title='Удалить'>
									<IconButton size='small' onClick={() => setDeletingTemplateId(template.id)} color='error'>
										<Delete />
									</IconButton>
								</Tooltip>
							</Box>
						</Box>
					))}
				</Stack>
			)}

			<PeriodicTaskDialog
				open={dialogOpen}
				onClose={() => setDialogOpen(false)}
				templateId={editingTemplateId}
				realmId={unit.realmId}
				equipmentTypeId={unit.typeId ?? ''}
			/>

			<ConfirmDialog
				open={deletingTemplateId != null}
				title='Удалить шаблон периодической задачи?'
				message='Шаблон будет удалён вместе со связанными экземплярами.'
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={() => deletingTemplateId && handleDelete(deletingTemplateId)}
				onCancel={() => setDeletingTemplateId(null)}
			/>
		</Box>
	)
}
