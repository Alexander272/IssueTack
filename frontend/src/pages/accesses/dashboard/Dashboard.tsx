import { Box } from '@mui/material'

import { Dashboard } from '@/features/access/components/Dasboard/Dashboard'

export default function DashboardPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Dashboard />
		</Box>
	)
}
