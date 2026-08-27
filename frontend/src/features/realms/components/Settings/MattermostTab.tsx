import { type FC } from 'react'
import { Stack, TextField, Typography, Button, CircularProgress, Alert } from '@mui/material'
import { useForm } from 'react-hook-form'
import { toast } from 'react-toastify'
import { TrashIcon, SaveIcon } from 'lucide-mui'

import type { IFetchError } from '@/app/types/error'
import type { IRealmMattermostDTO } from '@/features/realms/types/mattermost'
import {
	useGetMattermostSettingsQuery,
	useSaveMattermostSettingsMutation,
	useDeleteMattermostSettingsMutation,
} from '@/features/realms/realmsApiSlice'
import { BoxFallback } from '@/components/Fallback/BoxFallback'

type Props = {
	realmId: string
}

export const MattermostTab: FC<Props> = ({ realmId }) => {
	const { data, isFetching } = useGetMattermostSettingsQuery(realmId, { skip: !realmId })
	const [save, { isLoading: isSaving }] = useSaveMattermostSettingsMutation()
	const [remove, { isLoading: isRemoving }] = useDeleteMattermostSettingsMutation()

	const { control, handleSubmit, reset } = useForm<IRealmMattermostDTO>({
		values: data?.data
			? { botToken: data.data.botToken, channelId: data.data.channelId }
			: { botToken: '', channelId: '' },
	})

	const onSave = handleSubmit(async form => {
		try {
			await save({ realmId, dto: form }).unwrap()
			toast.success('Настройки Mattermost сохранены')
		} catch (error) {
			const err = error as IFetchError
			toast.error(err.data.message, { autoClose: false })
		}
	})

	const onDelete = async () => {
		try {
			await remove(realmId).unwrap()
			reset({ botToken: '', channelId: '' })
			toast.success('Настройки Mattermost удалены')
		} catch (error) {
			const err = error as IFetchError
			toast.error(err.data.message, { autoClose: false })
		}
	}

	if (isFetching) return <BoxFallback />

	const isConfigured = Boolean(data?.data)

	return (
		<Stack spacing={3}>
			{isConfigured && (
				<Alert severity='success' sx={{ borderRadius: '12px' }}>
					Mattermost интеграция настроена. Бот подключён и слушает ЛС.
				</Alert>
			)}

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Токен бота
				</Typography>
				<TextField
					{...control.register('botToken', { required: 'Обязательное поле' })}
					fullWidth
					type='password'
					placeholder='Введите токен бота Mattermost'
					error={Boolean(control._formState.errors.botToken)}
					helperText={control._formState.errors.botToken?.message}
				/>
			</Stack>

			<Stack>
				<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
					Channel ID
				</Typography>
				<TextField
					{...control.register('channelId')}
					fullWidth
					placeholder='ID канала для уведомлений (необязательно)'
				/>
			</Stack>

			<Stack direction='row' spacing={2} sx={{ pt: 1 }}>
				<Button
					onClick={onSave}
					variant='contained'
					disabled={isSaving || isRemoving}
					sx={{ textTransform: 'none', px: 3 }}
					startIcon={isSaving ? <CircularProgress size={16} /> : <SaveIcon sx={{ fontSize: 16 }} />}
				>
					{isConfigured ? 'Обновить' : 'Подключить'}
				</Button>

				{isConfigured && (
					<Button
						onClick={onDelete}
						variant='outlined'
						color='error'
						disabled={isSaving || isRemoving}
						sx={{ textTransform: 'none' }}
						startIcon={isRemoving ? <CircularProgress size={16} /> : <TrashIcon sx={{ fontSize: 16 }} />}
					>
						Отключить
					</Button>
				)}
			</Stack>
		</Stack>
	)
}
