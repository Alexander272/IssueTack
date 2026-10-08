import type { FC } from 'react'
import { Box, Paper, Typography } from '@mui/material'
import { PieChart } from '@mui/x-charts/PieChart'

import type { TicketStatus } from '@/features/tasks/types/task'
import type { IStatusBucket } from '../types/statistics'
import { ACTIVE_STATUSES, STATUS_MAP } from '@/features/tasks/constants/taskMaps'
import { getChartColorFromText } from '../utils/color'

interface Props {
	buckets: IStatusBucket[]
	isLoading: boolean
}

const ORDER: TicketStatus[] = [...ACTIVE_STATUSES, 'resolved', 'closed', 'cancelled']

const fallbackColor = '#90a4ae'

export const StatusChart: FC<Props> = ({ buckets, isLoading }) => {
	const counts = new Map(buckets.map(bucket => [bucket.status, bucket.count]))
	const data = ORDER.filter(status => (counts.get(status) ?? 0) > 0).map(status => ({
		id: status,
		value: counts.get(status) ?? 0,
		label: STATUS_MAP[status]?.label ?? status,
		color: getChartColorFromText(STATUS_MAP[status]?.bgColor) ?? fallbackColor,
	}))

	return (
		<Paper sx={{ p: 3, border: '1px solid #e5e7eb', borderRadius: '12px', height: '100%' }} elevation={0}>
			<Typography variant='h6' sx={{ fontWeight: 600, mb: 2 }}>
				Заявки по статусам
			</Typography>
			{data.length === 0 && !isLoading ? (
				<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
					<Typography variant='body2'>Нет данных за выбранный период</Typography>
				</Box>
			) : (
				<PieChart
					height={320}
					series={[
						{
							data,
							innerRadius: '55%',
							paddingAngle: 3,
							cornerRadius: 8,
							arcLabel: 'value',
							arcLabelMinAngle: 10,
							highlightScope: { fade: 'global', highlight: 'item' },
						},
					]}
					margin={{ top: 8, bottom: 8, left: 8, right: 8 }}
				/>
			)}
		</Paper>
	)
}
