import { Box, Typography, type SxProps, type Theme } from '@mui/material'

import type { EquipmentStatus } from '../types/equipment'
import { getEquipmentStatus } from '../types/equipment'

interface Props {
	status: EquipmentStatus
	sx?: SxProps<Theme>
}

// Пилл статуса оборудования в стиле бейджей страницы заявок (TaskStatusBadge):
// тонированный фон + точка цвета статуса.
export const EquipmentStatusBadge = ({ status, sx }: Props) => {
	const info = getEquipmentStatus(status)

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 0.75,
				px: 1.25,
				py: 0.5,
				borderRadius: '999px',
				bgcolor: `${info.color}1a`,
				color: info.color,
				whiteSpace: 'nowrap',
				...sx,
			}}
		>
			<Box component='span' sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: info.color, flexShrink: 0 }} />
			<Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
				{info.label}
			</Typography>
		</Box>
	)
}
