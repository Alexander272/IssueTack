import { Box } from '@mui/material'

import { SettingsPage } from '@/features/equipment/pages/SettingPage'

export default function EquipmentSettings() {
	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<SettingsPage />
		</Box>
	)
}
