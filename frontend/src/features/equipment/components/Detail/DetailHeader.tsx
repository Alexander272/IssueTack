import { useState, type FC } from 'react'
import { Box, Button, IconButton, Stack, Typography } from '@mui/material'
import { PencilIcon, XIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { getEquipmentStatus } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentDialog } from '../Dialogs'
import { EquipmentKindIcon } from '../EquipmentKindIcon'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

type Props = {
	unit: IEquipment
	onClose: () => void
}

// Шапка модалки единицы: иконка типа и название со статусом, действие
// (редактировать) и кнопка закрытия. Выдача и возврат живут в блоке «Держатель»
// на вкладке «Обзор» — там видно, у кого единица сейчас, — а удаление переехало
// в форму редактирования, рядом с сохранением. Диалог открывается здесь же,
// рядом со своей кнопкой, поэтому наружу уходят только единица и закрытие
// модалки — мостиков-колбэков в контейнер нет. Прав на запись нужны только
// здесь, поэтому компонент берёт их сам.
export const DetailHeader: FC<Props> = ({ unit, onClose }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const [editOpen, setEditOpen] = useState(false)

	const accent = equipmentAccent(unit)
	const status = getEquipmentStatus(unit.status)

	return (
		<>
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
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
						<Button
							variant='outlined'
							size='small'
							sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
							onClick={() => setEditOpen(true)}
						>
							<PencilIcon sx={{ fontSize: 16 }} />
							Редактировать
						</Button>
					</Stack>
				)}
				<IconButton size='large' onClick={onClose} sx={{ color: '#9ca3af' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Box>

			{/* Условный рендер — форма монтируется заново при каждом открытии,
			    чтобы не показывать значения предыдущего элемента. */}
			{editOpen && <EquipmentDialog open onClose={() => setEditOpen(false)} onDeleted={onClose} unit={unit} />}
		</>
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
