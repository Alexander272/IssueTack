import { type FC, useState } from 'react'
import { useMatch, useNavigate } from 'react-router'
import { Box } from '@mui/material'

import { AppRoutes } from '@/pages/router/routes'
import { EquipmentDialog } from '../components/Dialogs'
import { EquipmentDetailModal } from '../components/Detail'
import { EquipmentList } from '../components/List'
import { Header } from '../components/Header'

// Страница оборудования: состояние открытой карточки и модалки. Список (фильтры,
// ленивое дерево, серверная пагинация) живёт в EquipmentList и грузит данные сам.
export const EquipmentPage: FC = () => {
	const navigate = useNavigate()

	// Карточка единицы открывается локально: клик по строке или по связанной
	// единице в модалке URL не меняет. Роут /equipment/:id — только вход для
	// deep-link (ссылка на оборудование из заявки, прямая ссылка): он задаёт
	// openId до тех пор, пока пользователь не выбрал карточку сам.
	const match = useMatch({ path: AppRoutes.EquipmentDetail, end: true })
	const deepLinkId = match?.params.id ?? null
	const [pickedId, setPickedId] = useState<string | null>(null)
	const openId = pickedId ?? deepLinkId

	const closeDetail = () => {
		setPickedId(null)
		// Если карточку открыли по ссылке, убираем её из адреса — иначе закрытие
		// карточки на возврате на страницу снова открыло бы её.
		if (deepLinkId) navigate(AppRoutes.Equipment, { replace: true })
	}

	const [dialogOpen, setDialogOpen] = useState(false)
	const [dialogKit, setDialogKit] = useState(false)
	const [dialogSeq, setDialogSeq] = useState(0)

	const openCreate = (kit: boolean) => {
		setDialogKit(kit)
		setDialogSeq(s => s + 1)
		setDialogOpen(true)
	}

	return (
		<Box sx={{ p: 3 }}>
			<Header onCreate={openCreate} />

			<EquipmentList onOpen={setPickedId} />

			<EquipmentDialog
				key={dialogSeq}
				open={dialogOpen}
				onClose={() => setDialogOpen(false)}
				unit={null}
				isKitDefault={dialogKit}
			/>

			<EquipmentDetailModal
				key={openId ?? 'closed'}
				open={Boolean(openId)}
				unitId={openId}
				onClose={closeDetail}
				onOpenUnit={setPickedId}
			/>
		</Box>
	)
}
