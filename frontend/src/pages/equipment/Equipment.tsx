import { Box } from '@mui/material'

import { EquipmentPage } from '@/features/equipment/pages/EquipmentPage'

export default function Equipment() {
	// Обёртка со скроллом — как у остальных страниц (`Tasks`, `Home`, …): без неё
	// каркас `Layout` (Stack с `overflow: hidden`) обрезает содержимое, а flex-элемент
	// с `min-width: auto` не даёт странице сузиться уже своей минимальной ширины —
	// на узком окне не появляется ни скролла, ни сжатия.
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<EquipmentPage />
		</Box>
	)
}
