import type { FC } from 'react'
import { Box, Button, IconButton, Stack, Typography } from '@mui/material'
import { PackageCheckIcon, PackageMinusIcon, PencilIcon, Trash2Icon, XIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { getEquipmentStatus } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentKindIcon } from '../EquipmentKindIcon'

type Props = {
	unit: IEquipment
	canWrite: boolean
	isUnassigning: boolean
	onAssign: () => void
	onUnassign: () => void
	onEdit: () => void
	onDelete: () => void
	onClose: () => void
}

// Шапка модалки единицы: иконка типа и название со статусом, действия
// (выдать/вернуть, редактировать, удалить) и кнопка закрытия.
export const EquipmentDetailHeader: FC<Props> = ({
	unit,
	canWrite,
	isUnassigning,
	onAssign,
	onUnassign,
	onEdit,
	onDelete,
	onClose,
}) => {
	const accent = equipmentAccent(unit)
	const status = getEquipmentStatus(unit.status)
	const hasHolder = Boolean(unit.holderUserId || unit.holderName)
	const installed = Boolean(unit.parentId || unit.kitId)

	return (
		<Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
			<Box
				sx={{
					width: 48,
					height: 48,
					borderRadius: '12px',
					bgcolor: `${accent}1a`,
					color: accent,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					flexShrink: 0,
				}}
			>
				<EquipmentKindIcon isKit={unit.isKit} kind={unit.typeKind} sx={{ fontSize: 24 }} />
			</Box>
			<Box sx={{ flexGrow: 1, minWidth: 0 }}>
				<Stack direction='row' spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
					<Typography variant='h6' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						{unit.name}
					</Typography>
					{status && <StatusChip label={status.label} color={status.color} />}
					{unit.isKit && <StatusChip label='Комплект' color={accent} />}
				</Stack>
				<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.25 }}>
					{unit.typeName}
					{unit.inventoryNumber ? ` · Инв. №${unit.inventoryNumber}` : ''}
					{unit.serialNumber ? ` · S/N: ${unit.serialNumber}` : ''}
				</Typography>
			</Box>
			{canWrite && (
				<Stack direction='row' spacing={1} sx={{ flexWrap: 'wrap', gap: 1, alignItems: 'center' }}>
					{/* Установленную единицу (внутри устройства или в комплекте) отдельно не выдают. */}
					{!installed &&
						(hasHolder ? (
							<Button
								variant='outlined'
								size='small'
								disabled={isUnassigning}
								startIcon={<PackageMinusIcon sx={{ fontSize: 16 }} />}
								onClick={onUnassign}
								sx={{ borderRadius: '8px', textTransform: 'none' }}
							>
								Возврат
							</Button>
						) : (
							<Button
								variant='outlined'
								size='small'
								startIcon={<PackageCheckIcon sx={{ fontSize: 16 }} />}
								onClick={onAssign}
								sx={{ borderRadius: '8px', textTransform: 'none' }}
							>
								Выдать
							</Button>
						))}
					<Button
						variant='outlined'
						size='small'
						sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
						onClick={onEdit}
					>
						<PencilIcon sx={{ fontSize: 16 }} />
						Редактировать
					</Button>
					<IconButton size='small' color='error' onClick={onDelete}>
						<Trash2Icon sx={{ fontSize: 18 }} />
					</IconButton>
				</Stack>
			)}
			<IconButton size='small' onClick={onClose} sx={{ color: '#9ca3af' }}>
				<XIcon sx={{ fontSize: 18 }} />
			</IconButton>
		</Box>
	)
}

const StatusChip: FC<{ label: string; color: string }> = ({ label, color }) => (
	<Box
		component='span'
		sx={{
			px: 1,
			py: 0.25,
			borderRadius: '6px',
			fontSize: 12,
			fontWeight: 600,
			bgcolor: `${color}1a`,
			color,
			whiteSpace: 'nowrap',
		}}
	>
		{label}
	</Box>
)
