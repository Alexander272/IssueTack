import type { FC } from 'react'
import {
	Dialog,
	DialogTitle,
	DialogContent,
	DialogActions,
	Button,
	TextField,
	MenuItem,
	IconButton,
	Grid,
	Typography,
	Select,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IUserData, IUserDataDTO } from '@/features/user/types/user'
import { useGetRolesQuery } from '@/features/user/roleApiSlice'
import { useUpdateUserMutation } from '../usersApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { Fallback } from '@/components/Fallback/Fallback'
import { BoxFallback } from '@/components/Fallback/BoxFallback'

type Props = {
	user: IUserData | null
	onClose: () => void
}

export const UpdateModal: FC<Props> = ({ user, onClose }) => {
	const { data: roles, isFetching } = useGetRolesQuery(null)

	const { control, handleSubmit } = useForm<IUserDataDTO>({ values: user || undefined })

	const [update, { isLoading }] = useUpdateUserMutation()

	const saveHandler = handleSubmit(async form => {
		console.log('save user', form)

		try {
			await update(form).unwrap()
			toast.success('Пользователь обновлен')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	})

	if (isFetching) return <Fallback />
	return (
		<Dialog
			open={Boolean(user)}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{
				paper: {
					sx: {
						borderRadius: '16px',
						p: 1,
					},
				},
			}}
		>
			{/* Header */}
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Добавить пользователя
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			{isLoading && <BoxFallback />}

			{/* Body */}
			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Grid container spacing={2}>
					<Grid size={{ xs: 12, sm: 6 }}>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
							Имя
						</Typography>
						<Controller
							control={control}
							name='firstName'
							disabled
							render={({ field }) => <TextField {...field} fullWidth />}
						/>
					</Grid>
					<Grid size={{ xs: 12, sm: 6 }}>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
							Фамилия
						</Typography>
						<Controller
							control={control}
							name='lastName'
							disabled
							render={({ field }) => <TextField {...field} fullWidth />}
						/>
					</Grid>
					<Grid size={{ xs: 12 }}>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
							Email
						</Typography>
						<Controller
							control={control}
							name='email'
							disabled
							render={({ field }) => <TextField {...field} fullWidth />}
						/>
					</Grid>
					<Grid size={{ xs: 12, sm: 6 }}>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
							Роль
						</Typography>
						<Controller
							control={control}
							name='roleId'
							render={({ field }) => (
								<Select {...field} fullWidth>
									{roles?.data.map(role =>
										role.isEditable ? (
											<MenuItem key={role.id} value={role.id}>
												{role.name}
											</MenuItem>
										) : null,
									)}
								</Select>
							)}
						/>
					</Grid>
					<Grid size={{ xs: 12, sm: 6 }}>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
							Статус
						</Typography>
						<Controller
							control={control}
							name='isActive'
							render={({ field }) => (
								<Select
									value={field.value ? 'active' : 'inactive'}
									onChange={e => {
										field.onChange(e.target.value == 'active')
									}}
									fullWidth
								>
									<MenuItem value={'active'}>Активный</MenuItem>
									<MenuItem value={'inactive'}>Неактивный</MenuItem>
								</Select>
							)}
						/>
					</Grid>
				</Grid>
			</DialogContent>

			{/* Footer */}
			<DialogActions sx={{ p: 2, gap: 1 }}>
				<Button
					onClick={onClose}
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
				>
					Отмена
				</Button>
				<Button
					onClick={saveHandler}
					variant='contained'
					sx={{ borderRadius: '8px', textTransform: 'none', px: 3 }}
				>
					Сохранить
				</Button>
			</DialogActions>
		</Dialog>
	)
}
