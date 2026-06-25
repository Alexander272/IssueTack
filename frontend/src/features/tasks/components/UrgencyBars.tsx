import { Box } from '@mui/material'
import type { FC } from 'react'

type Props = {
	level: number
	color: string
}

const BAR_WIDTH = 10
const BAR_HEIGHT = 2
const GAP = 2
const TOTAL = 4
const EMPTY_COLOR = '#e5e7eb'

export const UrgencyBars: FC<Props> = ({ level, color }) => {
	return (
		<Box
			sx={{
				display: 'inline-flex',
				flexDirection: 'column',
				alignItems: 'center',
				gap: `${GAP}px`,
			}}
		>
			{Array.from({ length: TOTAL }, (_, i) => {
				const filled = i >= TOTAL - level
				return (
					<Box
						key={i}
						sx={{
							width: BAR_WIDTH,
							height: BAR_HEIGHT,
							borderRadius: '1px',
							bgcolor: filled ? color : EMPTY_COLOR,
							flexShrink: 0,
						}}
					/>
				)
			})}
		</Box>
	)
}
