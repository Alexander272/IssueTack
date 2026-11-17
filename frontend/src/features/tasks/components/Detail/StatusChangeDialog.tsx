import { useState } from 'react'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers'
import dayjs, { Dayjs } from 'dayjs'

import { DateTextField } from '@/components/DatePicker/DatePicker'

interface Props {
	open: boolean
	statusLabel: string
	showDueDateField?: boolean
	currentDueDate?: string | null
	onSubmit: (text: string, dueDate?: string | null) => void
	onCancel: () => void
}

export const StatusChangeDialog = ({
	open,
	statusLabel,
	showDueDateField = false,
	currentDueDate,
	onSubmit,
	onCancel,
}: Props) => {
	const [text, setText] = useState('')
	const [submitted, setSubmitted] = useState(false)
	const [dueDateIso, setDueDateIso] = useState<string | null>(currentDueDate ?? null)

	const handleSubmit = () => {
		if (!text.trim() || submitted) return
		setSubmitted(true)
		onSubmit(text.trim(), dueDateIso)
		setText('')
	}

	const handleClose = () => {
		setText('')
		setSubmitted(false)
		setDueDateIso(currentDueDate ?? null)
		onCancel()
	}

	const handleDateChange = (newDate: Dayjs | null) => {
		if (!newDate) {
			setDueDateIso(null)
			return
		}

		// Если срок уже был задан, сохраняем точное время.
		// Если это новый срок, округляем до конца выбранного дня.
		const iso = currentDueDate ? newDate.toISOString() : newDate.endOf('day').toISOString()
		setDueDateIso(iso)
	}

	return (
		<Dialog open={open} onClose={handleClose} maxWidth='sm' fullWidth>
			<DialogTitle sx={{ fontSize: '1rem' }}>Комментарий к смене статуса</DialogTitle>
			<DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
				<TextField
					autoFocus
					multiline
					rows={3}
					placeholder={`Укажите причину смены статуса на «${statusLabel}»...`}
					value={text}
					onChange={e => {
						setSubmitted(false)
						setText(e.target.value)
					}}
					onKeyDown={e => {
						if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSubmit()
					}}
					fullWidth
					sx={{
						'& .MuiOutlinedInput-root': { borderRadius: '8px' },
					}}
				/>
				{showDueDateField && (
					<DateTimePicker
						label='Новый срок'
						value={dueDateIso ? dayjs(dueDateIso) : null}
						onChange={handleDateChange}
						minDateTime={dayjs()}
						slots={{ textField: DateTextField }}
						slotProps={{ textField: { fullWidth: true, size: 'small' } }}
					/>
				)}
			</DialogContent>
			<DialogActions sx={{ px: 3, pb: 2 }}>
				<Button onClick={handleClose} sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button
					variant='contained'
					onClick={handleSubmit}
					disabled={!text.trim() || submitted}
					sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
				>
					Отправить
				</Button>
			</DialogActions>
		</Dialog>
	)
}
