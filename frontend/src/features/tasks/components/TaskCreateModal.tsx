import { type FC } from 'react'
import { Dialog, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'

import { TaskCreateForm } from './TaskCreateForm'
import { XIcon } from 'lucide-mui'

type Props = {
	open: boolean
	onClose: () => void
}

export const TaskCreateModal: FC<Props> = ({ open, onClose }) => {
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
					Создание заявки
				</Typography>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent>
				<TaskCreateForm embedded onSuccess={onClose} onCancel={onClose} />
			</DialogContent>
		</Dialog>
	)
}
