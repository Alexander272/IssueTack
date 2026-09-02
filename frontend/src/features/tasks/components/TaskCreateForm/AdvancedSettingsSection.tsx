import { useEffect, useMemo } from 'react'
import { Autocomplete, Box, Stack, TextField, Typography } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { Controller, useFormContext, useWatch } from 'react-hook-form'
import dayjs from 'dayjs'
import type { Priority } from '../../types/task'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { CustomerSelector } from './CustomerSelector'
import { SectionCard } from './SectionCard'
import { PriorityCard } from './PriorityCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'
import { DateTextField } from '@/components/DatePicker/DatePicker'

type Props = {
	number?: number
	autoAssign?: boolean
	isAdmin?: boolean
	isManager?: boolean
}

export const AdvancedSettingsSection = ({ number = 3, autoAssign = true, isAdmin = true, isManager = true }: Props) => {
	const { control, setValue } = useFormContext<FormValues>()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: customersData } = useGetRealmUsersQuery('customers')
	const { data: executorsData } = useGetRealmUsersQuery('executors')

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const groups = useMemo(() => groupsData?.data ?? [], [groupsData])
	const customers = useMemo(() => customersData?.data ?? [], [customersData])
	const executors = useMemo(() => executorsData?.data ?? [], [executorsData])

	const selectedCategoryId = useWatch({ control, name: 'categoryId' })
	const selectedGroupId = useWatch({ control, name: 'groupId' })

	useEffect(() => {
		if (!autoAssign) return
		const cat = categories.find(c => c.id === selectedCategoryId)
		setValue('groupId', cat?.groupId ?? null)
	}, [selectedCategoryId, categories, autoAssign, setValue])

	useEffect(() => {
		if (!autoAssign) return
		if (!selectedGroupId) {
			setValue('assigneeId', null)
			return
		}
		const group = groups.find(g => g.id === selectedGroupId)
		setValue('assigneeId', group?.defaultAssigneeId ?? null)
	}, [selectedGroupId, groups, autoAssign, setValue])

	return (
		<SectionCard number={number} title='Расширенные настройки' subtitle='Доступно менеджерам'>
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

				<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							Заявитель
						</Typography>
						<Controller
							control={control}
							name='ownerId'
							render={({ field }) => (
								<CustomerSelector options={customers} value={field.value} onChange={field.onChange} />
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
										<TextField {...params} size='small' placeholder='Авто (по категории)' />
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
									disabled={!isAdmin && !isManager}
									noOptionsText='Нет исполнителей'
									renderInput={params => (
										<TextField {...params} size='small' placeholder='Авто (по группе)' />
									)}
								/>
							)}
						/>
					</Box>

					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							Срок выполнения
						</Typography>
						<Controller
							control={control}
							name='dueDate'
							render={({ field }) => (
								<DatePicker
									value={field.value ? dayjs(field.value) : null}
									onChange={date => field.onChange(date ? date.toISOString() : null)}
									slots={{
										textField: DateTextField,
									}}
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
