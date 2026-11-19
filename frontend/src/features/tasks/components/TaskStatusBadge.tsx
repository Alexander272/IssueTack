import { Box, Typography, type SxProps, type Theme } from '@mui/material'
import { CircleDashed } from 'lucide-mui'
import type { TicketStatus } from '../types/task'
import { STATUS_MAP, type StatusMapValue } from '../constants/taskMaps'

interface Props {
	status: TicketStatus
	sx?: SxProps<Theme>
}

export const TaskStatusBadge = ({ status, sx }: Props) => {
	// Статус приходит из API, поэтому неизвестное значение не должно ронять рендер:
	// показываем нейтральный бейдж с сырым значением вместо обращения к undefined.
	const info: StatusMapValue | undefined = STATUS_MAP[status]
	const label = info?.label ?? status
	const textColor = info?.textColor ?? '#6b7280'
	const Icon = info?.icon ?? CircleDashed

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
				bgcolor: info?.bgColor ?? '#f3f4f6',
				color: textColor,
				...sx,
			}}
		>
			<Icon sx={{ fontSize: 14, color: textColor }} />
			<Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
				{label}
			</Typography>
		</Box>
	)
}
