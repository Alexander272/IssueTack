import type { FC } from 'react'
import { Box, TableCell, TableRow, Typography } from '@mui/material'
import { ChevronDownIcon } from 'lucide-mui'

import type { IEquipmentTreeNode } from '../../types/equipment'
import { EQUIPMENT_TABLE_COLUMNS } from './EquipmentTable'

type Props = {
	items: IEquipmentTreeNode[]
	expanded: boolean
	onToggle: () => void
}

// Строка «партии» складских единиц — в стиле групповых строк таблицы заявок.
export const EquipmentGroupRow: FC<Props> = ({ items, expanded, onToggle }) => {
	const sample = items[0]
	const count = items.length
	const invs = items.map(i => i.inventoryNumber).filter((v): v is string => Boolean(v)).sort()
	const invRange =
		invs.length > 1 ? `Инв. №${invs[0]}–${invs[invs.length - 1]}` : `Инв. №${invs[0] ?? '—'}`

	return (
		<TableRow
			sx={{
				bgcolor: '#f9fafb',
				cursor: 'pointer',
				borderBottom: '1px solid #f3f4f6',
				'&:hover': { bgcolor: '#f3f4f6' },
			}}
			onClick={onToggle}
		>
			<TableCell colSpan={EQUIPMENT_TABLE_COLUMNS} sx={{ px: 3, py: 2 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
					<ChevronDownIcon
						sx={{
							fontSize: 10,
							color: '#6b7280',
							transform: expanded ? 'rotate(0deg)' : 'rotate(-90deg)',
							transition: 'transform 0.2s',
							flexShrink: 0,
						}}
					/>
					<Typography
						sx={{
							fontWeight: 700,
							color: '#374151',
							fontSize: '0.75rem',
							textTransform: 'uppercase',
							letterSpacing: '0.05em',
						}}
						noWrap
					>
						{sample.name}
					</Typography>
					<Typography sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 500, flexShrink: 0 }}>
						({count})
					</Typography>
					<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }} noWrap>
						{invRange} · {count} шт. на складе
					</Typography>
				</Box>
			</TableCell>
		</TableRow>
	)
}
