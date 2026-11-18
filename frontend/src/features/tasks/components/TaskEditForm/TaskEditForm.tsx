import { Box, Button, Stack, Typography } from '@mui/material'
import { FormProvider, useForm } from 'react-hook-form'
import { useEffect, useState } from 'react'
import { toast } from 'react-toastify'

import type { ITask, ITaskDTO } from '../../types/task'
import type { FormValues } from './types'
import { useAppSelector } from '@/hooks/redux'
import { getCurrentCapabilities, getHasNoGroups, getIsManager } from '@/features/user/userSlice'
import { useUpdateTaskMutation } from '../../tasksApiSlice'
import { AdvancedSettingsSection } from '../TaskCreateForm/AdvancedSettingsSection'
import { CategoryAndSiteSection } from '../TaskCreateForm/CategoryAndSiteSection'
import { CustomerAndDateSection } from '../TaskCreateForm/CustomerAndDateSection'
import { DescriptionSection } from '../TaskCreateForm/DescriptionSection'

type Props = {
	task: ITask
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
	onSavingChange?: (saving: boolean) => void
}

export const TaskEditForm = ({ task, onSuccess, onCancel, embedded, onSavingChange }: Props) => {
	const isManager = useAppSelector(getIsManager)
	const hasNoGroups = useAppSelector(getHasNoGroups)
	const capabilities = useAppSelector(getCurrentCapabilities)
	const canEditNotifiers = !hasNoGroups
	const canEditOwnerAndDueDate = !isManager && canEditNotifiers && task.status === 'open'

	const [updateTask, { isLoading }] = useUpdateTaskMutation()

	// Заголовок/описание может править только создатель/менеджер группы/владелец
	// (canEditFields). Начальник области (task.access.isAdmin) правит через форму
	// остальные поля — категорию, площадку, приоритет, срок, группу, исполнителя —
	// но не содержимое (см. режим adminOnlyEdit в TicketService.Update).
	const canEditContent = Boolean(task.access?.canEditFields)

	// Формула прав доступа для расширенных настроек: менеджер группы или админ реалма
	const isAdmin = Boolean(task.access?.isManager) || capabilities.isRealmAdmin

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

			<FormProvider {...methods}>
				<Box component='form' onSubmit={onSubmit}>
					<Stack sx={{ gap: 1 }}>
						<CategoryAndSiteSection number={1} />
						<DescriptionSection readOnly={!canEditContent} />

						{isManager && (
							<AdvancedSettingsSection
								number={3}
								autoAssign={false}
								canEditDueDate={isAdmin}
								canEditGroup={isAdmin}
								canEditPriority={isAdmin}
								canEditOwnerMuted={isAdmin}
								creatorId={task.creator.id}
							/>
						)}

						{!isManager && canEditNotifiers && (
							<CustomerAndDateSection
								number={3}
								creatorId={task.creator.id}
								title={canEditOwnerAndDueDate ? 'Заказчик, сроки и уведомления' : 'Уведомлять'}
								subtitle={
									canEditOwnerAndDueDate
										? 'Пока задача новая, можно изменить заказчика, срок и список уведомляемых'
										: ''
								}
								showOwner={canEditOwnerAndDueDate}
								showDueDate={canEditOwnerAndDueDate}
								showNotifierFieldLabel={canEditOwnerAndDueDate}
							/>
						)}

						<Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, pt: 1, pb: !embedded ? 2 : 0 }}>
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
		</Box>
	)
}
