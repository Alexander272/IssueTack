import { useState, type FC } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormHelperText,
	IconButton,
	MenuItem,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentType, IEquipmentTypeDTO } from '../../types/equipment'
import { EQUIPMENT_TYPE_KIND_OPTIONS } from '../../types/equipment'
import {
	useCreateEquipmentTypeMutation,
	useUpdateEquipmentTypeMutation,
	useDeleteEquipmentTypeMutation,
} from '../../equipmentApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	open: boolean
	onClose: () => void
	type?: IEquipmentType | null
	setType?: Dispatch<SetStateAction<IEquipmentTypeDTO | null>>
}

const CATEGORIES = EQUIPMENT_TYPE_KIND_OPTIONS

export const EquipmentTypeDialog: FC<Props> = ({ open, onClose, type, setType }) => {
	const theme = useTheme()
	// На телефоне диалог типа — во весь экран.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const isEdit = Boolean(type?.id)
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateEquipmentTypeMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentTypeMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentTypeMutation()

	const { control, handleSubmit, formState } = useForm<IEquipmentTypeDTO>({
		values: type
			? {
				id: type.id,
				name: type.name,
				kind: type.kind,
				sortOrder: type.sortOrder,
			}
			: { id: undefined, name: '', kind: 'device', sortOrder: 0 },
		mode: 'onTouched',
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async form => {
		const dto: IEquipmentTypeDTO = {
			id: form.id,
			name: form.name,
			kind: form.kind,
			sortOrder: form.sortOrder ?? 0,
		}
		try {
			if (dto.id) {
				await update(dto).unwrap()
				toast.success('Тип обновлён')
			} else {
				await create(dto).unwrap()
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
				fullScreen={isMobile}
				maxWidth='sm'
				slotProps={{
					paper: { sx: { borderRadius: { xs: 0, sm: '12px' } } },
				}}
			>
				<DialogTitle sx={{ m: 0, px: 3, py: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
					<Typography variant='h6' sx={{ fontWeight: 700 }}>
						{isEdit ? 'Редактирование типа' : 'Новый тип оборудования'}
					</Typography>
					<IconButton onClick={onClose} size='small'>
						<XIcon sx={{ fontSize: 18 }} />
					</IconButton>
				</DialogTitle>

				<form onSubmit={saveHandler}>
					<DialogContent dividers sx={{ px: 3, py: 2.5 }}>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}>
									Название *
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
											placeholder='Например: Веб-камера'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
											sx={{ '& .MuiOutlinedInput-root': { borderRadius: '8px' } }}
										/>
									)}
								/>
							</Box>
							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}>
									Категория *
								</Typography>
								<Controller
									control={control}
									name='kind'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											select
											fullWidth
											size='small'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
											sx={{ '& .MuiOutlinedInput-root': { borderRadius: '8px' } }}
										>
											{CATEGORIES.map(c => (
												<MenuItem key={c.value} value={c.value}>
													{c.label}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
								<FormHelperText sx={{ mt: 0.5 }}>
									Комплектующие и расходники можно включить в физический состав; устройства и периферию — нет
								</FormHelperText>
							</Box>
						</Box>
					</DialogContent>

					<DialogActions sx={{ px: 3, py: 2, gap: 1, justifyContent: isEdit ? 'space-between' : 'flex-end' }}>
						{isEdit && (
							<Button
								type='button'
								variant='outlined'
								color='error'
								onClick={() => setDeleteOpen(true)}
								disabled={isDeleting}
								sx={{ textTransform: 'none', borderRadius: '8px' }}
							>
								Удалить
							</Button>
						)}
						<Box sx={{ display: 'flex', gap: 1 }}>
							<Button
								type='button'
								onClick={onClose}
								variant='outlined'
								sx={{ textTransform: 'none', color: '#374151', borderColor: '#d1d5db', borderRadius: '8px' }}
							>
								Отмена
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={isLoading || !formState.isValid || (isEdit && !formState.isDirty)}
								sx={{ textTransform: 'none', px: 3, borderRadius: '8px', bgcolor: '#0432a5', '&:hover': { bgcolor: '#002d91' } }}
							>
								{isEdit ? 'Сохранить' : 'Создать'}
							</Button>
						</Box>
					</DialogActions>
				</form>
			</Dialog>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление типа оборудования'
				message={`Удалить тип «${type?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
			/>
		</>
	)
}
