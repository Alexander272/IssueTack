import { type FC } from 'react'
import { Chip, type ChipProps } from '@mui/material'

import { getReminderLabel, getReminderVisual } from './reminderOptions'

interface ReminderChipProps extends Omit<ChipProps, 'icon' | 'label' | 'size'> {
	value: number
}

export const ReminderChip: FC<ReminderChipProps> = ({ value, sx, ...props }) => {
	const { Icon, bg, color } = getReminderVisual(value)

	return (
		<Chip
			icon={<Icon sx={{ fontSize: 16 }} />}
			label={getReminderLabel(value)}
			{...props}
			sx={[
				{
					bgcolor: bg,
					color,
					fontWeight: 600,
					pl: 0.5,
					'& .MuiChip-icon': { color },
					'& .MuiChip-deleteIcon': { color, '&:hover': { color } },
				},
				...(Array.isArray(sx) ? sx : [sx]),
			]}
		/>
	)
}
