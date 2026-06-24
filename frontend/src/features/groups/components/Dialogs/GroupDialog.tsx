import { type FC } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IUserData } from '@/features/user/types/user'
import type { IGroupDTO } from '../../types/group'
import { useCreateGroupMutation, useUpdateGroupMutation } from '../../groupsApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { GroupForm } from '../Form/GroupForm'

type Props = {
	group?: IGroupDTO
	users: IUserData[]
	open: boolean
	onClose: () => void
}

export const GroupDialog: FC<Props> = ({ group, users, open, onClose }) => {
	const [create, { isLoading: isCreating }] = useCreateGroupMutation()
	const [update, { isLoading: isUpdating }] = useUpdateGroupMutation()

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
					{group?.id ? 'Редактировать группу' : 'Создать группу'}
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<GroupForm control={control} users={users} availableUsers={availableUsers} />
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
					{group?.id ? 'Сохранить' : 'Создать'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
