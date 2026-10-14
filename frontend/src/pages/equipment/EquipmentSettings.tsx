import { Box, Tab, Tabs } from '@mui/material'
import { useState } from 'react'

import {
	EquipmentImportSection,
	EquipmentTemplatesSection,
	EquipmentTypesSection,
} from '@/features/equipment/components/Settings'

export default function EquipmentSettings() {
	const [tab, setTab] = useState(0)

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
			<Box sx={{ px: 3, borderBottom: '1px solid #e5e7eb', bgcolor: '#fff' }}>
				<Tabs
					value={tab}
					onChange={(_e, v) => setTab(v)}
					variant='standard'
					sx={{
						'& .MuiTab-root': { minHeight: 48, textTransform: 'none', fontWeight: 500 },
					}}
				>
					<Tab label='Типы оборудования' />
					<Tab label='Шаблоны комплектов' />
					<Tab label='Импорт' />
				</Tabs>
			</Box>
			<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
				{tab === 0 && <EquipmentTypesSection />}
				{tab === 1 && <EquipmentTemplatesSection />}
				{tab === 2 && <EquipmentImportSection />}
			</Box>
		</Box>
	)
}
