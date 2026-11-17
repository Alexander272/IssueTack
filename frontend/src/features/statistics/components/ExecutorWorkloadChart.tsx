import { useMemo, type FC } from 'react'
import { Box, Paper, Skeleton, Typography } from '@mui/material'
import { BarChart } from '@mui/x-charts/BarChart'

import type { IWorkloadBucket } from '../types/statistics'
import { TopNSelect } from './TopNSelect'
import { WorkloadBarTooltip } from './WorkloadBarTooltip'
import { EXECUTOR_SERIES_COLORS, getWorkloadCellColor, WORKLOAD_REST_ID } from './workloadChartColors'
import { CustomBarLabel } from './BarLabel'

interface Props {
	title: string
	buckets: IWorkloadBucket[]
	topN: number
	onTopNChange: (value: number) => void
	isLoading: boolean
	onBarClick?: (bucketId: string, ring: 'active' | 'closed') => void
}

const aggregate = (buckets: IWorkloadBucket[], topN: number): IWorkloadBucket[] => {
	const sorted = [...buckets].sort(
		(a, b) =>
			b.activeTasks +
			b.closedTasks +
			b.activeSubtasks +
			b.closedSubtasks -
			(a.activeTasks + a.closedTasks + a.activeSubtasks + a.closedSubtasks),
	)
	const shown = topN > 0 ? sorted.slice(0, topN) : sorted
	const rest = sorted.slice(shown.length)
	if (rest.length === 0) return shown
	return [
		...shown,
		{
			userId: WORKLOAD_REST_ID,
			name: 'Прочие',
			activeTasks: rest.reduce((acc, b) => acc + b.activeTasks, 0),
			activeSubtasks: rest.reduce((acc, b) => acc + b.activeSubtasks, 0),
			closedTasks: rest.reduce((acc, b) => acc + b.closedTasks, 0),
			closedSubtasks: rest.reduce((acc, b) => acc + b.closedSubtasks, 0),
		},
	]
}

export const ExecutorWorkloadChart: FC<Props> = ({ title, buckets, topN, onTopNChange, isLoading, onBarClick }) => {
	const rows = useMemo(() => aggregate(buckets, topN), [buckets, topN])

	const userIds = useMemo(() => rows.map(r => r.userId), [rows])

	const nameByUserId = useMemo(() => {
		const map = new Map<string, string>()
		rows.forEach(r => map.set(r.userId, r.name))
		return map
	}, [rows])

	const series = useMemo(
		() => [
			{
				id: 'activeTasks',
				label: 'Задачи (активные)',
				data: rows.map(r => r.activeTasks),
				stack: 'active',
				barLabel: 'value' as const,
				color: EXECUTOR_SERIES_COLORS.activeTasks.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(
						rows[dataIndex]?.userId ?? WORKLOAD_REST_ID,
						EXECUTOR_SERIES_COLORS.activeTasks,
					),
			},
			{
				id: 'activeSubtasks',
				label: 'Подзадачи (активные)',
				data: rows.map(r => r.activeSubtasks),
				stack: 'active',
				barLabel: 'value' as const,
				color: EXECUTOR_SERIES_COLORS.activeSubtasks.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(
						rows[dataIndex]?.userId ?? WORKLOAD_REST_ID,
						EXECUTOR_SERIES_COLORS.activeSubtasks,
					),
			},
			{
				id: 'closedTasks',
				label: 'Задачи (закрытые)',
				data: rows.map(r => r.closedTasks),
				stack: 'closed',
				barLabel: 'value' as const,
				color: EXECUTOR_SERIES_COLORS.closedTasks.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(
						rows[dataIndex]?.userId ?? WORKLOAD_REST_ID,
						EXECUTOR_SERIES_COLORS.closedTasks,
					),
			},
			{
				id: 'closedSubtasks',
				label: 'Подзадачи (закрытые)',
				data: rows.map(r => r.closedSubtasks),
				stack: 'closed',
				barLabel: 'value' as const,
				color: EXECUTOR_SERIES_COLORS.closedSubtasks.legendColor,
				colorGetter: ({ dataIndex }: { dataIndex: number }) =>
					getWorkloadCellColor(
						rows[dataIndex]?.userId ?? WORKLOAD_REST_ID,
						EXECUTOR_SERIES_COLORS.closedSubtasks,
					),
			},
		],
		[rows],
	)

	const handleItemClick = (_event: unknown, item: { dataIndex: number; seriesId: string }) => {
		if (!onBarClick || item.dataIndex == null) return
		const bucket = rows[item.dataIndex]
		if (!bucket || bucket.userId === WORKLOAD_REST_ID) return
		const ring: 'active' | 'closed' = item.seriesId.startsWith('active') ? 'active' : 'closed'
		onBarClick(bucket.userId, ring)
	}

	const empty = buckets.length === 0
	const height = rows.length * 56 + 95

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
							data: userIds,
							valueFormatter: (userId: string) => nameByUserId.get(userId) ?? userId,
							tickLabelStyle: { fontSize: 12 },
							width: 150,
							// categoryGapRatio: 0.3,
							barGapRatio: 0.15,
						},
					]}
					xAxis={[{ tickLabelStyle: { fontSize: 11 } }]}
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
