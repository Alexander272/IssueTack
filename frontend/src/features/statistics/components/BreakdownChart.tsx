import { useEffect, useRef, useState, type FC } from 'react'
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

// До первого замера ResizeObserver — фолбэк ширины контента карточки.
const FALLBACK_CONTENT_WIDTH = 160

export const BreakdownChart: FC<Props> = ({ title, buckets, color = '#2196f3', isLoading }) => {
	const contentRef = useRef<HTMLDivElement | null>(null)
	const [contentWidth, setContentWidth] = useState(0)

	useEffect(() => {
		const el = contentRef.current
		if (!el) return
		const observer = new ResizeObserver(entries => {
			setContentWidth(entries[0]?.contentRect.width ?? 0)
		})
		observer.observe(el)
		return () => observer.disconnect()
	}, [])

	const height = Math.max(160, buckets.length * 44 + 48)
	const longest = buckets.reduce((acc, bucket) => Math.max(acc, bucket.name.length), 0)
	// Ширина оси подписей — не больше доли карточки: иначе на узких карточках
	// (md:4, ~260px) фиксированные 300px съедают всё поле графика.
	const needed = Math.max(120, Math.min(longest, 40) * 7)
	const allowed = Math.max(80, Math.round((contentWidth || FALLBACK_CONTENT_WIDTH) * 0.4))
	const axisWidth = Math.min(needed, allowed, 300)
	const labelMax = Math.max(8, Math.floor(axisWidth / 7))

	return (
		<Paper sx={{ p: 3, border: '1px solid #e5e7eb', borderRadius: '12px', height: '100%' }} elevation={0}>
			<Box ref={contentRef}>
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
								data: buckets.map(bucket => truncate(bucket.name, labelMax)),
								tickLabelStyle: { fontSize: 12 },
								width: axisWidth,
							},
						]}
						// xAxis={[{ min: 0 }]}
						series={[{ data: buckets.map(bucket => bucket.count), color }]}
						margin={{ right: 24, top: 8, bottom: 24 }}
						borderRadius={8}
					/>
				)}
			</Box>
		</Paper>
	)
}
