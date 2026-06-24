import { type FC } from 'react'
import { MenuItem, Stack, TextField, Typography } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { ICategoryDTO } from '../../types/category'
import type { IGroup } from '@/features/groups/types/group'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { Switch } from '@/components/Switch/Switch'

type Props = {
	control: Control<ICategoryDTO>
	groups: IGroup[]
}

export const Form: FC<Props> = ({ control, groups }) => {
	return (
		<Stack spacing={2}>
			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Название <Typography component='span' color='error'>*</Typography>
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
			</Stack>

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Описание
				</Typography>
				<Controller
					control={control}
					name='description'
					render={({ field }) => <TextField {...field} fullWidth multiline minRows={3} />}
				/>
			</Stack>

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Группа-владелец <Typography component='span' color='error'>*</Typography>
				</Typography>
				<Controller
					control={control}
					name='groupId'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<TextField {...field} select fullWidth error={Boolean(fieldState.error)} helperText={fieldState.error?.message}>
							{groups.map(g => (
								<MenuItem key={g.id} value={g.id}>{g.name}</MenuItem>
							))}
						</TextField>
					)}
				/>
			</Stack>

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Приоритет по умолчанию <Typography component='span' color='error'>*</Typography>
				</Typography>
				<Controller
					control={control}
					name='priority'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<TextField {...field} select fullWidth error={Boolean(fieldState.error)} helperText={fieldState.error?.message}>
							{Object.entries(PRIORITY_MAP).map(([value, info]) => (
								<MenuItem key={value} value={value}>{info.label}</MenuItem>
							))}
						</TextField>
					)}
				/>
			</Stack>

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Статус
				</Typography>
				<Controller
					control={control}
					name='isActive'
					render={({ field }) => (
						<Switch value={field.value} onChange={field.onChange} labels={['Неактивна', 'Активна']} />
					)}
				/>
			</Stack>
		</Stack>
	)
}
