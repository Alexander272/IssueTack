import { type FC, useState } from 'react'
import { Box, Dialog, DialogContent, Tab, Tabs, useMediaQuery, useTheme } from '@mui/material'

import { Loader } from '@/components/Fallback/Loader'
import { useGetEquipmentQuery } from '../../equipmentApiSlice'
import { HistoryTimeline } from './HistoryTimeline'
import { ItemActs } from './ItemActs'
import { DetailHeader } from './DetailHeader'
import { OverviewTab } from './OverviewTab'

type Props = {
	open: boolean
	unitId: string | null
	onClose: () => void
	// onOpenUnit — открыть карточку связанной единицы на месте (без смены URL).
	onOpenUnit: (id: string) => void
}

// Модалка единицы оборудования. Контейнер держит данные единицы и вкладки; кнопки
// и диалоги действий живут в шапке, сами вкладки — презентационные компоненты.
export const DetailModal: FC<Props> = ({ open, unitId, onClose, onOpenUnit }) => {
	const theme = useTheme()
	// На телефоне карточка раскрывается на весь экран: в окне по центру (maxWidth='lg')
	// шапке с вкладками и содержимым не хватает ширины, а сама карточка — это
	// полноценный экран единицы со скроллом внутри, а не подтверждение действия.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const [tab, setTab] = useState(0)

	const { data, isLoading } = useGetEquipmentQuery(unitId ?? '', { skip: !unitId })
	const unit = data?.data

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='lg'
			slotProps={{
				paper: {
					sx: {
						// Скругление у окна во весь экран срезало бы углы у краёв экрана
						// и не читалось — на телефоне его убираем, на планшете оставляем.
						borderRadius: { xs: 0, sm: '16px' },
						backgroundImage: 'none',
					},
				},
			}}
		>
			{!unit ? (
				<Box sx={{ minHeight: 320, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
					{isLoading ? <Loader /> : null}
				</Box>
			) : (
				<>
					<Box sx={{ px: { xs: 2, sm: 3 }, pt: { xs: 1.5, sm: 2.5 } }}>
						<DetailHeader unit={unit} onClose={onClose} />

						<Tabs
							value={tab}
							onChange={(_e, value) => setTab(value)}
							textColor='primary'
							indicatorColor='primary'
							// На узком экране вкладки уезжают в скролл, а не сжимаются
							// до нечитаемых подписей; на широком порядок прежний.
							variant={isMobile ? 'scrollable' : 'standard'}
							scrollButtons='auto'
							allowScrollButtonsMobile
							sx={{ mt: { xs: 1.5, sm: 2 }, minHeight: 40, borderBottom: '1px solid #eef2f6' }}
						>
							<Tab label='Обзор' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='История' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='Акты' sx={{ textTransform: 'none', minHeight: 40 }} />
						</Tabs>
					</Box>

					<DialogContent
						dividers
						sx={{
							minHeight: { xs: 240, sm: 360 },
							bgcolor: '#fff',
							px: { xs: 2, sm: 3 },
							py: { xs: 2, sm: 2.5 },
						}}
					>
						{tab === 0 && <OverviewTab unit={unit} onOpenUnit={onOpenUnit} />}
						{tab === 1 && <HistoryTimeline unit={unit} />}
						{tab === 2 && <ItemActs unit={unit} />}
					</DialogContent>
				</>
			)}
		</Dialog>
	)
}
