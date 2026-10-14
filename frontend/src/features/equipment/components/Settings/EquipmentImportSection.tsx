import { Box, Typography } from '@mui/material'
import type { FC } from 'react'

export const EquipmentImportSection: FC = () => {
	return (
		<Box sx={{ p: 3 }}>
			<Typography variant='body1' sx={{ fontWeight: 600, mb: 1 }}>
				Импорт
			</Typography>
			<Typography variant='body2' sx={{ color: '#6b7280' }}>
				Раздел в разработке
			</Typography>
		</Box>
	)
}
