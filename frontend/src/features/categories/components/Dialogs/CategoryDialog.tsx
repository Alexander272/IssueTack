import { useState, type FC } from 'react'
import { Box, Dialog, DialogTitle, DialogContent, DialogActions, Button, IconButton, Typography } from '@mui/material'
import { useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ICategoryDTO } from '../../types/category'
import type { IGroup } from '@/features/groups/types/group'
import { useCreateCategoryMutation, useUpdateCategoryMutation, useDeleteCategoryMutation } from '../../categoriesApiSlice'
import { Form } from '../Form/Form'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	category?: ICategoryDTO
	groups: IGroup[]
	open: boolean
	onClose: () => void
}

export const CategoryDialog: FC<Props> = ({ category, groups, open, onClose }) => {
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateCategoryMutation()
	const [update, { isLoading: isUpdating }] = useUpdateCategoryMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteCategoryMutation()

	const { control, handleSubmit } = useForm<ICategoryDTO>({
		values: category ?? {
			id: null,
			name: '',
			description: '',
			groupId: '',
			priority: 'medium',
			isActive: true,
		},
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async form => {
		try {
			if (form.id) {
				await update(form).unwrap()
			} else {
				await create(form).unwrap()
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!category?.id) return
		try {
			await remove(category.id).unwrap()
			toast.success('Категория удалена')
			setDeleteOpen(false)
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data.message, { autoClose: false })
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
					{category ? 'Редактировать категорию' : 'Создать категорию'}
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Form control={control} groups={groups} />
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1, justifyContent: category?.id ? 'space-between' : 'flex-end' }}>
				{category?.id && (
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
						disabled={isLoading}
						sx={{ textTransform: 'none', px: 3 }}
					>
						{category?.id ? 'Сохранить' : 'Создать'}
					</Button>
				</Box>
			</DialogActions>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление категории'
				message={`Вы уверены, что хотите удалить категорию «${category?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</Dialog>
	)
}
