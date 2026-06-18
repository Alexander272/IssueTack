import { Box } from '@mui/material'

import { Permissions } from '@/features/access/components/Permissions'

export default function PermissionsPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Permissions />
		</Box>
	)
}
