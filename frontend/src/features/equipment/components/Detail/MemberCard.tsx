import type { FC } from 'react'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowUpRightIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentTypeIcon } from '../EquipmentTypeIcon'

type Props = {
	unit: IEquipment
	onOpen: (id: string) => void
}

// Карточка связанной единицы (хозяин или член состава). Вся карточка — кнопка:
// открытие делегируется вкладке и идёт на месте, без перехода на /equipment/:id.
export const MemberCard: FC<Props> = ({ unit, onOpen }) => {
	const accent = equipmentAccent(unit)

	return (
		<ButtonBase
			onClick={() => onOpen(unit.id)}
			sx={{
				width: '100%',
				display: 'flex',
				alignItems: 'center',
				gap: 1.5,
				p: 1.5,
				borderRadius: '12px',
				border: '1px solid #eef2f6',
				bgcolor: '#f8fafc',
				textAlign: 'left',
				transition: 'background-color .15s, border-color .15s',
				'&:hover': { bgcolor: '#eff6ff', borderColor: '#bfdbfe' },
				'&:hover .member-open': { color: '#2563eb', borderColor: '#bfdbfe' },
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
				<EquipmentTypeIcon isKit={unit.isKit} icon={unit.typeIcon} typeName={unit.typeName} sx={{ fontSize: 18 }} />
			</Box>
			<Box sx={{ flexGrow: 1, minWidth: 0 }}>
				<Typography sx={{ fontSize: 14, fontWeight: 600, color: '#0f172a' }} noWrap>
					{unit.name}
				</Typography>
				<Typography sx={{ fontSize: 12, color: '#64748b' }} noWrap>
					{unit.typeName} · Инв. №{unit.inventoryNumber ?? '—'}
				</Typography>
			</Box>
			<Box
				className='member-open'
				sx={{
					width: 32,
					height: 32,
					borderRadius: '8px',
					border: '1px solid #e2e8f0',
					bgcolor: '#fff',
					color: '#94a3b8',
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					flexShrink: 0,
					transition: 'color .15s, border-color .15s',
				}}
			>
				<ArrowUpRightIcon sx={{ fontSize: 15 }} />
			</Box>
		</ButtonBase>
	)
}
