import { Box, Button, Stack, Typography } from '@mui/material'
import { FormProvider, useForm } from 'react-hook-form'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'react-toastify'

import type { ITask, ITaskDTO } from '../../types/task'
import type { FormValues } from './types'
import { useAppSelector } from '@/hooks/redux'
import { getHasNoGroups } from '@/features/user/userSlice'
import { useUpdateTaskMutation } from '../../tasksApiSlice'
import { AdvancedSettingsSection } from '../TaskCreateForm/AdvancedSettingsSection'
import { CategoryAndSiteSection } from '../TaskCreateForm/CategoryAndSiteSection'
import { CustomerAndDateSection } from '../TaskCreateForm/CustomerAndDateSection'
import { DescriptionSection } from '../TaskCreateForm/DescriptionSection'
import { TaskFormProvider } from '../TaskCreateForm/TaskFormProvider'
import type { TaskFormSettings } from '../TaskCreateForm/taskFormMode'

type Props = {
	task: ITask
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
	onSavingChange?: (saving: boolean) => void
}

export const TaskEditForm = ({ task, onSuccess, onCancel, embedded, onSavingChange }: Props) => {
	const hasNoGroups = useAppSelector(getHasNoGroups)
	const canEditNotifiers = !hasNoGroups

	// Управление заявкой: менеджер её группы или начальник области. Тот же флаг
	// считает бэкенд (AccessFlags.CanManage), поэтому начальник области без
	// управляемых групп тоже получает расширенные настройки.
	const canManage = Boolean(task.access?.canManage)
	// Заказчика и срок безгрупповой пользователь не видит; менять их можно, только
	// пока заявка новая (у владельца дальше права на поля уже нет).
	const canEditOwnerAndDueDate = !canManage && canEditNotifiers && task.status === 'open'

	// Заголовок/описание может править только создатель/менеджер группы/владелец
	// (canEditFields). Начальник области (task.access.isAdmin) правит через форму
	// остальные поля — категорию, площадку, приоритет, срок, группу, исполнителя —
	// но не содержимое (см. режим adminOnlyEdit в TicketService.Update).
	const canEditContent = Boolean(task.access?.canEditFields)

	const formSettings = useMemo<TaskFormSettings>(
		() => ({
			mode: 'edit',
			creatorId: task.creator.id,
			canEditContent,
			canEditDueDate: canManage,
			canPickAssignee: true,
			canEditPriority: canManage,
			canEditGroup: canManage,
			canEditOwnerMuted: canManage,
			canEditNotifiers,
			showOwner: canEditOwnerAndDueDate,
			showDueDate: canEditOwnerAndDueDate,
			showNotifierFieldLabel: canEditOwnerAndDueDate,
		}),
		[task.creator.id, canEditContent, canManage, canEditNotifiers, canEditOwnerAndDueDate],
	)

	const [updateTask, { isLoading }] = useUpdateTaskMutation()

	useEffect(() => {
		onSavingChange?.(isLoading)
	}, [isLoading, onSavingChange])

	const methods = useForm<FormValues>({
		defaultValues: {
			title: task.title,
			description: task.description,
			priority: task.priority,
			categoryId: task.category.id,
			groupId: task.group?.id ?? null,
			ownerId: task.owner?.id ?? null,
			assigneeId: task.assignee?.id ?? null,
			siteId: task.site.id,
			dueDate: task.dueDate ?? null,
			subtasks: [],
			notifierIds: (task.notifiers ?? []).map(u => u.id),
			ownerMuted: task.ownerMuted ?? false,
			equipmentId: task.equipment?.id ?? null,
		},
		mode: 'onTouched',
	})
	const { handleSubmit, reset, formState } = methods
	const [submitting, setSubmitting] = useState(false)

	const onSubmit = handleSubmit(async data => {
		if (submitting) return
		setSubmitting(true)
		try {
			const dto: Omit<ITaskDTO, 'id'> & { id: string } = {
				id: task.id,
				title: data.title.trim(),
				description: data.description,
				status: task.status,
				priority: data.priority,
				realmId: task.realmId ?? '',
				siteId: data.siteId,
				categoryId: data.categoryId,
				creatorId: task.creator.id,
				ownerId: data.ownerId || null,
				groupId: data.groupId || null,
				assigneeId: data.assigneeId || null,
				managerId: task.manager?.id ?? null,
				dueDate: data.dueDate || null,
				equipmentId: data.equipmentId || null,
				...(canEditNotifiers ? { notifierIds: data.notifierIds ?? [] } : {}),
				ownerMuted: Boolean(data.ownerMuted && data.ownerId),
			}

			try {
				await updateTask(dto).unwrap()
				toast.success('Задача обновлена')
				reset()
				onSuccess?.()
			} catch {
				// ошибка уже показана тостом в tasksApiSlice (updateTask.onQueryStarted)
			}
		} finally {
			setSubmitting(false)
		}
	})

	return (
		<Box sx={{ maxWidth: !embedded ? 720 : undefined, mx: 'auto' }}>
			{!embedded && (
				<Box sx={{ mb: 3 }}>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						Редактирование задачи
					</Typography>
				</Box>
			)}

			<TaskFormProvider settings={formSettings}>
				<FormProvider {...methods}>
					<Box component='form' onSubmit={onSubmit}>
						<Stack sx={{ gap: 1 }}>
							<CategoryAndSiteSection number={1} />
							<DescriptionSection readOnly={!canEditContent} />

							{canManage && <AdvancedSettingsSection number={3} />}

							{!canManage && canEditNotifiers && (
								<CustomerAndDateSection
									number={3}
									title={canEditOwnerAndDueDate ? 'Заказчик, сроки и уведомления' : 'Уведомлять'}
									subtitle={
										canEditOwnerAndDueDate
											? 'Пока задача новая, можно изменить заказчика, срок и список уведомляемых'
											: ''
									}
								/>
							)}

							<Box
								sx={{
									display: 'flex',
									justifyContent: 'flex-end',
									gap: 1,
									pt: 1,
									pb: !embedded ? 2 : 0,
								}}
							>
								<Button
									type='button'
									variant='outlined'
									onClick={embedded ? onCancel : () => reset()}
									sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
								>
									{embedded ? 'Отмена' : 'Сбросить'}
								</Button>
								<Button
									type='submit'
									variant='contained'
									disabled={isLoading || !formState.isValid || !formState.isDirty}
									sx={{ textTransform: 'none', px: 3 }}
								>
									{isLoading ? 'Сохранение...' : 'Сохранить'}
								</Button>
							</Box>
						</Stack>
					</Box>
				</FormProvider>
			</TaskFormProvider>
		</Box>
	)
}
