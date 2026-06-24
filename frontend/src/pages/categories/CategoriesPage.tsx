import { Box } from '@mui/material'

import { CategoriesView } from '@/features/categories/pages/CategoriesView'

export default function CategoriesPage() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<CategoriesView />
		</Box>
	)
}
