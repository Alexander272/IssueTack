import { Box } from '@mui/material'

import { Users } from '@/features/access/components/Users/Users'

export default function UsersPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Users />
		</Box>
	)
}
