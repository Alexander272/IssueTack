import { Box, Button, Stack, TextField, Typography } from '@mui/material'
import { Controller, FormProvider, useForm, useFormContext } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { ITask, ITaskDTO } from '../../types/task'
import type { FormValues } from './types'
import { useAppSelector } from '@/hooks/redux'
import { getIsManager } from '@/features/user/userSlice'
import { useUpdateTaskMutation } from '../../tasksApiSlice'
import { AdvancedSettingsSection } from '../TaskCreateForm/AdvancedSettingsSection'
import { SectionCard } from '../TaskCreateForm/SectionCard'
import { fieldLabelSx } from '../TaskCreateForm/styles'

type Props = {
	task: ITask
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
}

const EditDescriptionSection = () => {
	const { control } = useFormContext<FormValues>()

	return (
		<SectionCard number={1} title='Заголовок и описание'>
			<Box>
				<Typography variant='caption' sx={fieldLabelSx}>
					Заголовок{' '}
					<Typography component='span' color='error'>
						*
					</Typography>
				</Typography>
				<Controller
					control={control}
					name='title'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<Box>
							<TextField
								{...field}
								fullWidth
								size='small'
								error={Boolean(fieldState.error)}
								helperText={fieldState.error?.message}
								slotProps={{ htmlInput: { maxLength: 150 } }}
							/>
							<Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', mt: 0.25 }}>
									{(field.value ?? '').length} / 150
								</Typography>
							</Box>
						</Box>
					)}
				/>
			</Box>

			<Box>
				<Typography variant='caption' sx={fieldLabelSx}>
					Подробное описание
				</Typography>
				<Controller
					control={control}
					name='description'
					render={({ field }) => <TextField {...field} fullWidth size='small' multiline minRows={6} />}
				/>
			</Box>
		</SectionCard>
	)
}

export const TaskEditForm = ({ task, onSuccess, onCancel, embedded }: Props) => {
	const isManager = useAppSelector(getIsManager)

	const [updateTask, { isLoading }] = useUpdateTaskMutation()

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
		},
	})
	const { handleSubmit, reset } = methods

	const onSubmit = handleSubmit(async data => {
		const dto: Omit<ITaskDTO, 'id'> & { id: string } = {
			id: task.id,
			title: data.title,
			description: data.description,
			status: task.status,
			priority: data.priority,
			realmId: task.realmId ?? '',
			siteId: task.site.id,
			categoryId: task.category.id,
			creatorId: task.creator.id,
			ownerId: data.ownerId || null,
			groupId: data.groupId || null,
			assigneeId: data.assigneeId || null,
			managerId: task.manager?.id ?? null,
			dueDate: data.dueDate || null,
		}

		try {
			await updateTask(dto).unwrap()
			toast.success('Задача обновлена')
			reset()
			onSuccess?.()
		} catch {
			// ошибка уже показана тостом в tasksApiSlice (updateTask.onQueryStarted)
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
						<EditDescriptionSection />

						{isManager && (
							<AdvancedSettingsSection
								number={2}
								autoAssign={false}
								isAdmin={task.access?.isAdmin ?? false}
								isManager={task.access?.isManager ?? false}
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
								disabled={isLoading}
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
