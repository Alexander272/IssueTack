import { Box } from '@mui/material'
import { StatisticsView } from '@/features/statistics/pages/StatisticsView'

export default function Statistics() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<StatisticsView />
		</Box>
	)
}