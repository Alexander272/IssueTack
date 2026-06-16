import { useState } from 'react'
import {
	Button,
	Checkbox,
	FormControlLabel,
	InputAdornment,
	LinearProgress,
	Stack,
	TextField,
	Typography,
	useTheme,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ISignIn } from '../types/auth'
import { useAppDispatch } from '@/hooks/redux'
import { setUser } from '@/features/user/userSlice'
import { VisibleIcon } from '@/components/Icons/VisibleIcon'
import { InVisibleIcon } from '@/components/Icons/InVisibleIcon'
import { useSignInMutation } from '../authApiSlice'

const rememberKey = 'remember'
const defaultValues: ISignIn = { username: '', password: '', remember: false }

export const SignInForm = () => {
	const [passIsVisible, setPassIsVisible] = useState(false)
	const { palette } = useTheme()

	const dispatch = useAppDispatch()

	const {
		control,
		handleSubmit,
		formState: { errors },
	} = useForm<ISignIn>({ defaultValues: JSON.parse(localStorage.getItem(rememberKey) || 'null') || defaultValues })

	const [signIn, { isLoading }] = useSignInMutation()

	const togglePassVisible = () => setPassIsVisible(prev => !prev)

	const signInHandler = async (data: ISignIn) => {
		if (data.remember) {
			localStorage.setItem(rememberKey, JSON.stringify(data))
		} else {
			localStorage.removeItem(rememberKey)
		}

		try {
			const payload = await signIn(data).unwrap()
			dispatch(setUser(payload.data))
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	}

	return (
		<Stack component='form' onSubmit={handleSubmit(signInHandler)} sx={{ position: 'relative' }}>
			{isLoading ? <LinearProgress sx={{ position: 'absolute', bottom: -20, left: 0, right: 0 }} /> : null}

			<Typography
				variant='h2'
				align='center'
				fontSize={'1.5rem'}
				color={palette.primary.main}
				paddingBottom={1.25}
				mb={1.25}
				fontWeight={'bold'}
				lineHeight={'inherit'}
				sx={{ borderBottom: '1px solid #e5e4e9', letterSpacing: '1.2px' }}
			>
				Вход
			</Typography>

			<Stack spacing={2} marginTop={2}>
				<Controller
					control={control}
					name='username'
					rules={{ required: true }}
					render={({ field }) => (
						<TextField
							name={field.name}
							value={field.value}
							onChange={field.onChange}
							placeholder='Имя пользователя'
							error={Boolean(errors.username)}
							helperText={errors.username ? 'Поле не может быть пустым' : ''}
							disabled={isLoading}
							sx={{ '& .MuiOutlinedInput-root': { borderRadius: 10 } }}
						/>
					)}
				/>

				<Controller
					control={control}
					name='password'
					rules={{ required: true }}
					render={({ field }) => (
						<TextField
							name={field.name}
							value={field.value}
							onChange={field.onChange}
							type={passIsVisible ? 'text' : 'password'}
							placeholder='Пароль'
							error={Boolean(errors.password)}
							helperText={errors.password ? 'Поле не может быть пустым' : ''}
							disabled={isLoading}
							sx={{ '& .MuiOutlinedInput-root': { borderRadius: 10, paddingRight: 0.5 } }}
							slotProps={{
								input: {
									endAdornment: (
										<InputAdornment
											position='start'
											onClick={togglePassVisible}
											sx={{ cursor: 'pointer' }}
										>
											{passIsVisible ? <VisibleIcon /> : <InVisibleIcon />}
										</InputAdornment>
									),
								},
							}}
						/>
					)}
				/>
			</Stack>

			<Stack direction={'row'} spacing={1} alignItems={'center'} mt={1} mb={1}>
				<Controller
					control={control}
					name='remember'
					render={({ field }) => (
						<FormControlLabel
							control={<Checkbox {...field} checked={field.value || false} />}
							label='Запомнить пароль'
							sx={{
								pr: 2,
								borderRadius: 20,
								width: '100%',
								transition: 'all 0.2s ease-in-out',
								':hover': { cursor: 'pointer', background: palette.action.hover },
							}}
						/>
					)}
				/>
			</Stack>

			<Button type='submit' disabled={isLoading} variant='contained' sx={{ borderRadius: 10, marginY: 3 }}>
				Войти
			</Button>
		</Stack>
	)
}
