import type { FC } from 'react'
import { Box, Paper, Skeleton, Typography } from '@mui/material'
import { BarChart } from '@mui/x-charts/BarChart'

import type { IStatisticsBucket } from '../types/statistics'

interface Props {
	title: string
	buckets: IStatisticsBucket[]
	color?: string
	isLoading: boolean
}

const truncate = (value: string, max = 40) => (value.length > max ? `${value.slice(0, max - 1)}…` : value)

export const BreakdownChart: FC<Props> = ({ title, buckets, color = '#2196f3', isLoading }) => {
	const height = Math.max(160, buckets.length * 44 + 48)
	const longest = buckets.reduce((acc, bucket) => Math.max(acc, bucket.name.length), 0)
	const width = Math.min(300, Math.max(120, Math.min(longest, 40) * 7))

	return (
		<Paper sx={{ p: 3, border: '1px solid #eee', borderRadius: 2, height: '100%' }} elevation={0}>
			<Typography variant='h6' sx={{ fontWeight: 600, mb: 2 }}>
				{title}
			</Typography>
			{buckets.length === 0 ? (
				isLoading ? (
					<Skeleton variant='rounded' height={height} />
				) : (
					<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
						<Typography variant='body2'>Нет данных за выбранный период</Typography>
					</Box>
				)
			) : (
				<BarChart
					layout='horizontal'
					height={height}
					yAxis={[
						{
							scaleType: 'band',
							data: buckets.map(bucket => truncate(bucket.name)),
							tickLabelStyle: { fontSize: 12 },
							width: width,
						},
					]}
					// xAxis={[{ min: 0 }]}
					series={[{ data: buckets.map(bucket => bucket.count), color }]}
					margin={{ right: 24, top: 8, bottom: 24 }}
					borderRadius={8}
				/>
			)}
		</Paper>
	)
}
