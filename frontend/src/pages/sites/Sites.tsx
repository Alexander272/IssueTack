import { Box } from '@mui/material'
import { SitesView } from '@/features/sites/pages/SitesView'

export default function Sites() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<SitesView />
		</Box>
	)
}
