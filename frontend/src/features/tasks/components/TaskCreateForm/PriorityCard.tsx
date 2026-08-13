import { Box, Typography } from '@mui/material'
import type { Priority } from '../../types/task'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { PRIORITY_DESCRIPTIONS } from './priorityDescriptions'

type Props = {
	value: Priority
	selected: boolean
	onSelect: (v: Priority) => void
}

export const PriorityCard = ({ value, selected, onSelect }: Props) => {
	const info = PRIORITY_MAP[value]
	return (
		<Box
			onClick={() => onSelect(value)}
			sx={{
				border: `2px solid ${selected ? 'primary.main' : '#e5e7eb'}`,
				borderRadius: '12px',
				p: 2,
				cursor: 'pointer',
				bgcolor: selected ? '#eff6ff' : '#fff',
				transition: 'transform 0.15s ease-in-out 0s, background-color 0.15s ease-in-out 0s',
				'&:hover': { transform: 'translateY(-2px)', boxShadow: '0 4px 6px rgba(0,0,0,0.08)' },
			}}
		>
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
				<Box sx={{ width: 12, height: 12, borderRadius: '50%', bgcolor: info.barColor, flexShrink: 0 }} />
				<Typography sx={{ fontWeight: 600, fontSize: '0.875rem', color: '#1f2937' }}>{info.label}</Typography>
			</Box>
			<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>{PRIORITY_DESCRIPTIONS[value]}</Typography>
		</Box>
	)
}
