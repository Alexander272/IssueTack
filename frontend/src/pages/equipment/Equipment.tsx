import { Box, Tab, Tabs } from '@mui/material'
import { useState } from 'react'

import { EquipmentView } from '@/features/equipment/pages/EquipmentView'
import { EquipmentTypesSection } from '@/features/equipment/components/EquipmentTypesSection'
import { EquipmentActsView } from '@/features/equipment/components/EquipmentActsView'
import { EquipmentTemplatesSection } from '@/features/equipment/components/EquipmentTemplatesSection'
import { EquipmentImportSection } from '@/features/equipment/components/EquipmentImportSection'

export default function Equipment() {
	const [tab, setTab] = useState(0)
	const [settingsTab, setSettingsTab] = useState(0)

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
			<Tabs
				value={tab}
				onChange={(_e, value) => setTab(value)}
				textColor='primary'
				indicatorColor='primary'
				sx={{ px: 3, pt: 1 }}
			>
				<Tab label='Единицы' />
				<Tab label='Настройки' />
				<Tab label='Акты' />
			</Tabs>

			{tab === 0 && <EquipmentView />}
			{tab === 1 && (
				<Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
					<Box sx={{ px: 3, borderBottom: '1px solid #e5e7eb', bgcolor: '#fff' }}>
						<Tabs
							value={settingsTab}
							onChange={(_e, v) => setSettingsTab(v)}
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
						{settingsTab === 0 && <EquipmentTypesSection />}
						{settingsTab === 1 && <EquipmentTemplatesSection />}
						{settingsTab === 2 && <EquipmentImportSection />}
					</Box>
				</Box>
			)}
			{tab === 2 && <EquipmentActsView />}
		</Box>
	)
}