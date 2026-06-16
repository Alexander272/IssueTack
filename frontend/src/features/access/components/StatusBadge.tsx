import { Box } from '@mui/material'
import type { FC } from 'react'

type Props = {
	active: boolean
	label: string
}

export const StatusBadge: FC<Props> = ({ active, label }) => {
	// Конфиг только для стилей
	const theme = active
		? { color: '#047857', bg: '#ecfdf5', dot: '#10b981' }
		: { color: '#b91c1c', bg: '#fef2f2', dot: '#ef4444' }

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 1,
				bgcolor: theme.bg,
				color: theme.color,
				px: 2,
				py: 0.75,
				borderRadius: '16px',
				fontSize: '0.875rem',
				fontWeight: 500,
			}}
		>
			<Box component='span' sx={{ width: 8, height: 8, bgcolor: theme.dot, borderRadius: '50%' }} />
			{label}
		</Box>
	)
}
