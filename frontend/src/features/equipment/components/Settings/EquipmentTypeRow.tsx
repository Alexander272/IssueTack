import type { FC } from 'react'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { PencilIcon, Trash2Icon } from 'lucide-mui'

import type { IEquipmentType } from '../../types/equipment'
import { EQUIPMENT_TYPE_KIND_LABELS } from '../../types/equipment'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { resolveTypeIcon } from '../../typeIcons'
import { BADGE_STYLES } from './typeGroups'

type Props = {
	type: IEquipmentType
	count: number
	onEdit: (type: IEquipmentType) => void
	// mobile — на телефоне строка рисуется карточкой (см. ниже). Режим приходит
	// пропсом из секции, чтобы не подписываться на media query в каждой строке.
	mobile?: boolean
}

// Строка списка типов: иконка, название, бейдж категории, число единиц и
// действия. Удаление пока недоступно (тип может использоваться единицами).
// Прав на запись нужны только здесь, поэтому строка берёт их сама. На узком
// экране 12-колоночная сетка нечитаема — там та же строка становится карточкой:
// действия в углу, категория и число единиц отдельной строкой под названием.
export const EquipmentTypeRow: FC<Props> = ({ type, count, onEdit, mobile = false }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

	const { Icon, color, bg } = resolveTypeIcon({ icon: type.icon, typeName: type.name })
	const icon = (
		<Box
			sx={{
				width: 40,
				height: 40,
				borderRadius: '8px',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				bgcolor: bg,
				flexShrink: 0,
			}}
		>
			<Icon sx={{ color }} />
		</Box>
	)

	const name = (
		<Box sx={{ minWidth: 0 }}>
			<Typography variant='body2' sx={{ fontWeight: 600, color: '#111827' }}>
				{type.name}
			</Typography>
			<Typography variant='caption' sx={{ color: '#6b7280' }}>
				Создан {new Date(type.createdAt).toLocaleDateString('ru-RU')}
			</Typography>
		</Box>
	)

	const badge = (
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
	)

	const countLabel = (
		<Typography variant='body2' sx={{ color: '#374151' }}>
			{count} {pluralize(count, ['единица', 'единицы', 'единиц'])}
		</Typography>
	)

	const actions = (
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
	)

	if (mobile) {
		return (
			<Box
				sx={{
					display: 'flex',
					alignItems: 'flex-start',
					gap: 1.5,
					px: 2,
					py: 1.5,
					borderBottom: '1px solid #f3f4f6',
					'&:last-child': { borderBottom: 'none' },
				}}
			>
				{icon}

				<Box sx={{ flexGrow: 1, minWidth: 0 }}>
					{name}
					<Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 1, mt: 0.75 }}>
						{badge}
						{countLabel}
					</Box>
				</Box>

				{canWrite && (
					<Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0, ml: -0.5 }}>{actions}</Box>
				)}
			</Box>
		)
	}

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
				{icon}
				{name}
			</Box>

			<Box sx={{ gridColumn: 'span 3' }}>{badge}</Box>

			<Box sx={{ gridColumn: 'span 2' }}>{countLabel}</Box>

			<Box sx={{ gridColumn: 'span 2', textAlign: 'right' }}>{canWrite && actions}</Box>
		</Box>
	)
}
