import { Box } from '@mui/material'
import type { Priority } from '../types/task'
import { PRIORITY_MAP } from '../constants/taskMaps'
import { UrgencyBars } from './UrgencyBars'

interface Props {
	priority: Priority
}

export const TaskPriorityBadge = ({ priority }: Props) => {
	const info = PRIORITY_MAP[priority]

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 1,
				bgcolor: info.bgColor,
				color: info.textColor,
				px: 2,
				py: 0.75,
				borderRadius: '16px',
				fontSize: '0.875rem',
				fontWeight: 500,
			}}
		>
			<UrgencyBars level={info.barCount} color={info.barColor} />
			{info.label}
		</Box>
	)
}
