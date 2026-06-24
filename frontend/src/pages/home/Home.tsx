import { Box } from '@mui/material'

import { TaskList } from '@/features/tasks/pages/TaskList'

export default function Home() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<TaskList />
		</Box>
	)
}
