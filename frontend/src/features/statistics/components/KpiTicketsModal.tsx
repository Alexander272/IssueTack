import { useState, type FC } from 'react'
import { useNavigate } from 'react-router'
import { Box, Dialog, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import { XIcon } from 'lucide-mui'

import type { ITask } from '@/features/tasks/types/task'
import { AppRoutes } from '@/pages/router/routes'
import { getShortDate } from '@/utils/date'
import { useGetKpiTicketsQuery } from '../statisticsApiSlice'
import type { IStatisticsFilter, KpiCardKey } from '../types/statistics'
import { StatisticsTicketsList } from './StatisticsTicketsList'

const CARD_LABELS: Record<KpiCardKey, string> = {
	total: 'Новые заявки',
	active: 'Активные заявки',
	overdue: 'Просроченные заявки',
	resolved: 'Решённые заявки',
}

const CARD_COLORS: Record<KpiCardKey, string> = {
	total: '#2196f3',
	active: '#ff9800',
	overdue: '#f44336',
	resolved: '#4caf50',
}

interface Props {
	cardKey: KpiCardKey
	filter: IStatisticsFilter
	onClose: () => void
}

export const KpiTicketsModal: FC<Props> = ({ cardKey, filter, onClose }) => {
	const navigate = useNavigate()
	const [limit, setLimit] = useState(20)

	const { data, isFetching, isLoading } = useGetKpiTicketsQuery({
		...filter,
		cardKey,
		limit,
		offset: 0,
	})

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const handleOpenTask = (task: ITask) => {
		onClose()
		navigate(`${AppRoutes.Tasks}/${task.id}`)
	}

	const title = CARD_LABELS[cardKey]
	const color = CARD_COLORS[cardKey]

	return (
		<Dialog
			open
			onClose={onClose}
			fullWidth
			maxWidth='md'
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '16px' } } } }}
		>
			<DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
				<Box
					sx={{
						width: 44,
						height: 44,
						borderRadius: '12px',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						bgcolor: color,
						color: '#fff',
					}}
				>
					<Typography variant='h6' sx={{ fontWeight: 'bold' }}>
						{total}
					</Typography>
				</Box>
				<Box sx={{ minWidth: 0, flexGrow: 1 }}>
					<Typography variant='h6' sx={{ fontWeight: 'bold', lineHeight: 1.2, color }}>
						{title}
					</Typography>
					<Typography sx={{ fontSize: '0.825rem', color: '#6b7280' }}>
						За период {getShortDate(filter.from)} – {getShortDate(filter.to)}
					</Typography>
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
					onShowMore={() => setLimit(value => value + 20)}
				/>
			</DialogContent>
		</Dialog>
	)
}
