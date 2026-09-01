import { Box } from '@mui/material'
import { FavoritesPage } from '@/features/favorites/pages/FavoritesPage'

export default function Favorites() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<FavoritesPage />
		</Box>
	)
}
