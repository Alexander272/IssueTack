import type { FC } from 'react'
import { Box, Paper, Typography } from '@mui/material'
import { LineChart } from '@mui/x-charts/LineChart'

import type { ITrendPoint } from '../types/statistics'

interface Props {
	points: ITrendPoint[]
	isLoading: boolean
}

const formatLabel = (date: string) => {
	const [, month, day] = date.split('-')
	return `${day}.${month}`
}

export const TrendChart: FC<Props> = ({ points, isLoading }) => {
	return (
		<Paper sx={{ p: 3, border: '1px solid #e5e7eb', borderRadius: '12px', height: '100%' }} elevation={0}>
			<Typography variant='h6' sx={{ fontWeight: 600, mb: 2 }}>
				Динамика заявок
			</Typography>
			{points.length === 0 && !isLoading ? (
				<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
					<Typography variant='body2'>Нет данных за выбранный период</Typography>
				</Box>
			) : (
				<LineChart
					height={320}
					xAxis={[{ scaleType: 'point', data: points.map(point => formatLabel(point.date)) }]}
					yAxis={[{ min: 0 }]}
					series={[
						{
							data: points.map(point => point.created),
							label: 'Создано',
							color: '#2196f3',
							curve: 'monotoneX',
							showMark: true,
						},
						{
							data: points.map(point => point.resolved),
							label: 'Решено',
							color: '#4caf50',
							curve: 'monotoneX',
							showMark: true,
						},
					]}
				/>
			)}
		</Paper>
	)
}
