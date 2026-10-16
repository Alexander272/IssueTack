import type { FC } from 'react'
import { Box, Chip, IconButton, Tooltip, Typography } from '@mui/material'
import { LayersIcon, PencilIcon, Trash2Icon } from 'lucide-mui'

import type { IEquipmentKitTemplate } from '../../types/equipment'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'

type Props = {
	template: IEquipmentKitTemplate
	onEdit: (template: IEquipmentKitTemplate) => void
	onDelete: (template: IEquipmentKitTemplate) => void
	// mobile — на телефоне строка рисуется карточкой (см. ниже). Режим приходит
	// пропсом из секции, чтобы не подписываться на media query в каждой строке.
	mobile?: boolean
}

// Строка шаблона комплекта: иконка, название с описанием, состав (типы) и
// действия. Позиции шаблона — типы оборудования, поэтому их подписи приходят с
// сервера (typeName) и не требуют загрузки справочника типов в этой секции.
export const EquipmentTemplateRow: FC<Props> = ({ template, onEdit, onDelete, mobile = false }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

	const items = template.items ?? []
	const count = items.length

	const icon = (
		<Box
			sx={{
				width: 40,
				height: 40,
				borderRadius: '8px',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				bgcolor: '#eef2ff',
				flexShrink: 0,
			}}
		>
			<LayersIcon sx={{ color: '#4338ca' }} />
		</Box>
	)

	const name = (
		<Box sx={{ minWidth: 0 }}>
			<Typography variant='body2' sx={{ fontWeight: 600, color: '#111827' }}>
				{template.name}
			</Typography>
			<Typography
				variant='caption'
				sx={{
					color: '#6b7280',
					display: '-webkit-box',
					WebkitLineClamp: 1,
					WebkitBoxOrient: 'vertical',
					overflow: 'hidden',
				}}
			>
				{template.description || 'Без описания'}
			</Typography>
		</Box>
	)

	const composition = (
		<Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
			{count === 0 ? (
				<Typography variant='caption' sx={{ color: '#9ca3af' }}>
					Пустой состав
				</Typography>
			) : (
				items.slice(0, 4).map(item => (
					<Chip
						key={item.id}
						label={item.typeName}
						size='small'
						sx={{ bgcolor: '#f3f4f6', color: '#374151', fontSize: 11, height: 22 }}
					/>
				))
			)}
			{count > 4 && (
				<Typography variant='caption' sx={{ color: '#6b7280' }}>
					+{count - 4}
				</Typography>
			)}
		</Box>
	)

	const countLabel = (
		<Typography variant='body2' sx={{ color: '#374151' }}>
			{count} {pluralize(count, ['позиция', 'позиции', 'позиций'])}
		</Typography>
	)

	const actions = (
		<>
			<Tooltip title='Редактировать'>
				<IconButton size='small' onClick={() => onEdit(template)}>
					<PencilIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
				</IconButton>
			</Tooltip>
			<Tooltip title='Удалить'>
				<IconButton size='small' onClick={() => onDelete(template)}>
					<Trash2Icon sx={{ fontSize: 16, color: '#9ca3af' }} />
				</IconButton>
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
						{countLabel}
					</Box>
					<Box sx={{ mt: 0.75 }}>{composition}</Box>
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
			<Box sx={{ gridColumn: 'span 4', display: 'flex', alignItems: 'center', gap: 1.5 }}>
				{icon}
				{name}
			</Box>

			<Box sx={{ gridColumn: 'span 4' }}>{composition}</Box>

			<Box sx={{ gridColumn: 'span 2' }}>{countLabel}</Box>

			<Box sx={{ gridColumn: 'span 2', textAlign: 'right' }}>{canWrite && actions}</Box>
		</Box>
	)
}
