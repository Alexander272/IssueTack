import { type FC, useState } from 'react'
import { Box, Dialog, DialogContent, Tab, Tabs } from '@mui/material'

import { Loader } from '@/components/Fallback/Loader'
import { useGetEquipmentQuery } from '../../equipmentApiSlice'
import { EquipmentHistoryTimeline } from './EquipmentHistoryTimeline'
import { EquipmentItemActs } from './EquipmentItemActs'
import { EquipmentDetailHeader } from './EquipmentDetailHeader'
import { EquipmentOverviewTab } from './EquipmentOverviewTab'

type Props = {
	open: boolean
	unitId: string | null
	onClose: () => void
	// onOpenUnit — открыть карточку связанной единицы на месте (без смены URL).
	onOpenUnit: (id: string) => void
}

// Модалка единицы оборудования. Контейнер держит данные единицы и вкладки; кнопки
// и диалоги действий живут в шапке, сами вкладки — презентационные компоненты.
export const EquipmentDetailModal: FC<Props> = ({ open, unitId, onClose, onOpenUnit }) => {
	const [tab, setTab] = useState(0)

	const { data, isLoading } = useGetEquipmentQuery(unitId ?? '', { skip: !unitId })
	const unit = data?.data

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='lg'
			slotProps={{ paper: { sx: { borderRadius: '16px', backgroundImage: 'none' } } }}
		>
			{!unit ? (
				<Box sx={{ minHeight: 320, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
					{isLoading ? <Loader /> : null}
				</Box>
			) : (
				<>
					<Box sx={{ px: 3, pt: 2.5 }}>
						<EquipmentDetailHeader unit={unit} onClose={onClose} />

						<Tabs
							value={tab}
							onChange={(_e, value) => setTab(value)}
							textColor='primary'
							indicatorColor='primary'
							sx={{ mt: 2, minHeight: 40, borderBottom: '1px solid #eef2f6' }}
						>
							<Tab label='Обзор' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='История' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='Акты' sx={{ textTransform: 'none', minHeight: 40 }} />
						</Tabs>
					</Box>

					<DialogContent dividers sx={{ minHeight: 360, bgcolor: '#fff' }}>
						{tab === 0 && <EquipmentOverviewTab unit={unit} onOpenUnit={onOpenUnit} />}
						{tab === 1 && <EquipmentHistoryTimeline unit={unit} />}
						{tab === 2 && <EquipmentItemActs unit={unit} />}
					</DialogContent>
				</>
			)}
		</Dialog>
	)
}
