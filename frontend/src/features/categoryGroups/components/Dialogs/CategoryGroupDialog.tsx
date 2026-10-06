import { useState, type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogTitle,
	DialogContent,
	DialogActions,
	IconButton,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'
import { XIcon } from 'lucide-mui'

import type { IFetchError } from '@/app/types/error'
import type { ICategoryGroupDTO } from '../../types/categoryGroup'
import {
	useCreateCategoryGroupMutation,
	useUpdateCategoryGroupMutation,
	useDeleteCategoryGroupMutation,
} from '../../categoryGroupsApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	categoryGroup?: ICategoryGroupDTO
	open: boolean
	onClose: () => void
}

export const CategoryGroupDialog: FC<Props> = ({ categoryGroup, open, onClose }) => {
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateCategoryGroupMutation()
	const [update, { isLoading: isUpdating }] = useUpdateCategoryGroupMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteCategoryGroupMutation()

	const { control, handleSubmit, formState } = useForm<ICategoryGroupDTO>({
		values: categoryGroup ?? { id: null, name: '', description: '', sortOrder: 0 },
		mode: 'onTouched',
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async form => {
		try {
			if (form.id) {
				await update(form).unwrap()
				toast.success('Раздел обновлён')
			} else {
				await create(form).unwrap()
				toast.success('Раздел создан')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!categoryGroup?.id) return
		try {
			await remove(categoryGroup.id).unwrap()
			toast.success('Раздел удалён')
			setDeleteOpen(false)
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{
				paper: {
					sx: { borderRadius: '16px', p: 1 },
				},
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					{categoryGroup ? 'Редактировать раздел' : 'Создать раздел'}
				</Typography>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
							Название{' '}
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
									value={field.value ?? ''}
									error={Boolean(fieldState.error)}
									helperText={fieldState.error?.message}
								/>
							)}
						/>
					</Box>
					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
							Описание
						</Typography>
						<Controller
							control={control}
							name='description'
							render={({ field }) => (
								<TextField {...field} fullWidth multiline minRows={3} value={field.value ?? ''} />
							)}
						/>
					</Box>
					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
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
									value={field.value ?? 0}
									onChange={e => field.onChange(Number(e.target.value) || 0)}
									helperText='Чем меньше, тем выше раздел в списках'
								/>
							)}
						/>
					</Box>
				</Box>
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1, justifyContent: categoryGroup?.id ? 'space-between' : 'flex-end' }}>
				{categoryGroup?.id && (
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
					<Button
						onClick={onClose}
						variant='outlined'
						sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
					>
						Отмена
					</Button>
					<Button
						onClick={saveHandler}
						variant='contained'
						disabled={isLoading || !formState.isValid || (Boolean(categoryGroup?.id) && !formState.isDirty)}
						sx={{ textTransform: 'none', px: 3 }}
					>
						{categoryGroup?.id ? 'Сохранить' : 'Создать'}
					</Button>
				</Box>
			</DialogActions>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление раздела'
				message={`Вы уверены, что хотите удалить раздел «${categoryGroup?.name}»? Раздел можно удалить, только если в нём нет категорий.`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</Dialog>
	)
}
