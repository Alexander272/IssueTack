import { Box } from '@mui/material'

import { TaskCreateForm } from '@/features/tasks/components/TaskCreateForm'

export default function Home() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', p: 3 }}>
			<TaskCreateForm />
		</Box>
	)
}
