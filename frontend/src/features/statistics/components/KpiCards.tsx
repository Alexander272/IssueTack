import type { FC, ReactNode } from 'react'
import { alpha } from '@mui/material/styles'
import { Box, Grid, Paper, Skeleton, Typography } from '@mui/material'
import {
	AlertTriangleIcon,
	CheckCircleIcon,
	MinusIcon,
	SendIcon,
	TimerIcon,
	TrendingDownIcon,
	TrendingUpIcon,
} from 'lucide-mui'

import type { IStatisticsSummary } from '../types/statistics'

interface Props {
	summary?: IStatisticsSummary
	isLoading: boolean
}

type DeltaTone = 'up' | 'down' | 'flat'
interface Delta {
	pct: number
	tone: DeltaTone
}

const TONE_COLOR: Record<DeltaTone, string> = {
	up: '#16a34a',
	down: '#dc2626',
	flat: '#6b7280',
}

const DELTA_ICON = {
	up: TrendingUpIcon,
	down: TrendingDownIcon,
	flat: MinusIcon,
}

const percent = (value: number) => `${Math.round(value)}%`

const share = (part: number, whole: number) => (whole > 0 ? percent((part / whole) * 100) : null)

const delta = (current: number, previous: number): Delta | null => {
	if (previous <= 0) return null
	const pct = Math.round(((current - previous) / previous) * 100)
	return { pct, tone: pct > 0 ? 'up' : pct < 0 ? 'down' : 'flat' }
}

const Subtitle: FC<{ children: ReactNode; color?: string }> = ({ children, color = '#6b7280' }) => (
	<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, fontSize: 12, lineHeight: 1.4, color, mt: 0.5 }}>
		{children}
	</Box>
)

const renderDelta = (d: Delta | null, suffix: string): ReactNode => {
	if (!d) return <Subtitle>нет данных за прошлый период</Subtitle>
	const Icon = DELTA_ICON[d.tone]
	const text = `${d.pct > 0 ? '+' : ''}${d.pct}%`
	return (
		<Subtitle color={TONE_COLOR[d.tone]}>
			<Icon sx={{ fontSize: 14, color: TONE_COLOR[d.tone] }} />
			<span>
				{text} {suffix}
			</span>
		</Subtitle>
	)
}

const resolveSubtitle = (key: 'total' | 'active' | 'overdue' | 'resolved', s: IStatisticsSummary): ReactNode => {
	switch (key) {
		case 'total':
			return renderDelta(delta(s.total, s.totalPrev), 'к прошлому периоду')
		case 'resolved': {
			const resolvedShare = share(s.resolved, s.total)
			const d = delta(s.resolved, s.resolvedPrev)
			return (
				<Subtitle>
					{resolvedShare ? `${resolvedShare} от созданных` : 'нет данных за период'}
					{d && (
						<span style={{ color: TONE_COLOR[d.tone] }}>
							{' '}
							· {d.pct > 0 ? '+' : ''}
							{d.pct}%
						</span>
					)}
				</Subtitle>
			)
		}
		case 'active':
			return <Subtitle>{s.overdue > 0 ? `из них просрочено ${s.overdue}` : 'просроченных нет'}</Subtitle>
		case 'overdue': {
			const overdueShare = share(s.overdue, s.active)
			return <Subtitle>{overdueShare ? `${overdueShare} от активных` : 'активных нет'}</Subtitle>
		}
	}
}

const cards = [
	{ key: 'total', label: 'Новых заявок', color: '#2196f3', icon: SendIcon },
	{ key: 'active', label: 'Активные', color: '#ff9800', icon: TimerIcon },
	{ key: 'overdue', label: 'Просроченные', color: '#f44336', icon: AlertTriangleIcon },
	{ key: 'resolved', label: 'Решено', color: '#4caf50', icon: CheckCircleIcon },
] as const

export const KpiCards: FC<Props> = ({ summary, isLoading }) => {
	return (
		<Grid container spacing={3} sx={{ mb: 4 }}>
			{cards.map(card => {
				const Icon = card.icon
				return (
					<Grid key={card.key} size={{ xs: 12, sm: 6, md: 3 }}>
						<Paper sx={{ py: 2, px: 3, border: '1px solid #e5e7eb', borderRadius: '12px' }} elevation={0}>
							<Box
								sx={{
									display: 'flex',
									alignItems: 'flex-start',
									justifyContent: 'space-between',
									gap: 2,
								}}
							>
								<Box sx={{ minWidth: 0 }}>
									<Typography sx={{ display: 'block', mb: 0.5, color: '#4b5563' }}>
										{card.label}
									</Typography>
									{isLoading ? (
										<Skeleton variant='text' width={64} height={40} />
									) : (
										<Typography variant='h4' sx={{ color: card.color, fontWeight: 'bold' }}>
											{summary?.[card.key] ?? 0}
										</Typography>
									)}
									{!isLoading && summary && resolveSubtitle(card.key, summary)}
								</Box>
								<Box
									sx={{
										flexShrink: 0,
										width: 48,
										height: 48,
										borderRadius: 2,
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'center',
										bgcolor: alpha(card.color, 0.12),
									}}
								>
									<Icon sx={{ color: card.color, fontSize: 26 }} />
								</Box>
							</Box>
						</Paper>
					</Grid>
				)
			})}
		</Grid>
	)
}
