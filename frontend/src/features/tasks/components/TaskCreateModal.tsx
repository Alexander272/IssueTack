import { type FC } from 'react'
import { Dialog, DialogContent, DialogTitle, IconButton, Typography, useMediaQuery, useTheme } from '@mui/material'

import { TaskCreateForm } from './TaskCreateForm'
import { XIcon } from 'lucide-mui'

type Props = {
	open: boolean
	onClose: () => void
}

export const TaskCreateModal: FC<Props> = ({ open, onClose }) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='md'
			slotProps={{
				paper: { sx: { borderRadius: { xs: 0, sm: '16px' }, p: { xs: 0, sm: 1 } } },
			}}
		>
			<DialogTitle
				sx={{ m: 0, p: { xs: 1.5, sm: 2 }, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
			>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Создание заявки
				</Typography>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent sx={{ p: { xs: 1, sm: 2.5 } }}>
				<TaskCreateForm embedded onSuccess={onClose} onCancel={onClose} />
			</DialogContent>
		</Dialog>
	)
}
