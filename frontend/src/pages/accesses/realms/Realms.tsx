import { Box } from '@mui/material'

import { Realms } from '@/features/access/components/Realms/Realms'

export default function RealmsPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Realms />
		</Box>
	)
}
