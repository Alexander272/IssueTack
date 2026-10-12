import { useState, type FC } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import {
	Box,
	Button,
	Checkbox,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControlLabel,
	FormHelperText,
	IconButton,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentType, IEquipmentTypeDTO } from '../types/equipment'
import {
	useCreateEquipmentTypeMutation,
	useUpdateEquipmentTypeMutation,
	useDeleteEquipmentTypeMutation,
} from '../equipmentApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	open: boolean
	onClose: () => void
	type?: IEquipmentType | null
	setType?: Dispatch<SetStateAction<IEquipmentTypeDTO | null>>
}

const fieldLabelSx = { fontWeight: 600, mb: 0.5, display: 'block' }

export const EquipmentTypeDialog: FC<Props> = ({ open, onClose, type, setType }) => {
	const isEdit = Boolean(type?.id)
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateEquipmentTypeMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentTypeMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentTypeMutation()

	const { control, handleSubmit, formState } = useForm<IEquipmentTypeDTO>({
		values: type
			? { id: type.id, name: type.name, isComponent: type.isComponent, sortOrder: type.sortOrder }
			: { id: undefined, name: '', isComponent: false, sortOrder: 0 },
		mode: 'onTouched',
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const isComponent = useWatch({ control, name: 'isComponent' })

	const saveHandler = handleSubmit(async form => {
		try {
			if (form.id) {
				await update(form).unwrap()
				toast.success('Тип обновлён')
			} else {
				await create(form).unwrap()
				toast.success('Тип создан')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при сохранении', { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!type?.id) return
		try {
			await remove(type.id).unwrap()
			toast.success('Тип удалён')
			setDeleteOpen(false)
			setType?.(null)
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
				<DialogTitle
					sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
				>
					<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
						{isEdit ? 'Редактирование типа' : 'Создание типа'}
					</Typography>
					<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
						<XIcon sx={{ fontSize: 16 }} />
					</IconButton>
				</DialogTitle>

				<form onSubmit={saveHandler}>
					<DialogContent
						dividers
						sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
					>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Название типа{' '}
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
											size='small'
											placeholder='Например: Ноутбук, Память (RAM), Принтер'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Порядок сортировки
								</Typography>
								<Controller
									control={control}
									name='sortOrder'
									render={({ field }) => (
										<TextField
											{...field}
											type='number'
											fullWidth
											size='small'
											onChange={e => field.onChange(Number(e.target.value))}
										/>
									)}
								/>
							</Box>

							<Box>
								<Controller
									control={control}
									name='isComponent'
									render={({ field }) => (
										<FormControlLabel
											control={
												<Checkbox
													checked={field.value}
													onChange={e => field.onChange(e.target.checked)}
												/>
											}
											label='Компонент (входит в состав комплекта)'
										/>
									)}
								/>
								{isComponent && (
									<FormHelperText>
										Компоненты — запчасти и периферия: RAM, SSD, мышь. Их нельзя выдать отдельно.
									</FormHelperText>
								)}
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
								sx={{
									textTransform: 'none',
									borderColor: '#fecaca',
									'&:hover': { borderColor: '#fca5a5' },
								}}
							>
								Удалить
							</Button>
						)}
						<Box sx={{ display: 'flex', gap: 1 }}>
							<Button
								type='button'
								onClick={onClose}
								variant='outlined'
								sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
							>
								Отмена
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={isLoading || !formState.isValid || (isEdit && !formState.isDirty)}
								sx={{ textTransform: 'none', px: 3 }}
							>
								{isEdit ? 'Сохранить' : 'Создать'}
							</Button>
						</Box>
					</DialogActions>
				</form>
			</Dialog>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление типа'
				message={`Вы уверены, что хотите удалить тип «${type?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</>
	)
}