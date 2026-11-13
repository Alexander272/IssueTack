import { useState, type FC } from 'react'
import { useNavigate } from 'react-router'
import { Box, Dialog, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import { UsersIcon, XIcon } from 'lucide-mui'

import type { ITask } from '@/features/tasks/types/task'
import { Avatar } from '@/components/Avatar'
import { AppRoutes } from '@/pages/router/routes'
import { getAvatarColor, getInitials } from '@/utils/avatar'
import { useGetStatisticsTicketsQuery } from '../statisticsApiSlice'
import type { IStatisticsFilter, StatisticsDrilldownDimension, StatisticsDrilldownRing } from '../types/statistics'
import { StatisticsTicketsList } from './StatisticsTicketsList'

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

const DIMENSION_LABELS: Record<StatisticsDrilldownDimension, string> = {
	assignee: 'исполнитель',
	owner: 'заказчик',
	category: 'категория',
	site: 'площадка',
	group: 'группа',
	status: 'статус',
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
							· {DIMENSION_LABELS[dimension]}
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
				<StatisticsTicketsList
					tasks={tasks}
					isLoading={isLoading}
					isFetching={isFetching}
					total={total}
					onOpenTask={handleOpenTask}
					onShowMore={() => setLimit(value => value + PAGE_SIZE)}
					dimension={dimension === 'assignee' ? 'assignee' : dimension === 'owner' ? 'owner' : undefined}
				/>
			</DialogContent>
		</Dialog>
	)
}
