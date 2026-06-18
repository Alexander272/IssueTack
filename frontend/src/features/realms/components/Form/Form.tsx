import { type FC } from 'react'
import { TextField, Typography, Stack } from '@mui/material'
import { Controller, type Control } from 'react-hook-form'

import type { IRealmDTO } from '@/features/realms/types/realm'
import { Switch } from '@/components/Switch/Switch'

type Props = {
	control: Control<IRealmDTO>
}

export const Form: FC<Props> = ({ control }) => {
	return (
		<Stack spacing={2}>
			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Название
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
					Slug
				</Typography>
				<Controller
					control={control}
					name='slug'
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
