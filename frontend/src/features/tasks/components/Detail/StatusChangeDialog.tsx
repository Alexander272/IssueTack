import { useState } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'

interface Props {
	open: boolean
	statusLabel: string
	onSubmit: (text: string) => void
	onCancel: () => void
}

export const StatusChangeDialog = ({ open, statusLabel, onSubmit, onCancel }: Props) => {
	const [text, setText] = useState('')

	const handleSubmit = () => {
		if (!text.trim()) return
		onSubmit(text.trim())
		setText('')
	}

	const handleClose = () => {
		setText('')
		onCancel()
	}

	return (
		<Dialog open={open} onClose={handleClose} maxWidth='sm' fullWidth>
			<DialogTitle sx={{ fontSize: '1rem' }}>Комментарий к смене статуса</DialogTitle>
			<DialogContent>
				<TextField
					autoFocus
					multiline
					rows={3}
					placeholder={`Укажите причину смены статуса на «${statusLabel}»...`}
					value={text}
					onChange={e => setText(e.target.value)}
					onKeyDown={e => {
						if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSubmit()
					}}
					fullWidth
					sx={{
						mt: 1,
						'& .MuiOutlinedInput-root': { borderRadius: '8px', fontSize: '0.8125rem' },
					}}
				/>
			</DialogContent>
			<DialogActions sx={{ px: 3, pb: 2 }}>
				<Button onClick={handleClose} sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button
					variant='contained'
					onClick={handleSubmit}
					disabled={!text.trim()}
					sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
				>
					Отправить
				</Button>
			</DialogActions>
		</Dialog>
	)
}
