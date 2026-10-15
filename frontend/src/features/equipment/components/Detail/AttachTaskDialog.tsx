import { useMemo, useState, type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	List,
	ListItemButton,
	Stack,
	TextField,
	Typography,
} from '@mui/material'
import { LinkIcon, SearchIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { Loader } from '@/components/Fallback/Loader'
import { TaskStatusBadge } from '@/features/tasks/components/TaskStatusBadge'
import { useGetTasksQuery, useUpdateTaskMutation } from '@/features/tasks/tasksApiSlice'
import type { IEquipment } from '../../types/equipment'
import { EmptyState } from './EmptyState'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

// Диалог привязки существующей заявки к единице оборудования. Показываются только
// заявки без оборудования (фильтр на клиенте): привязка занятой заявки отобрала бы
// её у другой единицы, а «перевесить» заявку можно в её собственной форме правки.
// Права проверяет бэкенд (правка полей заявки — создатель/менеджер группы/админ
// области), поэтому недоступные заявки вернут отказ и он же уйдёт в тост.
export const AttachTaskDialog: FC<Props> = ({ open, onClose, unit }) => {
	const [search, setSearch] = useState('')
	const [attachingId, setAttachingId] = useState<string | null>(null)

	const { data, isLoading } = useGetTasksQuery({ search: search.trim() || undefined, limit: 30 }, { skip: !open })
	const [updateTask] = useUpdateTaskMutation()

	const tasks = useMemo(() => (data?.data ?? []).filter(task => !task.equipment), [data])

	const handleAttach = async (taskId: string) => {
		setAttachingId(taskId)
		try {
			await updateTask({ id: taskId, equipmentId: unit.id }).unwrap()
			toast.success('Заявка привязана к оборудованию')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Не удалось привязать заявку', { autoClose: false })
		} finally {
			setAttachingId(null)
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{ paper: { sx: { borderRadius: '16px', p: 1 } } }}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Привязать заявку
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Stack spacing={2}>
					<TextField
						value={search}
						onChange={e => setSearch(e.target.value)}
						size='small'
						fullWidth
						placeholder='Поиск заявки по теме или номеру'
						slotProps={{ input: { startAdornment: <SearchIcon sx={{ fontSize: 18, color: '#94a3b8', mr: 1 }} /> } }}
					/>

					{isLoading ? (
						<Box sx={{ py: 4 }}>
							<Loader />
						</Box>
					) : tasks.length === 0 ? (
						<EmptyState
							icon={LinkIcon}
							title='Заявок без оборудования нет'
							hint='Занятые заявки сначала нужно отвязать в их карточке'
						/>
					) : (
						<List dense sx={{ py: 0, maxHeight: 320, overflowY: 'auto' }}>
							{tasks.map(task => (
								<ListItemButton
									key={task.id}
									disabled={Boolean(attachingId)}
									onClick={() => handleAttach(task.id)}
									sx={{ borderRadius: '10px', gap: 1, alignItems: 'flex-start' }}
								>
									<Box sx={{ flex: 1, minWidth: 0 }}>
										<Typography sx={{ fontSize: 14, fontWeight: 500, color: '#1f2937' }} noWrap>
											{task.ticketNumber ? `№${task.ticketNumber} · ` : ''}
											{task.title}
										</Typography>
										<Typography sx={{ fontSize: 12, color: '#6b7280' }} noWrap>
											{[task.category?.name, task.site?.name].filter(Boolean).join(' · ')}
										</Typography>
									</Box>
									<TaskStatusBadge status={task.status} sx={{ px: 1, py: 0.5, flexShrink: 0 }} />
								</ListItemButton>
							))}
						</List>
					)}
				</Stack>
			</DialogContent>

			<DialogActions sx={{ p: 2 }}>
				<Button
					onClick={onClose}
					variant='outlined'
					sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
				>
					Закрыть
				</Button>
			</DialogActions>
		</Dialog>
	)
}
