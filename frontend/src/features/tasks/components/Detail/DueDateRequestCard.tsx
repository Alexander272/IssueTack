import { useState } from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField, Typography } from '@mui/material'
import { Check, X } from 'lucide-mui'
import dayjs from 'dayjs'

import { useDecideDueDateRequestMutation } from '../../tasksApiSlice'
import type { IDueDateRequest, ITask } from '../../types/task'

type Props = {
	task: ITask
	request: IDueDateRequest | null
}

// DueDateRequestCard — карточка pending-запроса переноса срока для менеджера
// (менеджера группы или начальника области). Утверждение применяет новый срок,
// отклонение — только откатывает запрос (комментарий обязателен при отклонении).
export const DueDateRequestCard = ({ task, request }: Props) => {
	const [rejectOpen, setRejectOpen] = useState(false)
	const [decisionComment, setDecisionComment] = useState('')
	const [decide, { isLoading }] = useDecideDueDateRequestMutation()

	if (!request) {
		return null
	}

	const proposed = request.proposedDueDate ? dayjs(request.proposedDueDate) : null
	const current = task.dueDate ? dayjs(task.dueDate) : null

	const approve = async () => {
		try {
			await decide({ ticketId: task.id, requestId: request.id, approved: true })
		} catch {
			// handled by toast in apiSlice
		}
	}

	const reject = async () => {
		if (!decisionComment.trim()) return
		try {
			await decide({
				ticketId: task.id,
				requestId: request.id,
				approved: false,
				decisionComment: decisionComment.trim(),
			})
			setDecisionComment('')
			setRejectOpen(false)
		} catch {
			// handled by toast in apiSlice
		}
	}

	return (
		<Box
			sx={{
				display: 'flex',
				flexWrap: 'wrap',
				alignItems: 'center',
				gap: 1.5,
				p: 2,
				borderRadius: '12px',
				border: '1px solid #fde68a',
				bgcolor: '#fffbeb',
			}}
		>
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
				<Typography sx={{ fontSize: '0.8125rem', fontWeight: 700, color: '#92400e' }}>
					Запрос переноса срока
				</Typography>
			</Box>
			{task.assignee && (
				<Typography sx={{ fontSize: '0.8125rem', color: '#78350f' }}>
					Исполнитель: {task.assignee.firstName} {task.assignee.lastName}
				</Typography>
			)}
			{current && (
				<Typography sx={{ fontSize: '0.8125rem', color: '#78350f' }}>
					Срок: {current.format('DD.MM.YYYY')} →{' '}
					{proposed ? proposed.format('DD.MM.YYYY') : '—'}
				</Typography>
			)}
			{request.reason && (
				<Typography sx={{ fontSize: '0.8125rem', color: '#78350f', fontStyle: 'italic' }}>
					{request.reason}
				</Typography>
			)}

			<Box sx={{ display: 'flex', gap: 1, ml: 'auto' }}>
				<Button
					variant='contained'
					color='success'
					disabled={isLoading}
					onClick={approve}
					startIcon={<Check sx={{ fontSize: 16 }} />}
					sx={{ textTransform: 'none' }}
				>
					Утвердить
				</Button>
				<Button
					variant='outlined'
					color='error'
					disabled={isLoading}
					onClick={() => setRejectOpen(true)}
					startIcon={<X sx={{ fontSize: 16 }} />}
					sx={{ textTransform: 'none' }}
				>
					Отклонить
				</Button>
			</Box>

			<Dialog open={rejectOpen} onClose={() => setRejectOpen(false)} maxWidth='xs' fullWidth>
				<DialogTitle>Отклонить запрос</DialogTitle>
				<DialogContent>
					<TextField
						autoFocus
						fullWidth
						label='Причина отклонения'
						required
						multiline
						minRows={2}
						value={decisionComment}
						onChange={e => setDecisionComment(e.target.value)}
					/>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => setRejectOpen(false)} sx={{ textTransform: 'none' }}>
						Отмена
					</Button>
					<Button
						variant='contained'
						color='error'
						onClick={reject}
						disabled={!decisionComment.trim() || isLoading}
						sx={{ textTransform: 'none' }}
					>
						Отклонить
					</Button>
				</DialogActions>
			</Dialog>
		</Box>
	)
}
