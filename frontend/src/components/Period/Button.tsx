import { Box } from '@mui/material'
import type { FC } from 'react'

type Props = {
	children: React.ReactNode
	onClick: () => void
	disabled?: boolean
}

export const ActionButton: FC<Props> = ({ children, onClick, disabled }) => (
	<Box
		onClick={disabled ? undefined : onClick}
		sx={{
			flex: 1,
			textAlign: 'center',
			py: 1,
			borderRadius: 2,
			fontSize: '14px',
			userSelect: 'none',
			bgcolor: disabled ? 'grey.300' : 'primary.main',
			color: disabled ? 'text.disabled' : '#fff',
			cursor: disabled ? 'default' : 'pointer',
			'&:hover': {
				bgcolor: disabled ? 'grey.300' : 'primary.dark',
			},
		}}
	>
		{children}
	</Box>
)
