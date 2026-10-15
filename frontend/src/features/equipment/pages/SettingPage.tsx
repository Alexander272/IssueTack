import { useState } from 'react'
import { Box, Tab, Tabs } from '@mui/material'

import { EquipmentTemplatesSection, EquipmentTypesSection, Header } from '../components/Settings'

export const SettingsPage = () => {
	const [tab, setTab] = useState(0)

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', bgcolor: '#f9fafb' }}>
			<Box sx={{ maxWidth: 'xl', mx: 'auto', p: 3 }}>
				<Box
					sx={{
						gap: 2,
						bgcolor: 'white',
						borderRadius: '12px',
						border: '1px solid #e5e7eb',
						overflow: 'hidden',
						px: 2,
						mb: 3,
					}}
				>
					<Header />

					<Box sx={{ mt: 1 }}>
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
						</Tabs>
					</Box>
				</Box>

				<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
					{tab === 0 && <EquipmentTypesSection />}
					{tab === 1 && <EquipmentTemplatesSection />}
				</Box>
			</Box>
		</Box>
	)
}
