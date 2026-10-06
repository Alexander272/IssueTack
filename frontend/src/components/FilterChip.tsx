import { Box, Chip } from '@mui/material'

import { CHIP_MAX_WIDTH, fitChipLabel } from '@/utils/chipLabel'

interface Props {
	prefix: string
	values: string[]
	onClear: () => void
}

export const FilterChip = ({ prefix, values, onClear }: Props) => {
	const { text, hidden } = fitChipLabel(prefix, values)

	return (
		<Chip
			onDelete={onClear}
			label={
				<Box component='span' sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
					<Box
						component='span'
						sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
					>
						{text}
					</Box>
					{hidden > 0 && (
						<Box component='span' sx={{ flexShrink: 0 }}>
							+{hidden}
						</Box>
					)}
				</Box>
			}
			sx={{
				height: 32,
				maxWidth: CHIP_MAX_WIDTH,
				bgcolor: '#ddf4ff',
				color: '#0969da',
				fontWeight: 500,
				fontSize: '0.875rem',
				'& .MuiChip-label': { display: 'flex', minWidth: 0, overflow: 'hidden' },
				'& .MuiChip-deleteIcon': { color: '#0969da', fontSize: 18 },
			}}
		/>
	)
}