import { useCallback, useMemo, useState } from 'react'
import {
	Alert,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	InputAdornment,
	TextField,
	Typography,
} from '@mui/material'
import { CopyPlus, Info, Search, X } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { ITask, ITaskFilter } from '../../../types/task'
import { useGetTicketDuplicatesQuery, useGetTasksQuery, useLinkDuplicateMutation } from '../../../tasksApiSlice'
import { useDebounce } from '@/hooks/useDebounce'
import { linkSuccessMessage, normalizeQuery, ticketNumber } from './utils'
import { CandidateList, FieldLabel, PickedPrimaryCard } from './parts'

interface Props {
	open: boolean
	/** Заявка, которую отмечаем дублем. */
	task: ITask
	onClose: () => void
}

// Стабильная ссылка для пустого результата — иначе memo списка ломается на каждом рендере.
const EMPTY: ITask[] = []

export const LinkDuplicateDialog = ({ open, task, onClose }: Props) => {
	const [query, setQuery] = useState('')
	const [picked, setPicked] = useState<ITask | null>(null)
	const debouncedQuery = useDebounce(normalizeQuery(query), 300)

	const [linkDuplicate, { isLoading: linking }] = useLinkDuplicateMutation()

	// Запросы живут только пока диалог открыт; поиск не нужен и когда основная уже выбрана.
	const { data: ownDuplicates } = useGetTicketDuplicatesQuery(task.id, { skip: !open })

	const candidatesFilter = useMemo<ITaskFilter>(() => {
		if (!debouncedQuery) return {}
		// readable — тот же скоуп, что у поиска заявки по номеру в боте: автор,
		// заказчик, исполнитель, наблюдатель или группа. Default-скоуп списка
		// отсекал бы заявку, созданную пользователем, который не состоит в её группе.
		return {
			mode: 'readable',
			...(/^\d+$/.test(debouncedQuery)
				? { number: Number(debouncedQuery) }
				: { search: debouncedQuery }),
			allStatuses: true,
			limit: 20,
		}
	}, [debouncedQuery])

	const { data: candidatesData, isFetching } = useGetTasksQuery(candidatesFilter, {
		skip: !open || !debouncedQuery || Boolean(picked),
	})
	const candidates = candidatesData?.data ?? EMPTY

	const reset = useCallback(() => {
		setQuery('')
		setPicked(null)
	}, [])

	const changePrimary = useCallback(() => {
		setPicked(null)
		setQuery('')
	}, [])

	const handleConfirm = async () => {
		if (!picked || linking) return
		try {
			await linkDuplicate({ id: task.id, primaryId: picked.id }).unwrap()
			toast.success(linkSuccessMessage(task, picked))
			onClose()
		} catch {
			// ошибка показана тостом в tasksApiSlice
		}
	}

	const hasOwnDuplicates = (ownDuplicates?.data?.length ?? 0) > 0
	const primaryNumber = picked?.duplicateOfNumber ?? '—'

	return (
		<Dialog
			open={open}
			onClose={onClose}
			aria-labelledby='mark-duplicate-title'
			fullWidth
			maxWidth='sm'
			// Сброс после анимации закрытия: контент не «мигает», следующее открытие — с чистого листа.
			// Для MUI v5: TransitionProps={{ onExited: reset }}
			slotProps={{ transition: { onExited: reset } }}
		>
			<DialogTitle
				id='mark-duplicate-title'
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: 1.25,
					px: 2.5,
					py: 2,
					borderBottom: '1px solid',
					borderColor: 'divider',
				}}
			>
				<CopyPlus sx={{ fontSize: 18, color: 'text.secondary' }} />
				<Typography component='span' variant='subtitle1' sx={{ flex: 1, fontWeight: 700 }}>
					{task.duplicateOf ? 'Изменить основную' : 'Отметить дублем'}
				</Typography>
				<IconButton aria-label='Закрыть' onClick={onClose}>
					<X sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent sx={{ p: 2.5, display: 'grid', gap: 2, alignContent: 'start', overflowWrap: 'anywhere' }}>
				{/* Какая заявка помечается — видно на обоих шагах. */}
				<Box
					sx={{
						display: 'flex',
						gap: 0.75,
						alignItems: 'baseline',
						p: 1.25,
						border: '1px solid',
						borderColor: 'divider',
						borderRadius: '8px',
						bgcolor: 'action.hover',
						mt: 2,
					}}
				>
					<Typography component='span' variant='body2' color='primary' sx={{ flex: 'none', fontWeight: 700 }}>
						№{ticketNumber(task)}
					</Typography>
					<Typography component='span' variant='body2' sx={{ minWidth: 0 }}>
						{task.title}
					</Typography>
				</Box>

				<Box>
					<FieldLabel htmlFor='duplicate-primary-search'>Основная заявка</FieldLabel>
					{!picked ? (
						<>
							<TextField
								id='duplicate-primary-search'
								autoFocus
								fullWidth
								size='small'
								placeholder='Номер или часть названия'
								autoComplete='off'
								value={query}
								onChange={e => setQuery(e.target.value)}
								slotProps={{
									input: {
										startAdornment: (
											<InputAdornment position='start'>
												<Search sx={{ fontSize: 16, color: 'text.secondary' }} />
											</InputAdornment>
										),
									},
								}}
							/>
							{debouncedQuery && (
								<CandidateList
									candidates={candidates}
									isFetching={isFetching}
									task={task}
									onPick={setPicked}
								/>
							)}
						</>
					) : (
						<PickedPrimaryCard picked={picked} onChange={changePrimary} />
					)}
				</Box>

				{picked?.duplicateOf && (
					<Alert
						severity='warning'
						icon={<Info sx={{ fontSize: 16 }} />}
						sx={{ alignItems: 'flex-start', fontSize: 13 }}
					>
						№{ticketNumber(picked)} сама является дублем заявки №{primaryNumber}. Заявка №
						{ticketNumber(task)} будет привязана к основной №{primaryNumber} (через №{ticketNumber(picked)}
						).
					</Alert>
				)}

				{picked && hasOwnDuplicates && (
					<Alert
						severity='warning'
						icon={<Info sx={{ fontSize: 16 }} />}
						sx={{ alignItems: 'flex-start', fontSize: 13 }}
					>
						У заявки №{ticketNumber(task)} уже есть свои дубли. После привязки они будут перепривязаны к
						новой основной заявке.
					</Alert>
				)}

				{picked && (
					<Box>
						<Typography sx={{ mb: 1, fontSize: 12, fontWeight: 600, color: 'text.secondary' }}>
							Что произойдёт
						</Typography>
						<Typography component='ul' variant='body2' sx={{ m: 0, pl: 2.5, display: 'grid', gap: 0.75 }}>
							<li>
								{task.duplicateOf
									? `Заявка №${ticketNumber(task)} останется закрытой — сменится только основная.`
									: `Заявка №${ticketNumber(task)} будет закрыта. Её можно будет вернуть кнопкой «Отвязать».`}
							</li>
							<li>Комментарии и вложения останутся в этой заявке.</li>
							<li>Автор получит уведомление и будет следить за основной заявкой.</li>
						</Typography>
					</Box>
				)}
			</DialogContent>

			<DialogActions sx={{ px: 2.5, py: 1.75, gap: 1, borderTop: '1px solid', borderColor: 'divider' }}>
				<Button onClick={onClose} variant='outlined' sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button
					onClick={handleConfirm}
					variant='contained'
					disabled={!picked || linking}
					sx={{ textTransform: 'none' }}
				>
					{task.duplicateOf ? 'Изменить основную' : 'Отметить дублем'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
