import { useEffect, useMemo } from 'react'
import { Box, Button, MenuItem, TextField, Typography, Autocomplete } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'
import dayjs from 'dayjs'

import type { IFetchError } from '@/app/types/error'
import type { Priority, ITaskDTO } from '../types/task'
import { PRIORITY_MAP } from '../constants/taskMaps'
import { useCreateTaskMutation } from '../tasksApiSlice'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetAvailableUsersQuery } from '@/features/user/usersApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useAppSelector } from '@/hooks/redux'
import { getRealm } from '@/features/realms/realmSlice'
import { useCheckPermission } from '@/features/user/hooks/check'
import { PermRules } from '@/features/access/constants/permissions'

interface FormValues {
	title: string
	description: string
	priority: Priority
	categoryId: string
	groupId: string | null
	assigneeId: string | null
	siteId: string
	dueDate: string | null
}

type Props = {
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
}

export const TaskCreateForm = ({ onSuccess, onCancel, embedded }: Props) => {
	const currentUserId = useAppSelector(state => state.user.id)
	const realm = useAppSelector(getRealm)
	const isManager = useCheckPermission(PermRules.Tasks.Write)

	const [createTask, { isLoading }] = useCreateTaskMutation()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: usersData } = useGetAvailableUsersQuery()
	const { data: sitesData } = useGetAllSitesQuery()

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const groups = useMemo(() => groupsData?.data ?? [], [groupsData])
	const users = useMemo(() => usersData?.data ?? [], [usersData])
	const sites = useMemo(() => sitesData?.data ?? [], [sitesData])

	const { control, handleSubmit, reset, setValue } = useForm<FormValues>({
		defaultValues: {
			title: '',
			description: '',
			priority: 'medium',
			categoryId: '',
			groupId: null,
			assigneeId: null,
			siteId: '',
			dueDate: null,
		},
	})

	const selectedCategoryId = useWatch({ control, name: 'categoryId' })

	useEffect(() => {
		const cat = categories.find(c => c.id === selectedCategoryId)
		if (cat) setValue('priority', cat.priority)
	}, [selectedCategoryId, categories, setValue])

	const categoryPriority = categories.find(c => c.id === selectedCategoryId)?.priority

	const onSubmit = handleSubmit(async data => {
		if (!currentUserId) {
			toast.error('Пользователь не найден')
			return
		}
		if (!realm?.id) {
			toast.error('Область не выбрана')
			return
		}

		const dto: ITaskDTO = {
			id: null,
			title: data.title,
			description: data.description,
			status: 'open',
			priority: isManager ? data.priority : categoryPriority || 'medium',
			realmId: realm.id,
			siteId: data.siteId,
			categoryId: data.categoryId,
			creatorId: currentUserId,
			ownerId: null,
			groupId: isManager ? data.groupId || null : null,
			assigneeId: isManager ? data.assigneeId || null : null,
			managerId: null,
			dueDate: isManager ? data.dueDate || null : null,
			closedAt: null,
		}

		try {
			await createTask(dto).unwrap()
			toast.success('Задача создана')
			reset()
			onSuccess?.()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при создании задачи', { autoClose: false })
		}
	})

	return (
		<Box sx={{ maxWidth: !embedded ? 720 : undefined, mx: 'auto' }}>
			{!embedded && (
				<Box sx={{ mb: 3 }}>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						Создание задачи
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.5 }}>
						Заполните форму для создания новой задачи
					</Typography>
				</Box>
			)}

			<Box
				component='form'
				onSubmit={onSubmit}
				sx={
					embedded
						? {}
						: { borderRadius: '12px', border: '1px solid #e5e7eb', p: 3, bgcolor: 'background.paper' }
				}
			>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
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
								<TextField
									{...field}
									fullWidth
									size='small'
									error={Boolean(fieldState.error)}
									helperText={fieldState.error?.message}
								/>
							)}
						/>
					</Box>

					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
							Описание
						</Typography>
						<Controller
							control={control}
							name='description'
							render={({ field }) => (
								<TextField {...field} fullWidth size='small' multiline minRows={3} />
							)}
						/>
					</Box>

					<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
						<Box>
							<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
								Категория{' '}
								<Typography component='span' color='error'>
									*
								</Typography>
							</Typography>
							<Controller
								control={control}
								name='categoryId'
								rules={{ required: 'Обязательное поле' }}
								render={({ field, fieldState }) => (
									<Autocomplete
										options={categories}
										getOptionLabel={o => o.name}
										value={categories.find(c => c.id === field.value) ?? null}
										onChange={(_, value) => field.onChange(value?.id ?? '')}
										noOptionsText='Нет категорий'
										renderInput={params => (
											<TextField
												{...params}
												size='small'
												error={Boolean(fieldState.error)}
												helperText={fieldState.error?.message}
												placeholder='Выберите категорию'
											/>
										)}
									/>
								)}
							/>
						</Box>

						<Box>
							<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
								Местонахождение{' '}
								<Typography component='span' color='error'>
									*
								</Typography>
							</Typography>
							<Controller
								control={control}
								name='siteId'
								rules={{ required: 'Обязательное поле' }}
								render={({ field, fieldState }) => (
									<Autocomplete
										options={sites}
										getOptionLabel={o => o.name}
										value={sites.find(s => s.id === field.value) ?? null}
										onChange={(_, value) => field.onChange(value?.id ?? '')}
										noOptionsText='Нет площадок'
										renderInput={params => (
											<TextField
												{...params}
												size='small'
												error={Boolean(fieldState.error)}
												helperText={fieldState.error?.message}
												placeholder='Выберите площадку'
											/>
										)}
									/>
								)}
							/>
						</Box>
					</Box>

					{isManager && (
						<>
							<Box sx={{ borderTop: '1px solid #e5e7eb', pt: 1 }}>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, color: '#6b7280', display: 'block', mb: 1.5 }}
								>
									Дополнительные параметры
								</Typography>
							</Box>

							<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
										Приоритет{' '}
										<Typography component='span' color='error'>
											*
										</Typography>
									</Typography>
									<Controller
										control={control}
										name='priority'
										rules={{ required: 'Обязательное поле' }}
										render={({ field }) => (
											<TextField {...field} select fullWidth size='small'>
												{Object.entries(PRIORITY_MAP).map(([value, info]) => (
													<MenuItem key={value} value={value}>
														{info.label}
													</MenuItem>
												))}
											</TextField>
										)}
									/>
								</Box>

								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
										Группа
									</Typography>
									<Controller
										control={control}
										name='groupId'
										render={({ field }) => (
											<Autocomplete
												options={groups}
												getOptionLabel={o => o.name}
												value={groups.find(g => g.id === field.value) ?? null}
												onChange={(_, value) => field.onChange(value?.id ?? null)}
												noOptionsText='Нет групп'
												renderInput={params => (
													<TextField {...params} size='small' placeholder='Не выбрана' />
												)}
											/>
										)}
									/>
								</Box>

								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
										Исполнитель
									</Typography>
									<Controller
										control={control}
										name='assigneeId'
										render={({ field }) => (
											<Autocomplete
												options={users}
												getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
												value={users.find(u => u.id === field.value) ?? null}
												onChange={(_, value) => field.onChange(value?.id ?? null)}
												noOptionsText='Нет пользователей'
												renderInput={params => (
													<TextField {...params} size='small' placeholder='Не назначен' />
												)}
											/>
										)}
									/>
								</Box>

								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
										Срок выполнения
									</Typography>
									<Controller
										control={control}
										name='dueDate'
										render={({ field }) => (
											<DatePicker
												value={field.value ? dayjs(field.value) : null}
												onChange={date => field.onChange(date ? date.toISOString() : null)}
												slotProps={{ textField: { fullWidth: true, size: 'small' } }}
											/>
										)}
									/>
								</Box>
							</Box>
						</>
					)}

					<Box
						sx={{
							display: 'flex',
							justifyContent: 'flex-end',
							gap: 1,
							pt: 1,
							borderTop: '1px solid #f3f4f6',
						}}
					>
						<Button
							type='button'
							variant='outlined'
							onClick={embedded ? onCancel : () => reset()}
							sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
						>
							{embedded ? 'Отмена' : 'Очистить'}
						</Button>
						<Button
							type='submit'
							variant='contained'
							disabled={isLoading}
							sx={{ textTransform: 'none', px: 3 }}
						>
							{isLoading ? 'Создание...' : 'Создать'}
						</Button>
					</Box>
				</Box>
			</Box>
		</Box>
	)
}
