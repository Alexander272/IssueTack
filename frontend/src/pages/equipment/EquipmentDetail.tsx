import { Box } from '@mui/material'
import { EquipmentDetailView } from '@/features/equipment/pages/EquipmentDetailView'

export default function EquipmentDetail() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<EquipmentDetailView />
		</Box>
	)
}