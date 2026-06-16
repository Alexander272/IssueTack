import { RouterProvider } from 'react-router'
import { Box } from '@mui/material'

import { useRefresh } from '@/features/auth/hooks/refresh'
import { Fallback } from '@/components/Fallback/Fallback'
import { router } from './router'

export const AppRouter = () => {
	const { ready } = useRefresh()

	if (!ready)
		return (
			<Box sx={{ height: '100vh' }}>
				<Fallback />
			</Box>
		)

	return <RouterProvider router={router} />
}
