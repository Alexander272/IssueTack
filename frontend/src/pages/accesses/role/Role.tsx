import { Box } from '@mui/material'

import { Role } from '@/features/access/components/Role/Role'

export default function RolePage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Role />
		</Box>
	)
}
