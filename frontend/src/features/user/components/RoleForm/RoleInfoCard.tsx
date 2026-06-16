import { Controller, useFormContext, useFormState, useWatch } from 'react-hook-form'
import { Box, Card, CardContent, IconButton, InputAdornment, Stack, TextField, Typography } from '@mui/material'

import type { IForm } from './UpdateRole'
import { generateSlug } from './utils'
import { RefreshIcon } from '@/components/Icons/RefreshIcon'

export const RoleInfoCard = () => {
	const { control, setValue } = useFormContext<IForm>()
	const { dirtyFields } = useFormState({ control })
	const nameValue = useWatch({ control, name: 'name' })
	const isSlugDirty = dirtyFields?.slug !== undefined

	const handleNameChange = (
		e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
		onChange: (value: string) => void,
	) => {
		const name = e.target.value
		onChange(name)
		if (!isSlugDirty) {
			setValue('slug', generateSlug(name))
		}
	}

	return (
		<Card sx={{ mb: 1.5, borderRadius: '16px', border: '1px solid #f3f4f6' }} elevation={0}>
			<CardContent>
				<Typography variant='h6' sx={{ mb: 2, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 1 }}>
					📋 Основная информация
				</Typography>
				<Stack spacing={2.5}>
					<Controller
						name='name'
						control={control}
						rules={{ required: 'Название роли обязательно' }}
						render={({ field: { onChange, value }, fieldState: { error } }) => (
							<TextField
								label='Название роли'
								placeholder='например, Старший менеджер'
								value={value}
								onChange={e => handleNameChange(e, onChange)}
								fullWidth
								required
								error={!!error}
								helperText={error?.message}
							/>
						)}
					/>
					<Box>
						<Controller
							name='slug'
							control={control}
							rules={{ required: 'Slug обязателен' }}
							render={({ field: { onChange, value }, fieldState: { error } }) => (
								<TextField
									label='Slug (идентификатор)'
									placeholder='senior_manager'
									value={value}
									onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
										onChange(e.target.value)
									}}
									fullWidth
									required
									error={!!error}
									helperText={error?.message}
									slotProps={{
										input: {
											endAdornment: (
												<InputAdornment position='end'>
													<IconButton
														onClick={() => setValue('slug', generateSlug(nameValue))}
														edge='end'
													>
														<RefreshIcon />
													</IconButton>
												</InputAdornment>
											),
										},
									}}
								/>
							)}
						/>
						<Typography variant='caption' color='text.secondary' sx={{ mt: -2 }}>
							Латиница, нижний регистр, дефисы. Используется в API.
						</Typography>
					</Box>

					<Controller
						name='description'
						control={control}
						render={({ field: { onChange, value } }) => (
							<TextField
								label='Описание'
								placeholder='Кратко опишите назначение роли и зону ответственности...'
								value={value}
								onChange={onChange}
								multiline
								rows={3}
								fullWidth
							/>
						)}
					/>

					<Box>
						<Controller
							name='level'
							control={control}
							render={({ field: { onChange, value } }) => (
								<TextField
									label='Порядок сортировки'
									type='number'
									value={value}
									onChange={e => onChange(Number(e.target.value))}
									fullWidth
									slotProps={{
										htmlInput: {
											min: 0,
											max: 10,
										},
									}}
								/>
							)}
						/>
						<Typography variant='caption' color='text.secondary' sx={{ mt: -2 }}>
							Число для управления порядком отображения ролей в списках и интерфейсах.
						</Typography>
					</Box>
				</Stack>
			</CardContent>
		</Card>
	)
}
