import type { FC } from 'react'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { PencilIcon, Trash2Icon } from 'lucide-mui'

import type { IEquipmentType } from '../../types/equipment'
import { EQUIPMENT_TYPE_KIND_LABELS } from '../../types/equipment'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { BADGE_STYLES, getIcon, getIconBg } from './typeIcons'

type Props = {
	type: IEquipmentType
	count: number
	onEdit: (type: IEquipmentType) => void
}

// Строка списка типов: иконка, название, бейдж категории, число единиц и
// действия. Удаление пока недоступно (тип может использоваться единицами).
// Прав на запись нужны только здесь, поэтому строка берёт их сама.
export const EquipmentTypeRow: FC<Props> = ({ type, count, onEdit }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

	return (
		<Box
			sx={{
				display: 'grid',
				gridTemplateColumns: 'repeat(12, 1fr)',
				gap: 2,
				px: 3,
				py: 2,
				alignItems: 'center',
				borderBottom: '1px solid #f3f4f6',
				'&:last-child': { borderBottom: 'none' },
				'&:hover': { bgcolor: '#f9fafb' },
			}}
		>
			<Box sx={{ gridColumn: 'span 5', display: 'flex', alignItems: 'center', gap: 1.5 }}>
				<Box
					sx={{
						width: 40,
						height: 40,
						borderRadius: '8px',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						bgcolor: getIconBg(type),
					}}
				>
					{getIcon(type)}
				</Box>
				<Box>
					<Typography variant='body2' sx={{ fontWeight: 600, color: '#111827' }}>
						{type.name}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						Создан {new Date(type.createdAt).toLocaleDateString('ru-RU')}
					</Typography>
				</Box>
			</Box>

			<Box sx={{ gridColumn: 'span 3' }}>
				<Box
					component='span'
					sx={{
						display: 'inline-flex',
						alignItems: 'center',
						px: 1,
						py: 0.5,
						borderRadius: '4px',
						fontSize: '11px',
						fontWeight: 500,
						...BADGE_STYLES[type.kind],
					}}
				>
					{EQUIPMENT_TYPE_KIND_LABELS[type.kind]}
				</Box>
			</Box>

			<Box sx={{ gridColumn: 'span 2' }}>
				<Typography variant='body2' sx={{ color: '#374151' }}>
					{count} {pluralize(count, ['единица', 'единицы', 'единиц'])}
				</Typography>
			</Box>

			<Box sx={{ gridColumn: 'span 2', textAlign: 'right' }}>
				{canWrite && (
					<>
						<Tooltip title='Редактировать'>
							<IconButton size='small' onClick={() => onEdit(type)}>
								<PencilIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
							</IconButton>
						</Tooltip>
						<Tooltip title='Нельзя удалить: используется'>
							<span>
								<IconButton size='small' disabled>
									<Trash2Icon sx={{ fontSize: 16, color: '#9ca3af' }} />
								</IconButton>
							</span>
						</Tooltip>
					</>
				)}
			</Box>
		</Box>
	)
}
