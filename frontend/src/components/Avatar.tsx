import { Box, type BoxProps } from '@mui/material'
import type { ReactNode } from 'react'

interface Props {
	children: ReactNode
	size?: number
	bgcolor?: string
	sx?: BoxProps['sx']
}

export const Avatar = ({ children, size = 28, bgcolor = '#3b82f6', sx }: Props) => (
	<Box
		sx={{
			width: size,
			height: size,
			borderRadius: '50%',
			bgcolor,
			display: 'flex',
			alignItems: 'center',
			justifyContent: 'center',
			fontSize: Math.max(8, size * 0.4),
			color: '#fff',
			fontWeight: 700,
			flexShrink: 0,
			...sx,
		}}
	>
		{children}
	</Box>
)
