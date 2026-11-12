import { useState } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	Divider,
	Stack,
	TextField,
	Typography,
} from '@mui/material'
import { CalendarClock, Send } from 'lucide-mui'
import dayjs, { type Dayjs } from 'dayjs'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'

import { DateTextField } from '@/components/DatePicker/DatePicker'
import { useCreateDueDateRequestMutation } from '../../tasksApiSlice'
import type { ITask } from '../../types/task'

type Props = {
	open: boolean
	task: ITask
	onClose: () => void
}

// DueDateRequestDialog — запрос исполнителем переноса срока заявки.
// Смысл запроса имеет смысл только при установленном сроке: поле «новый срок»
// обязательно и должно быть дальше текущего (backend валидирует оба условия).
export const DueDateRequestDialog = ({ open, task, onClose }: Props) => {
	const [customDate, setCustomDate] = useState<Dayjs | null>(null)
	const [reason, setReason] = useState('')
	const [createRequest, { isLoading }] = useCreateDueDateRequestMutation()

	const currentDueDate = task.dueDate ? dayjs(task.dueDate) : null
	const canSubmit = Boolean(customDate) && reason.trim().length > 0

	const submit = async () => {
		if (!customDate) return
		try {
			await createRequest({
				ticketId: task.id,
				proposedDueDate: customDate.endOf('day').toISOString(),
				reason: reason.trim(),
			})
			setReason('')
			setCustomDate(null)
			onClose()
		} catch {
			// handled by toast in apiSlice
		}
	}

	return (
		<Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
			<DialogTitle sx={{ fontSize: '1.125rem', fontWeight: 700 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
					<CalendarClock sx={{ fontSize: 20 }} />
					Запросить перенос срока
				</Box>
			</DialogTitle>
			<DialogContent>
				<Stack spacing={2} sx={{ pt: 1 }}>
					<Typography variant='body2' sx={{ color: 'text.secondary' }}>
						Заявка: <Box component='span' sx={{ fontWeight: 600, color: 'text.primary' }}>{task.title}</Box>
					</Typography>

					{currentDueDate && (
						<Typography variant='body2' sx={{ color: 'text.secondary' }}>
							Текущий срок: <Box component='span' sx={{ color: 'text.primary' }}>{task.dueDate}</Box>
						</Typography>
					)}

					<Box>
						<DatePicker
							value={customDate}
							onChange={setCustomDate}
							slots={{ textField: DateTextField }}
							slotProps={{ textField: { fullWidth: true, size: 'small' } }}
							disablePast
							shouldDisableDate={date =>
								currentDueDate ? date.startOf('day').isBefore(currentDueDate.startOf('day')) : false
							}
						/>
					</Box>

					<TextField
						label='Причина'
						required
						multiline
						minRows={3}
						value={reason}
						onChange={e => setReason(e.target.value)}
						sx={{ width: '100%' }}
					/>

					<Divider />
				</Stack>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose} sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button variant='contained' onClick={submit} disabled={!canSubmit || isLoading} sx={{ textTransform: 'none' }}>
					<Send sx={{ fontSize: 16, mr: 0.5 }} />
					Отправить запрос
				</Button>
			</DialogActions>
		</Dialog>
	)
}
