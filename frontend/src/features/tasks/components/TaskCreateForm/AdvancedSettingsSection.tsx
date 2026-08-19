import { Autocomplete, Box, Stack, TextField, Typography } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { Controller, useFormContext } from 'react-hook-form'
import dayjs from 'dayjs'
import type { IGroup } from '@/features/groups/types/group'
import type { IUserData, IUserShort } from '@/features/user/types/user'
import type { Priority } from '../../types/task'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { SectionCard } from './SectionCard'
import { PriorityCard } from './PriorityCard'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'
import { DateTextField } from '@/components/DatePicker/DatePicker'

type Props = {
	groups: IGroup[]
	users: IUserData[]
	assigneeOptions: IUserShort[]
	number?: number
}

export const AdvancedSettingsSection = ({ groups, users, assigneeOptions, number = 3 }: Props) => {
	const { control } = useFormContext<FormValues>()

	return (
		<SectionCard number={number} title='Расширенные настройки' subtitle='Доступно менеджерам'>
			<Stack sx={{ gap: 3 }}>
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
								<Autocomplete
									options={users}
									getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
									value={users.find(u => u.id === field.value) ?? null}
									onChange={(_, value) => field.onChange(value?.id ?? null)}
									noOptionsText='Нет пользователей'
									renderInput={params => (
										<TextField {...params} size='small' placeholder='Не указан' />
									)}
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
									options={assigneeOptions}
									getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
									value={assigneeOptions.find(u => u.id === field.value) ?? null}
									onChange={(_, value) => field.onChange(value?.id ?? null)}
									noOptionsText='Нет пользователей'
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
