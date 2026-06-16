import { Box } from '@mui/material'

import type { Preset } from './types'
import type { FC } from 'react'

type Props = {
	preset: Preset
	isActive: boolean
	onClick: () => void
}

export const PresetChip: FC<Props> = ({ preset, isActive, onClick }) => (
	<Box
		onClick={onClick}
		sx={{
			px: 1.5,
			py: 0.75,
			fontSize: '12px',
			borderRadius: 9999,
			bgcolor: isActive ? 'primary.main' : 'grey.100',
			color: isActive ? '#fff' : 'text.primary',
			cursor: 'pointer',
			userSelect: 'none',
			transition: '0.15s',
			'&:hover': {
				bgcolor: isActive ? 'primary.dark' : 'grey.200',
			},
		}}
	>
		{preset.label}
	</Box>
)
