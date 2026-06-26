import { Box, Typography, type SxProps, type Theme } from '@mui/material'
import type { TicketStatus } from '../types/task'
import { STATUS_MAP } from '../constants/taskMaps'

interface Props {
	status: TicketStatus
	sx?: SxProps<Theme>
}

export const TaskStatusBadge = ({ status, sx }: Props) => {
	const info = STATUS_MAP[status]

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 0.75,
				px: 1.25,
				py: 1,
				borderRadius: '999px',
				fontSize: '0.75rem',
				fontWeight: 500,
				bgcolor: info.bgColor,
				color: info.textColor,
				...sx,
			}}
		>
			<info.icon sx={{ fontSize: 14, fill: info.textColor }} />
			<Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
				{info.label}
			</Typography>
		</Box>
	)
}
