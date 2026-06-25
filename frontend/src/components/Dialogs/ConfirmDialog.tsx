import { type FC } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, Typography } from '@mui/material'

type Props = {
	open: boolean
	title: string
	message: string
	confirmLabel?: string
	confirmColor?: 'primary' | 'error' | 'warning'
	onConfirm: () => void
	onCancel: () => void
	loading?: boolean
}

export const ConfirmDialog: FC<Props> = ({
	open,
	title,
	message,
	confirmLabel = 'Подтвердить',
	confirmColor = 'primary',
	onConfirm,
	onCancel,
	loading = false,
}) => {
	return (
		<Dialog
			open={open}
			onClose={onCancel}
			fullWidth
			maxWidth='xs'
			slotProps={{
				paper: { sx: { borderRadius: '12px', p: 1 } },
			}}
		>
			<DialogTitle sx={{ fontWeight: 'bold', pb: 1 }}>{title}</DialogTitle>
			<DialogContent>
				<Typography color='text.secondary'>{message}</Typography>
			</DialogContent>
			<DialogActions sx={{ p: 2, gap: 1 }}>
				<Button
					onClick={onCancel}
					variant='outlined'
					sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
				>
					Отмена
				</Button>
				<Button
					onClick={onConfirm}
					variant='contained'
					color={confirmColor}
					disabled={loading}
					sx={{ textTransform: 'none' }}
				>
					{loading ? 'Удаление...' : confirmLabel}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
