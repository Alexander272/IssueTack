import { useState, type FC } from 'react'
import { Box, Button, IconButton, Stack, Typography } from '@mui/material'
import { PencilIcon, XIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { HOLDER_DERIVED_STATUSES, getEquipmentStatus } from '../../types/equipment'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentDialog, EquipmentKitDialog } from '../Dialogs'
import { EquipmentTypeIcon } from '../EquipmentTypeIcon'
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
	// Чипы в шапке — про исключение, а не пересказ карточки: «на складе»/«в
	// эксплуатации» видно в блоке «Держатель» ниже, а ремонт, дефект, резерв и
	// списание больше нигде в карточке не видны — их оставляем.
	const status = HOLDER_DERIVED_STATUSES.includes(unit.status) ? null : getEquipmentStatus(unit.status)

	return (
		<>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: { xs: 1.5, sm: 2 },
					// На телефоне шапка переносится: название и крестик остаются в первой
					// строке, а «Редактировать» уходит отдельной строкой во всю ширину
					// (иначе кнопка сжимала бы название до пары букв). На широком экране
					// порядок и вид прежние: `order`/`flexBasis` сбрасываются.
					flexWrap: { xs: 'wrap', sm: 'nowrap' },
				}}
			>
				<Box
					sx={{
						// На телефоне иконку типа прячем целиком: место в первой строке
						// нужнее названию и статусам, а сам тип виден в характеристиках.
						display: { xs: 'none', sm: 'flex' },
						width: 48,
						height: 48,
						borderRadius: '12px',
						bgcolor: `${accent}1a`,
						color: accent,
						alignItems: 'center',
						justifyContent: 'center',
						flexShrink: 0,
					}}
				>
					<EquipmentTypeIcon isKit={unit.isKit} icon={unit.typeIcon} typeName={unit.typeName} sx={{ fontSize: 24 }} />
				</Box>
				<Box sx={{ flexGrow: 1, minWidth: 0, flexBasis: { xs: 0, sm: 'auto' } }}>
					<Stack direction='row' spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
						<Typography variant='h6' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
							{unit.name}
						</Typography>
						{status && <StatusChip label={status.label} color={status.color} />}
						{unit.isKit && <StatusChip label='Комплект' color={accent} />}
					</Stack>
					{/* Тип, инвентарный и серийный номера на телефоне скрыты: строка
					    отжимала название, а те же номера есть в характеристиках.
					    У комплекта типа нет — его имя в подпись не подмешиваем
					    (тип мог остаться только от старых данных). */}
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.25, display: { xs: 'none', sm: 'block' } }}>
						{unit.isKit ? '' : unit.typeName}
						{unit.inventoryNumber ? ` · Инв. №${unit.inventoryNumber}` : ''}
						{unit.serialNumber ? ` · S/N: ${unit.serialNumber}` : ''}
					</Typography>
				</Box>
				{canWrite && (
					<Stack
						direction='row'
						spacing={1}
						sx={{
							flexWrap: 'wrap',
							gap: 1,
							alignItems: 'center',
							order: { xs: 3, sm: 0 },
							flexBasis: { xs: '100%', sm: 'auto' },
						}}
					>
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
				<IconButton
					size='large'
					onClick={onClose}
					sx={{ color: '#9ca3af', order: { xs: 2, sm: 0 }, flexShrink: 0 }}
				>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Box>

			{/* Условный рендер — форма монтируется заново при каждом открытии,
			    чтобы не показывать значения предыдущего элемента. Комплект
			    правится той же формой, что создавался (EquipmentKitDialog),
			    единица — EquipmentDialog. */}
			{editOpen &&
				(unit.isKit ? (
					<EquipmentKitDialog open onClose={() => setEditOpen(false)} onDeleted={onClose} unit={unit} />
				) : (
					<EquipmentDialog open onClose={() => setEditOpen(false)} onDeleted={onClose} unit={unit} />
				))}
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
