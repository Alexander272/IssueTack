import { useState, type FC } from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, TextField, Typography } from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ISiteDTO } from '../types/site'
import { useCreateSiteMutation, useUpdateSiteMutation, useDeleteSiteMutation } from '../sitesApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	site?: ISiteDTO | null
	open: boolean
	onClose: () => void
}

const fieldLabelSx = { fontWeight: 600, mb: 0.5, display: 'block' }

export const SiteDialog: FC<Props> = ({ site, open, onClose }) => {
	const isEdit = Boolean(site?.id)
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateSiteMutation()
	const [update, { isLoading: isUpdating }] = useUpdateSiteMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteSiteMutation()

	const { control, handleSubmit } = useForm<ISiteDTO>({
		values: site ?? { id: null, name: '', address: '' },
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async form => {
		try {
			if (form.id) {
				await update(form).unwrap()
				toast.success('Площадка обновлена')
			} else {
				await create(form).unwrap()
				toast.success('Площадка создана')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при сохранении', { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!site?.id) return
		try {
			await remove(site.id).unwrap()
			toast.success('Площадка удалена')
			setDeleteOpen(false)
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при удалении', { autoClose: false })
		}
	}

	return (
		<>
			<Dialog
				open={open}
				onClose={onClose}
				fullWidth
				maxWidth='sm'
				slotProps={{
					paper: { sx: { borderRadius: '16px', p: 1 } },
				}}
			>
				<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
					<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
						{isEdit ? 'Редактирование площадки' : 'Создание площадки'}
					</Typography>
					<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
						<TimesIcon fontSize={16} />
					</IconButton>
				</DialogTitle>

				<form onSubmit={saveHandler}>
					<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Название площадки{' '}
									<Typography component='span' color='error'>*</Typography>
								</Typography>
								<Controller
									control={control}
									name='name'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField {...field} fullWidth size='small' error={Boolean(fieldState.error)} helperText={fieldState.error?.message} />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Адрес{' '}
									<Typography component='span' color='error'>*</Typography>
								</Typography>
								<Controller
									control={control}
									name='address'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField {...field} fullWidth size='small' multiline rows={3} error={Boolean(fieldState.error)} helperText={fieldState.error?.message} />
									)}
								/>
							</Box>
						</Box>
					</DialogContent>

					<DialogActions sx={{ p: 2, gap: 1, justifyContent: isEdit ? 'space-between' : 'flex-end' }}>
						{isEdit && (
							<Button
								type='button'
								variant='outlined'
								color='error'
								onClick={() => setDeleteOpen(true)}
								disabled={isDeleting}
								sx={{ textTransform: 'none', borderColor: '#fecaca', '&:hover': { borderColor: '#fca5a5' } }}
							>
								Удалить
							</Button>
						)}
						<Box sx={{ display: 'flex', gap: 1 }}>
							<Button type='button' onClick={onClose} variant='outlined' sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}>
								Отмена
							</Button>
							<Button type='submit' variant='contained' disabled={isLoading} sx={{ textTransform: 'none', px: 3 }}>
								{isEdit ? 'Сохранить' : 'Создать'}
							</Button>
						</Box>
					</DialogActions>
				</form>
			</Dialog>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление площадки'
				message={`Вы уверены, что хотите удалить площадку «${site?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</>
	)
}
