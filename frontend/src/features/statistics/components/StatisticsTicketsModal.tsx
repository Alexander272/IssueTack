import { useState, type FC } from 'react'
import { useNavigate } from 'react-router'
import {
	Box,
	Button,
	CircularProgress,
	Dialog,
	DialogContent,
	DialogTitle,
	IconButton,
	Typography,
} from '@mui/material'
import { UsersIcon, XIcon } from 'lucide-mui'

import type { ITask } from '@/features/tasks/types/task'
import { Avatar } from '@/components/Avatar'
import { TaskStatusBadge } from '@/features/tasks/components/TaskStatusBadge'
import { TaskPriorityBadge } from '@/features/tasks/components/TaskPriorityBadge'
import {
	ACTIVE_STATUSES,
	DUE_DATE_COLORS,
	DUE_DATE_ICONS,
	DUE_DATE_DEFAULT_ICON,
} from '@/features/tasks/constants/taskMaps'
import { AppRoutes } from '@/pages/router/routes'
import { getAvatarColor, getInitials } from '@/utils/avatar'
import { getDueDateTone, getShortDate } from '@/utils/date'
import { useGetStatisticsTicketsQuery } from '../statisticsApiSlice'
import type { IStatisticsFilter, StatisticsDrilldownDimension, StatisticsDrilldownRing } from '../types/statistics'

const PAGE_SIZE = 20
// Заявки без заказчика бакет приходит с нулевым uuid (см. GetStatisticsByOwner).
const ZERO_UUID = '00000000-0000-0000-0000-000000000000'

interface Props {
	dimension: StatisticsDrilldownDimension
	bucketId: string
	bucketName: string
	ring: StatisticsDrilldownRing
	filter: IStatisticsFilter
	onClose: () => void
}

const RING_LABEL: Record<StatisticsDrilldownRing, string> = {
	'': 'Все заявки за период',
	active: 'Активные заявки',
	closed: 'Закрытые заявки',
}

const DUE_LABEL: Record<Exclude<ReturnType<typeof getDueDateTone>, null>, string> = {
	overdue: 'просрочено',
	today: 'сегодня',
	soon: 'скоро',
}

// personLabel показывает вторую сторону заявки: в разрезе исполнителя — заказчика,
// в разрезе заказчика — исполнителя.
const personLabel = (task: ITask, dimension: StatisticsDrilldownDimension) => {
	const user = dimension === 'owner' ? task.assignee : task.owner
	if (!user) return dimension === 'owner' ? 'Без исполнителя' : 'Без заказчика'
	return `${user.lastName} ${user.firstName}${user.internalNumber ? ` (${user.internalNumber})` : ''}`
}

