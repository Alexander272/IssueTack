import { type FC } from 'react'
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, IconButton, Typography } from '@mui/material'
import { useForm } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ICategoryDTO } from '../../types/category'
import type { IGroup } from '@/features/groups/types/group'
import { useCreateCategoryMutation, useUpdateCategoryMutation } from '../../categoriesApiSlice'
import { Form } from '../Form/Form'
import { TimesIcon } from '@/components/Icons/TimesIcon'

type Props = {
	category?: ICategoryDTO
	groups: IGroup[]
	open: boolean
	onClose: () => void
}

export const CategoryDialog: FC<Props> = ({ category, groups, open, onClose }) => {
	const [create, { isLoading: isCreating }] = useCreateCategoryMutation()
	const [update, { isLoading: isUpdating }] = useUpdateCategoryMutation()

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

	const isLoading = isCreating || isUpdating

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

			<DialogActions sx={{ p: 2, gap: 1 }}>
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
					{category ? 'Сохранить' : 'Создать'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
