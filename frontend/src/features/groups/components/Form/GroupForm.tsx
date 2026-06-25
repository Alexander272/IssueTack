import { type FC } from 'react'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { IGroupDTO } from '../../types/group'
import type { IUserData } from '@/features/user/types/user'
import { MemberPicker } from './MemberPicker'

type Props = {
	control: Control<IGroupDTO>
	users: IUserData[]
	availableUsers: IUserData[]
}

export const GroupForm: FC<Props> = ({ control, users, availableUsers }) => {
	return (
		<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
			<Box>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Название группы{' '}
					<Typography component='span' color='error'>
						*
					</Typography>
				</Typography>
				<Controller
					control={control}
					name='name'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<TextField
							{...field}
							fullWidth
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
					render={({ field }) => <TextField {...field} fullWidth multiline minRows={3} />}
				/>
			</Box>

			<Box>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
					Участники группы
				</Typography>
				<Controller
					control={control}
					name='memberIds'
					render={({ field }) => (
						<MemberPicker value={field.value ?? []} onChange={field.onChange} users={users} />
					)}
				/>
			</Box>

			<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
				<Box>
					<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
						Руководитель группы
					</Typography>
					<Controller
						control={control}
						name='managerId'
						render={({ field }) => (
							<Autocomplete
								options={availableUsers}
								getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
								value={availableUsers.find(u => u.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? null)}
								noOptionsText='Нет доступных пользователей'
								renderInput={params => <TextField {...params} placeholder='Не назначен' />}
							/>
						)}
					/>
					<Typography variant='caption' sx={{ color: '#9ca3af', mt: 0.5, display: 'block' }}>
						Должен быть участником группы
					</Typography>
				</Box>

				<Box>
					<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
						Исполнитель по умолчанию
					</Typography>
					<Controller
						control={control}
						name='defaultAssigneeId'
						render={({ field }) => (
							<Autocomplete
								options={availableUsers}
								getOptionLabel={u => `${u.lastName} ${u.firstName} (${u.username})`}
								value={availableUsers.find(u => u.id === field.value) ?? null}
								onChange={(_, value) => field.onChange(value?.id ?? null)}
								noOptionsText='Нет доступных пользователей'
								renderInput={params => <TextField {...params} placeholder='Не назначен' />}
							/>
						)}
					/>
					<Typography variant='caption' sx={{ color: '#9ca3af', mt: 0.5, display: 'block' }}>
						Должен быть участником группы
					</Typography>
				</Box>
			</Box>

			<Box sx={{ p: 2, bgcolor: '#eff6ff', border: '1px solid #bfdbfe', borderRadius: '8px' }}>
				<Typography variant='caption' sx={{ color: '#1e40af' }}>
					<strong>Примечание:</strong> Удаление участника из группы не повлияет на задачи, уже назначенные
					этому пользователю через данную группу.
				</Typography>
			</Box>
		</Box>
	)
}