// StatisticsTicketsModal — drill-down статистики: список заявок, стоящих за
// сектором диаграммы. Срез уже посчитан бэкендом по тем же период/уточнениям,
// что и агрегаты, поэтому число в списке совпадает с числом в секции. Клик по
// строке уводит на карточку заявки.
export const StatisticsTicketsModal: FC<Props> = ({ dimension, bucketId, bucketName, ring, filter, onClose }) => {
	const navigate = useNavigate()
	const [limit, setLimit] = useState(PAGE_SIZE)

	const { data, isFetching, isLoading } = useGetStatisticsTicketsQuery({
		...filter,
		dim: dimension,
		id: bucketId,
		...(ring ? { ring } : {}),
		limit,
		offset: 0,
	})

	// data реюзается между сменой аргументов (RTK держит последний результат),
	// поэтому при «Показать ещё» старый список не пропадает в спиннер — он
	// заменяется целиком, когда пришли новые строки.
	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const isNoPerson = dimension === 'owner' && bucketId === ZERO_UUID

	const handleOpenTask = (task: ITask) => {
		onClose()
		navigate(`${AppRoutes.Tasks}/${task.id}`)
	}

	return (
		<Dialog
			open
			onClose={onClose}
			fullWidth
			maxWidth='md'
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '16px' } } } }}
		>
			<DialogTitle sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.5 }}>
				{isNoPerson ? (
					<Avatar size={44} bgcolor='#9aa5b1'>
						<UsersIcon sx={{ fontSize: 22 }} />
					</Avatar>
				) : (
					<Avatar size={44} bgcolor={getAvatarColor(bucketId)}>
						{getInitials(bucketName)}
					</Avatar>
				)}
				<Box sx={{ minWidth: 0, flexGrow: 1 }}>
					<Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1, minWidth: 0 }}>
						<Typography variant='h6' noWrap sx={{ fontWeight: 'bold', lineHeight: 1.2 }}>
							{isNoPerson ? 'Без заказчика' : bucketName}
						</Typography>
						<Typography sx={{ flexShrink: 0, fontSize: '0.825rem', fontWeight: 400, color: '#6b7280' }}>
							{dimension === 'assignee' ? ' · исполнитель' : ' · заказчик'}
						</Typography>
					</Box>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 0.25, color: '#6b7280' }}>
						<Typography variant='body2' sx={{ color: 'inherit' }}>
							{RING_LABEL[ring]}
						</Typography>
						{total > 0 && (
							<>
								<Box sx={{ width: 4, height: 4, borderRadius: '50%', bgcolor: '#d1d5db' }} />
								<Typography variant='body2' sx={{ fontWeight: 600, color: 'primary.main' }}>
									{total}
								</Typography>
							</>
						)}
					</Box>
				</Box>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary', mt: -0.5 }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ p: 0 }}>
				{isLoading ? (
					<Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
						<CircularProgress size={28} />
					</Box>
				) : tasks.length === 0 ? (
					<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
						<Typography variant='body2'>Нет заявок по выбранному сектору</Typography>
					</Box>
				) : (
					tasks.map(task => {
						// Срок подсвечиваем как в таблице заявок, но только пока заявка
						// активна: у решённых/закрытых тон просрочки смысла не имеет.
						const dueTone = ACTIVE_STATUSES.includes(task.status) ? getDueDateTone(task.dueDate) : null
						const DueDateIcon = dueTone ? DUE_DATE_ICONS[dueTone] : DUE_DATE_DEFAULT_ICON
						const dueDateColor = dueTone ? DUE_DATE_COLORS[dueTone] : '#4b5563'
						const dueBold = dueTone === 'overdue' || dueTone === 'today'

						return (
							<Box
								key={task.id}
								onClick={() => handleOpenTask(task)}
								sx={{
									display: 'flex',
									alignItems: 'center',
									gap: 1.5,
									px: { xs: 2, sm: 2.5 },
									py: 1.75,
									cursor: 'pointer',
									borderBottom: '1px solid #f3f4f6',
									'&:hover': { bgcolor: '#f8fafc' },
									'&:hover .ticket-title': { color: 'primary.main' },
									'&:hover .row-open': { opacity: 1 },
									'&:last-child': { borderBottom: 0 },
								}}
							>
								<Box
									sx={{
										display: 'flex',
										flexDirection: 'column',
										alignItems: 'center',
										justifyContent: 'center',
										gap: 0.5,
										flexShrink: 0,
										minWidth: 28,
									}}
								>
									<Typography
										sx={{ fontWeight: 700, color: '#111827', fontSize: '1.125rem', lineHeight: 1 }}
									>
										{task.ticketNumber ?? '—'}
									</Typography>
									{dueTone && (
										<Box
											sx={{
												width: 6,
												height: 6,
												borderRadius: '50%',
												bgcolor: DUE_DATE_COLORS[dueTone],
											}}
										/>
									)}
								</Box>

								<Box sx={{ flexGrow: 1, minWidth: 0 }}>
									<Box
										sx={{
											display: 'flex',
											alignItems: 'flex-start',
											justifyContent: 'space-between',
											gap: 1,
											mb: 0.25,
										}}
									>
										<Typography
											className='ticket-title'
											sx={{
												fontWeight: 600,
												color: '#111827',
												fontSize: '0.875rem',
												overflow: 'hidden',
												textOverflow: 'ellipsis',
												whiteSpace: 'nowrap',
												transition: 'color .15s',
											}}
										>
											{task.title}
										</Typography>
									</Box>
									<Typography
										sx={{
											color: '#6b7280',
											fontSize: '0.75rem',
											overflow: 'hidden',
											textOverflow: 'ellipsis',
											whiteSpace: 'nowrap',
										}}
									>
										{personLabel(task, dimension)} · {task.site?.name ?? '—'}
									</Typography>
									<Typography sx={{ color: '#9ca3af', fontSize: '0.75rem', mt: 0.25 }}>
										Создана {getShortDate(task.createdAt)}
									</Typography>
								</Box>

								<Box
									sx={{
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'center',
										gap: 1.5,
										flexShrink: 0,
									}}
								>
									<Box sx={{ display: 'flex', gap: 0.75, flexShrink: 0 }}>
										<Box sx={{ display: { xs: 'none', sm: 'flex' } }}>
											<TaskPriorityBadge priority={task.priority} />
										</Box>
										<TaskStatusBadge status={task.status} />
									</Box>

									<Box
										sx={{
											display: { xs: 'none', md: 'flex' },
											flexDirection: 'column',
											alignItems: 'flex-end',
											gap: 0.5,
										}}
									>
										<Box
											sx={{
												display: 'flex',
												alignItems: 'center',
												gap: 0.75,
												color: dueDateColor,
											}}
										>
											<DueDateIcon sx={{ fontSize: 14, color: dueDateColor }} />
											<Typography
												component='span'
												sx={{
													fontSize: '0.8125rem',
													fontWeight: dueBold ? 700 : 400,
													lineHeight: 1,
													color: dueDateColor,
												}}
											>
												{getShortDate(task.dueDate)}
											</Typography>
										</Box>
										{dueTone && (
											<Typography
												sx={{
													fontSize: '0.6875rem',
													color: dueDateColor,
													lineHeight: 1,
												}}
											>
												{DUE_LABEL[dueTone]}
											</Typography>
										)}
									</Box>
								</Box>
							</Box>
						)
					})
				)}
			</DialogContent>

			{tasks.length > 0 && (
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						gap: 1,
						px: 2.5,
						py: 1.5,
						bgcolor: '#f9fafb',
					}}
				>
					<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
						Кликните на заявку, чтобы открыть её
					</Typography>
					{total > tasks.length && (
						<Button onClick={() => setLimit(value => value + PAGE_SIZE)} disabled={isFetching}>
							Показать ещё
						</Button>
					)}
				</Box>
			)}
		</Dialog>
	)
}
