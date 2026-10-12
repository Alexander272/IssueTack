import { Box, Tab, Tabs } from '@mui/material'
import { useState } from 'react'

import { EquipmentView } from '@/features/equipment/pages/EquipmentView'
import { EquipmentTypesSection } from '@/features/equipment/components/EquipmentTypesSection'

export default function Equipment() {
	const [tab, setTab] = useState(0)

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Tabs
				value={tab}
				onChange={(_e, value) => setTab(value)}
				textColor='primary'
				indicatorColor='primary'
				sx={{ px: 3, pt: 1 }}
			>
				<Tab label='Единицы' />
				<Tab label='Типы' />
			</Tabs>
			{tab === 0 ? <EquipmentView /> : <EquipmentTypesSection />}
		</Box>
	)
}