import { Box, Typography, type SxProps, type Theme } from '@mui/material'
import { CircleDashed, CopyPlus } from 'lucide-mui'
import type { TicketStatus } from '../types/task'
import { STATUS_MAP, type StatusMapValue } from '../constants/taskMaps'

interface Props {
	status: TicketStatus
	/** Номер основной заявки: дубль показываем вместо статуса «Закрыта». */
	duplicateOfNumber?: number | null
	sx?: SxProps<Theme>
}

const DUPLICATE_COLOR = '#8b5cf6'
const DUPLICATE_BG = '#f5f3ff'

export const TaskStatusBadge = ({ status, duplicateOfNumber, sx }: Props) => {
	const isDuplicate = duplicateOfNumber != null

	// Статус приходит из API, поэтому неизвестное значение не должно ронять рендер:
	// показываем нейтральный бейдж с сырым значением вместо обращения к undefined.
	const info: StatusMapValue | undefined = STATUS_MAP[status]
	const label = isDuplicate ? `дубль #${duplicateOfNumber}` : (info?.label ?? status)
	const textColor = isDuplicate ? DUPLICATE_COLOR : (info?.textColor ?? '#6b7280')
	const Icon = isDuplicate ? CopyPlus : (info?.icon ?? CircleDashed)

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
				bgcolor: isDuplicate ? DUPLICATE_BG : (info?.bgColor ?? '#f3f4f6'),
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
