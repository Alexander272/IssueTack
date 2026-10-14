import type { FC } from 'react'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { ArrowUpRightIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentKindIcon } from '../EquipmentKindIcon'

type Props = {
	unit: IEquipment
	onOpen: (id: string) => void
}

// Карточка связанной единицы (хозяин или член состава). Открытие делегируется
// контейнеру: карточка открывается на месте, без перехода на /equipment/:id.
export const MemberCard: FC<Props> = ({ unit, onOpen }) => {
	const accent = equipmentAccent(unit)

	return (
		<Box
			sx={{
				display: 'flex',
				alignItems: 'center',
				gap: 2,
				p: 1.5,
				border: '1px solid #eef2f6',
				borderRadius: '12px',
				bgcolor: '#fff',
			}}
		>
			<Box
				sx={{
					width: 36,
					height: 36,
					borderRadius: '10px',
					bgcolor: `${accent}1a`,
					color: accent,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					flexShrink: 0,
				}}
			>
				<EquipmentKindIcon isKit={unit.isKit} kind={unit.typeKind} sx={{ fontSize: 18 }} />
			</Box>
			<Box sx={{ flexGrow: 1, minWidth: 0 }}>
				<Typography sx={{ fontWeight: 500, color: '#1f2937' }} noWrap>
					{unit.name}
				</Typography>
				<Typography variant='caption' sx={{ color: '#6b7280' }} noWrap>
					{unit.typeName} · Инв. №{unit.inventoryNumber ?? '—'}
				</Typography>
			</Box>
			<Tooltip title='Открыть'>
				<IconButton size='small' onClick={() => onOpen(unit.id)} sx={{ color: accent }}>
					<ArrowUpRightIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Tooltip>
		</Box>
	)
}
