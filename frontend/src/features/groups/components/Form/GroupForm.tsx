import { type FC } from 'react'
import { Box, MenuItem, TextField, Typography } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { IGroupDTO } from '../../types/group'
import type { IUserData } from '@/features/user/types/user'

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

			<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
				<Box>
					<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
						Руководитель группы
					</Typography>
					<Controller
						control={control}
						name='managerId'
						render={({ field }) => (
							<TextField
								{...field}
								value={field.value ?? ''}
								onChange={e => field.onChange(e.target.value || null)}
								select
								fullWidth
							>
								<MenuItem value=''>Не назначен</MenuItem>
								{availableUsers.map(u => (
									<MenuItem key={u.id} value={u.id}>
										{u.lastName} {u.firstName}
									</MenuItem>
								))}
							</TextField>
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
							<TextField
								{...field}
								value={field.value ?? ''}
								onChange={e => field.onChange(e.target.value || null)}
								select
								fullWidth
							>
								<MenuItem value=''>Не назначен</MenuItem>
								{availableUsers.map(u => (
									<MenuItem key={u.id} value={u.id}>
										{u.lastName} {u.firstName}
									</MenuItem>
								))}
							</TextField>
						)}
					/>
					<Typography variant='caption' sx={{ color: '#9ca3af', mt: 0.5, display: 'block' }}>
						Должен быть участником группы
					</Typography>
				</Box>
			</Box>

			<Box>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
					Участники группы
				</Typography>
				<Controller
					control={control}
					name='memberIds'
					render={({ field }) => (
						<TextField
							{...field}
							value={field.value}
							onChange={e => {
								const selected = e.target.value as unknown as string[]
								field.onChange(selected)
							}}
							select
							fullWidth
							slotProps={{
								select: {
									multiple: true,
									renderValue: (selected: unknown) => {
										const ids = selected as string[]
										return ids.length
											? ids
													.map(id => {
														const user = users.find(u => u.id === id)
														return user ? `${user.lastName} ${user.firstName}` : id
													})
													.join(', ')
											: 'Не выбраны'
									},
								},
							}}
						>
							{users.map(u => (
								<MenuItem key={u.id} value={u.id}>
									{u.lastName} {u.firstName} ({u.email})
								</MenuItem>
							))}
						</TextField>
					)}
				/>
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
