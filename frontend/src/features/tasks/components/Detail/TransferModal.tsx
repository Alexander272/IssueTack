import { useMemo, useState, type FC } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	TextField,
	Typography,
} from '@mui/material'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ITask } from '../../types/task'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useTransferTaskMutation } from '../../tasksApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	task: ITask
}

export const TransferModal: FC<Props> = ({ open, onClose, task }) => {
	const [transferTask, { isLoading }] = useTransferTaskMutation()
	const { data: groupsData } = useGetAllGroupsQuery()

	const group = useMemo(
		() => groupsData?.data?.find(g => g.id === task.group?.id) ?? null,
		[groupsData, task.group?.id],
	)
	const candidates = useMemo(
		() => (group?.members ?? []).filter(m => m.id !== task.assignee?.id),
		[group, task.assignee?.id],
	)

	const [assigneeId, setAssigneeId] = useState<string | null>(null)

	const handleClose = () => {
		if (isLoading) return
		setAssigneeId(null)
		onClose()
	}

	const handleSubmit = async () => {
		if (!assigneeId) return
		try {
			await transferTask({ id: task.id, assigneeId }).unwrap()
			toast.success('Заявка передана')
			handleClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Не удалось передать заявку', { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={handleClose}
			fullWidth
			maxWidth='xs'
			slotProps={{
				paper: { sx: { borderRadius: '16px', p: 1 } },
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Передать заявку
				</Typography>
				<IconButton size='large' onClick={handleClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent>
				<Typography sx={{ fontSize: '0.8125rem', color: '#6b7280', mb: 2 }}>
					Выберите исполнителя из группы «{task.group?.name ?? '—'}», которому будет передана заявка.
				</Typography>
				<Autocomplete
					options={candidates}
					getOptionLabel={m => `${m.lastName} ${m.firstName} (${m.username})`}
					value={candidates.find(m => m.id === assigneeId) ?? null}
					onChange={(_, value) => setAssigneeId(value?.id ?? null)}
					noOptionsText='Нет доступных исполнителей'
					renderInput={params => <TextField {...params} size='small' placeholder='Исполнитель' />}
				/>
			</DialogContent>

			<DialogActions sx={{ p: 2, pt: 0 }}>
				<Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, width: '100%' }}>
					<Button
						variant='outlined'
						onClick={handleClose}
						color='inherit'
						sx={{ textTransform: 'none', color: 'text.primary' }}
					>
						Отмена
					</Button>
					<Button
						variant='outlined'
						disabled={!assigneeId || isLoading}
						onClick={handleSubmit}
						sx={{ textTransform: 'none' }}
					>
						{isLoading ? 'Передача...' : 'Передать'}
					</Button>
				</Box>
			</DialogActions>
		</Dialog>
	)
}
