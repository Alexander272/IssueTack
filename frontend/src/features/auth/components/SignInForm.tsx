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
import { Controller, useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ISignIn } from '../types/auth'
import { useAppDispatch, useAppSelector } from '@/hooks/redux'
import { setUser } from '@/features/user/userSlice'
import { getRealm, setRealm } from '@/features/realms/realmSlice'
import { EyeIcon, EyeOffIcon, TrashIcon } from 'lucide-mui'
import {
	clearRememberedCredentials,
	loadRememberedCredentials,
	saveRememberedCredentials,
} from '../storage/rememberedCredentials'
import { useSignInMutation } from '../authApiSlice'

const defaultValues: ISignIn = { username: '', password: '', remember: false }

export const SignInForm = () => {
	const [passIsVisible, setPassIsVisible] = useState(false)
	const [initialCredentials] = useState(() => loadRememberedCredentials())
	const [hasStoredPassword, setHasStoredPassword] = useState(() => !!initialCredentials)
	const { palette } = useTheme()

	const dispatch = useAppDispatch()
	const realm = useAppSelector(getRealm)

	const {
		control,
		handleSubmit,
		reset,
		getValues,
		formState: { errors },
	} = useForm<ISignIn>({
		defaultValues: initialCredentials
			? {
					username: initialCredentials.username,
					password: initialCredentials.password,
					remember: true,
				}
			: defaultValues,
	})

	const [signIn, { isLoading }] = useSignInMutation()

	const remember = useWatch({ control, name: 'remember' })

	const togglePassVisible = () => setPassIsVisible(prev => !prev)

	const forgetHandler = () => {
		clearRememberedCredentials()
		setHasStoredPassword(false)
		reset({ ...defaultValues, username: getValues('username') })
	}

	const signInHandler = async (data: ISignIn) => {
		if (!data.remember) clearRememberedCredentials()

		const request = signIn(data)
		try {
			const payload = await request.unwrap()
			if (data.remember) saveRememberedCredentials(data.username, data.password)
			dispatch(setUser(payload.data))
			if (!realm && payload.data.realms.length > 0 && payload.data.realms[0].realm) {
				dispatch(setRealm(payload.data.realms[0].realm))
			}
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		} finally {
			request.reset()
		}
	}

	return (
		<Stack component='form' onSubmit={handleSubmit(signInHandler)} sx={{ position: 'relative' }}>
			{isLoading ? <LinearProgress sx={{ position: 'absolute', bottom: -20, left: 0, right: 0 }} /> : null}

			<Typography
				variant='h2'
				align='center'
				sx={{
					fontSize: '1.5rem',
					color: palette.primary.main,
					paddingBottom: 1.25,
					mb: 1.25,
					fontWeight: 'bold',
					lineHeight: 'inherit',
					borderBottom: '1px solid #e5e4e9',
					letterSpacing: '1.2px',
				}}
			>
				Вход
			</Typography>

			<Stack spacing={2} sx={{ mt: 2 }}>
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
											{passIsVisible ? <EyeIcon /> : <EyeOffIcon />}
										</InputAdornment>
									),
								},
							}}
						/>
					)}
				/>
			</Stack>

			<Stack sx={{ mt: 1, mb: 1 }}>
				<Stack direction={'row'} spacing={1} sx={{ alignItems: 'center' }}>
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
									flex: 1,
									transition: 'all 0.2s ease-in-out',
									':hover': { cursor: 'pointer', background: palette.action.hover },
								}}
							/>
						)}
					/>

					{hasStoredPassword ? (
						<Button
							type='button'
							size='small'
							onClick={forgetHandler}
							startIcon={<TrashIcon sx={{ fontSize: 16 }} />}
							sx={{ color: palette.text.secondary, whiteSpace: 'nowrap', flexShrink: 0 }}
						>
							Забыть
						</Button>
					) : null}
				</Stack>

				{remember ? (
					<Typography variant='caption' sx={{ display: 'block', color: palette.text.secondary, mt: 0.5 }}>
						Пароль запомнится в этом браузере — не включайте на чужих машинах. При выходе из аккаунта он
						стирается.
					</Typography>
				) : null}
			</Stack>

			<Button type='submit' disabled={isLoading} variant='contained' sx={{ borderRadius: 10, marginY: 3 }}>
				Войти
			</Button>
		</Stack>
	)
}
