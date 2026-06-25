import { useState, type FC } from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IUserData } from '@/features/user/types/user'
import type { IGroupDTO } from '../../types/group'
import { useCreateGroupMutation, useUpdateGroupMutation, useDeleteGroupMutation } from '../../groupsApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { GroupForm } from '../Form/GroupForm'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	group?: IGroupDTO
	users: IUserData[]
	open: boolean
	onClose: () => void
}

export const GroupDialog: FC<Props> = ({ group, users, open, onClose }) => {
	const [deleteOpen, setDeleteOpen] = useState(false)

	const [create, { isLoading: isCreating }] = useCreateGroupMutation()
	const [update, { isLoading: isUpdating }] = useUpdateGroupMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteGroupMutation()

	const { control, handleSubmit } = useForm<IGroupDTO>({
		values: group ?? {
			id: undefined,
			name: '',
			description: '',
			managerId: null,
			defaultAssigneeId: null,
			memberIds: [],
		},
	})

	const memberIds = useWatch({ control, name: 'memberIds' })
	const availableUsers = users.filter(u => memberIds.includes(u.id))

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
		if (!group?.id) return
		try {
			await remove(group.id).unwrap()
			toast.success('Группа удалена')
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
			maxWidth='md'
			slotProps={{
				paper: {
					sx: { borderRadius: '16px', p: 1 },
				},
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					{group?.id ? 'Редактировать группу' : 'Создать группу'}
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<GroupForm control={control} users={users} availableUsers={availableUsers} />
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1, justifyContent: group?.id ? 'space-between' : 'flex-end' }}>
				{group?.id && (
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
						{group?.id ? 'Сохранить' : 'Создать'}
					</Button>
				</Box>
			</DialogActions>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление группы'
				message={`Вы уверены, что хотите удалить группу «${group?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</Dialog>
	)
}
