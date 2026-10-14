import type { FC } from 'react'
import { Box, Typography } from '@mui/material'
import { WarehouseIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'

type Props = {
	unit: IEquipment
}

// Блок «Держатель»: аватар с инициалами и площадкой, либо состояние «На складе».
export const HolderCard: FC<Props> = ({ unit }) => {
	const accent = equipmentAccent(unit)
	const holder = unit.holderFullName || unit.holderName || ''
	const hasHolder = Boolean(unit.holderUserId || unit.holderName)
	const initials = holder
		.split(' ')
		.map(s => s[0])
		.join('')
		.slice(0, 2)

	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 2, bgcolor: '#f9fafb', borderRadius: '12px', p: 2 }}>
			{hasHolder ? (
				<Box
					sx={{
						width: 44,
						height: 44,
						borderRadius: '50%',
						bgcolor: accent,
						color: '#fff',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						fontWeight: 600,
						flexShrink: 0,
					}}
				>
					{initials}
				</Box>
			) : (
				<Box
					sx={{
						width: 44,
						height: 44,
						borderRadius: '50%',
						bgcolor: '#e5e7eb',
						color: '#6b7280',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						flexShrink: 0,
					}}
				>
					<WarehouseIcon sx={{ fontSize: 20 }} />
				</Box>
			)}
			<Box sx={{ minWidth: 0 }}>
				<Typography sx={{ fontWeight: 600, color: '#1f2937' }}>{hasHolder ? holder : 'На складе'}</Typography>
				<Typography variant='body2' sx={{ color: '#6b7280' }}>
					{unit.siteName || unit.location || '—'}
				</Typography>
			</Box>
		</Box>
	)
}
