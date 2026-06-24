import { GroupsView } from '@/features/groups/pages/GroupsView'
import { Box } from '@mui/material'

export default function GroupsPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<GroupsView />
		</Box>
	)
}
