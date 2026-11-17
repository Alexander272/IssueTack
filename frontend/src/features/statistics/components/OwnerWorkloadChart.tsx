import { useMemo, type FC } from 'react'
import { Box, Paper, Skeleton, Typography } from '@mui/material'
import { BarChart } from '@mui/x-charts/BarChart'

import type { IDonutBucket } from './NestedDonut'
import { TopNSelect } from './TopNSelect'
import { WorkloadBarTooltip } from './WorkloadBarTooltip'
import { getWorkloadCellColor, OWNER_SERIES_COLORS, WORKLOAD_REST_ID } from './workloadChartColors'
import { CustomBarLabel } from './BarLabel'

interface Props {
	title: string
	buckets: IDonutBucket[]
	topN: number
	onTopNChange: (value: number) => void
	isLoading: boolean
	onBarClick?: (bucketId: string, ring: 'active' | 'closed') => void
}

const aggregate = (buckets: IDonutBucket[], topN: number): IDonutBucket[] => {
	const sorted = [...buckets].sort((a, b) => b.total - a.total)
	const shown = topN > 0 ? sorted.slice(0, topN) : sorted
	const rest = sorted.slice(shown.length)
	if (rest.length === 0) return shown
	return [
		...shown,
		{
			id: WORKLOAD_REST_ID,
			name: 'Прочие',
			total: rest.reduce((acc, b) => acc + b.total, 0),
			active: rest.reduce((acc, b) => acc + b.active, 0),
		},
	]
}

export const OwnerWorkloadChart: FC<Props> = ({ title, buckets, topN, onTopNChange, isLoading, onBarClick }) => {
	const rows = useMemo(() => aggregate(buckets, topN), [buckets, topN])

	const ownerIds = useMemo(() => rows.map(r => r.id), [rows])

	const nameById = useMemo(() => {
		const map = new Map<string, string>()
		rows.forEach(r => map.set(r.id, r.name))
		return map
	}, [rows])

	const series = useMemo(
		() => [
			{
				id: 'active',
				label: 'Активные',
				data: rows.map(r => r.active),
				stack: 'total',
				barLabel: 'value' as const,
				color: OWNER_SERIES_COLORS.active.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(rows[dataIndex]?.id ?? WORKLOAD_REST_ID, OWNER_SERIES_COLORS.active),
			},
			{
				id: 'closed',
				label: 'Закрытые',
				data: rows.map(r => Math.max(r.total - r.active, 0)),
				stack: 'total',
				barLabel: 'value' as const,
				color: OWNER_SERIES_COLORS.closed.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(rows[dataIndex]?.id ?? WORKLOAD_REST_ID, OWNER_SERIES_COLORS.closed),
			},
		],
		[rows],
	)

	const handleItemClick = (_event: unknown, item: { dataIndex: number; seriesId: string }) => {
		if (!onBarClick || item.dataIndex == null) return
		const bucket = rows[item.dataIndex]
		if (!bucket || bucket.id === WORKLOAD_REST_ID) return
		const ring: 'active' | 'closed' = item.seriesId === 'active' ? 'active' : 'closed'
		onBarClick(bucket.id, ring)
	}

	const empty = buckets.length === 0
	const height = rows.length * 38 + 95

	return (
		<Paper sx={{ p: 3, border: '1px solid #e5e7eb', borderRadius: '12px', height: '100%' }} elevation={0}>
			<Box
				sx={{
					display: 'flex',
					flexWrap: 'wrap',
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: 1,
					mb: 2,
				}}
			>
				<Typography variant='h6' sx={{ fontWeight: 600 }}>
					{title}
				</Typography>
				<TopNSelect value={topN} onChange={onTopNChange} />
			</Box>

			{empty ? (
				isLoading ? (
					<Skeleton variant='rounded' height={height} />
				) : (
					<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
						<Typography variant='body2'>Нет данных за выбранный период</Typography>
					</Box>
				)
			) : (
				<BarChart
					height={height}
					layout='horizontal'
					yAxis={[
						{
							scaleType: 'band',
							data: ownerIds,
							valueFormatter: (id: string) => nameById.get(id) ?? id,
							tickLabelStyle: { fontSize: 12 },
							width: 150,
							categoryGapRatio: 0.2,
						},
					]}
					xAxis={[{ min: 0, tickLabelStyle: { fontSize: 11 } }]}
					series={series}
					onItemClick={handleItemClick}
					margin={{ right: 24, top: 8, bottom: 24 }}
					borderRadius={8}
					hideLegend={false}
					slots={{ tooltip: WorkloadBarTooltip, barLabel: CustomBarLabel }}
				/>
			)}
		</Paper>
	)
}
