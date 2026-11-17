import { useEffect, useMemo, useRef } from 'react'
import { Autocomplete, Box, Stack, TextField, Typography } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { Controller, useFormContext, useWatch } from 'react-hook-form'
import dayjs from 'dayjs'
import type { Priority } from '../../types/task'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { CustomerSelector } from './CustomerSelector'
import { NotifierSelector } from './NotifierSelector'
import { SectionCard } from './SectionCard'
import { PriorityCard } from './PriorityCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'
import { DateTextField } from '@/components/DatePicker/DatePicker'

type Props = {
	number?: number
	autoAssign?: boolean
	isAdmin?: boolean
	/** Можно ли менять срок: только менеджер группы заявки или админ реалма. */
	canEditDueDate?: boolean
	/** Можно ли выбирать исполнителя: менеджер группы/админ реалма или исполнитель области. */
	canPickAssignee?: boolean
	/** Автор заявки — исключается из списка «Уведомлять». */
	creatorId?: string | null
	title?: string
	subtitle?: string
}

export const AdvancedSettingsSection = ({
	number = 4,
	autoAssign = true,
	isAdmin = true,
	canEditDueDate = true,
	canPickAssignee = true,
	creatorId = null,
	title = 'Расширенные настройки',
	subtitle = '',
}: Props) => {
	const { control, setValue } = useFormContext<FormValues>()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: ownersData } = useGetRealmUsersQuery('all')
	const { data: executorsData } = useGetRealmUsersQuery('executors')
	const { data: allUsersData } = useGetRealmUsersQuery('all')

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const groups = useMemo(() => groupsData?.data ?? [], [groupsData])
	const owners = useMemo(() => ownersData?.data ?? [], [ownersData])
	const executors = useMemo(() => executorsData?.data ?? [], [executorsData])
	const allUsers = useMemo(() => allUsersData?.data ?? [], [allUsersData])

	const selectedCategoryId = useWatch({ control, name: 'categoryId' })
	const selectedGroupId = useWatch({ control, name: 'groupId' })
	const selectedOwnerId = useWatch({ control, name: 'ownerId' })

	// Авто-назначение должно срабатывать только когда пользователь реально меняет
	// категорию/группу. При восстановлении черновика categoryId/groupId уже
	// заполнены на первом рендере, и без этой защиты эффекты затирали бы
	// выбранные вручную группу/исполнителя значениями по умолчанию.
	const prevCategoryId = useRef(selectedCategoryId)
	const prevGroupId = useRef(selectedGroupId)

	useEffect(() => {
		if (!autoAssign) return
		if (prevCategoryId.current === selectedCategoryId) return
		prevCategoryId.current = selectedCategoryId
		const cat = categories.find(c => c.id === selectedCategoryId)
		setValue('groupId', cat?.groupId ?? null)
		// Автосрок из категории: форма показывает тот же дефолт, который бэкенд
		// применил бы сам, пока dueDate пуст (только на создании — autoAssign).
		// Срок ставится на конец выбранного дня, как и при ручном выборе даты.
		// Категория без автосрока сбрасывает поле: иначе в форме остался бы срок
		// предыдущей категории и уехал бы на заявку, хотя бэкенд его не применял бы.
		setValue(
			'dueDate',
			cat?.defaultDueDays != null ? dayjs().add(cat.defaultDueDays, 'day').endOf('day').toISOString() : null,
		)
	}, [selectedCategoryId, categories, autoAssign, setValue])

	useEffect(() => {
		if (!autoAssign) return
		if (prevGroupId.current === selectedGroupId) return
		prevGroupId.current = selectedGroupId
		if (!selectedGroupId) {
			setValue('assigneeId', null)
			return
		}
		const group = groups.find(g => g.id === selectedGroupId)
		setValue('assigneeId', group?.defaultAssigneeId ?? null)
	}, [selectedGroupId, groups, autoAssign, setValue])

	return (
		<SectionCard number={number} title={title} subtitle={subtitle}>
			<Stack sx={{ gap: 2 }}>
				<Box>
					<Typography variant='caption' sx={fieldLabelSx}>
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
							<Box
								sx={{
									display: 'grid',
									gridTemplateColumns: { xs: '1fr', sm: 'repeat(4, 1fr)' },
									gap: 1.5,
								}}
							>
								{(Object.keys(PRIORITY_MAP) as Priority[]).map(value => (
									<PriorityCard
										key={value}
										value={value}
										selected={value === field.value}
										onSelect={v => field.onChange(v)}
									/>
								))}
							</Box>
						)}
					/>
				</Box>

				<Box
					sx={{
						display: 'grid',
						gridTemplateColumns: { xs: 'minmax(0, 1fr)', sm: 'minmax(0, 1fr) minmax(0, 1fr)' },
						gap: 2,
					}}
				>
					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							Заказчик
						</Typography>
						<Controller
							control={control}
							name='ownerId'
							render={({ field }) => (
								<CustomerSelector options={owners} value={field.value} onChange={field.onChange} />
							)}
						/>
					</Box>

					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							Уведомлять
						</Typography>
						<Controller
							control={control}
							name='notifierIds'
							render={({ field }) => (
								<NotifierSelector
									options={allUsers}
									value={field.value ?? []}
									onChange={field.onChange}
									excludeIds={[creatorId, selectedOwnerId]}
								/>
							)}
						/>
					</Box>

					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
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
									disabled={!isAdmin}
									noOptionsText='Нет групп'
									renderInput={params => (
										<TextField {...params} size='small' placeholder={autoAssign ? 'Авто (по категории)' : 'Выберите группу'} />
									)}
								/>
							)}
						/>
					</Box>

					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							Исполнитель
						</Typography>
						<Controller
							control={control}
							name='assigneeId'
							render={({ field }) => (
								<Autocomplete
									options={executors}
									getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
									value={executors.find(u => u.id === field.value) ?? null}
									onChange={(_, value) => field.onChange(value?.id ?? null)}
									disabled={!canPickAssignee}
									noOptionsText='Нет исполнителей'
									renderInput={params => (
										<TextField {...params} size='small' placeholder={autoAssign ? 'Авто (по группе)' : 'Выберите исполнителя'} />
									)}
								/>
							)}
						/>
					</Box>

					<Box sx={{ gridColumn: { xs: 'span 1', sm: 'span 2' } }}>
						<Typography variant='caption' sx={fieldLabelSx}>
							Срок выполнения
						</Typography>
						<Controller
							control={control}
							name='dueDate'
							render={({ field }) => (
								<DateTimePicker
									value={field.value ? dayjs(field.value) : null}
									onChange={date =>
										field.onChange(
											date ? (field.value ? date : date.endOf('day')).toISOString() : null,
										)
									}
									disabled={!canEditDueDate}
									slots={{ textField: DateTextField }}
									slotProps={{ textField: { fullWidth: true, size: 'small' } }}
								/>
							)}
						/>
					</Box>
				</Box>
			</Stack>
		</SectionCard>
	)
}
