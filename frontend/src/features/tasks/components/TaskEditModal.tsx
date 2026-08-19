import { type FC } from 'react'
import { Dialog, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import { XIcon } from 'lucide-mui'

import type { ITask } from '../types/task'
import { TaskEditForm } from './TaskEditForm'

type Props = {
	open: boolean
	onClose: () => void
	task: ITask
}

export const TaskEditModal: FC<Props> = ({ open, onClose, task }) => {
	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='md'
			slotProps={{
				paper: { sx: { borderRadius: '16px', p: 1 } },
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Редактирование заявки
				</Typography>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent>
				<TaskEditForm task={task} embedded onSuccess={onClose} onCancel={onClose} />
			</DialogContent>
		</Dialog>
	)
}
