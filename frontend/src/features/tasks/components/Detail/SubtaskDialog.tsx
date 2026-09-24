import { useState } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'

import type { ISubtask } from '../../types/task'

interface Props {
	open: boolean
	editSubtask?: ISubtask | null
	onSubmit: (values: { title: string; description: string }) => void
	onClose: () => void
}

export const SubtaskDialog = ({ open, editSubtask, onSubmit, onClose }: Props) => {
	const isEdit = Boolean(editSubtask)
	const [title, setTitle] = useState(editSubtask?.title ?? '')
	const [description, setDescription] = useState(editSubtask?.description ?? '')

	const handleSubmit = () => {
		if (!title.trim()) return
		onSubmit({ title: title.trim(), description: description.trim() })
		onClose()
	}

	const handleClose = () => {
		setTitle('')
		setDescription('')
		onClose()
	}

	return (
		<Dialog open={open} onClose={handleClose} maxWidth='sm' fullWidth>
			<DialogTitle sx={{ fontSize: '1rem' }}>
				{isEdit ? 'Редактировать подзадачу' : 'Новая подзадача'}
			</DialogTitle>
			<DialogContent>
				<TextField
					autoFocus
					fullWidth
					label='Заголовок'
					placeholder='Краткое описание подзадачи'
					value={title}
					onChange={e => setTitle(e.target.value)}
					onKeyDown={e => {
						if (e.key === 'Enter') handleSubmit()
					}}
					sx={{ mt: 1 }}
				/>
				<TextField
					fullWidth
					label='Описание'
					multiline
					rows={3}
					value={description}
					onChange={e => setDescription(e.target.value)}
					onKeyDown={e => {
						if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSubmit()
					}}
					sx={{
						mt: 1.5,
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
					disabled={!title.trim()}
					sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
				>
					{isEdit ? 'Сохранить' : 'Добавить'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
