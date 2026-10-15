import { useState, type FC, type ReactNode } from 'react'
import { Box, Button, Stack, Typography } from '@mui/material'
import { PlusIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { canBePart } from '../../types/equipment'
import {
	useGetEquipmentChildrenQuery,
	useGetEquipmentCompositionQuery,
	useGetEquipmentQuery,
} from '../../equipmentApiSlice'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { EquipmentCompositionDialog, EquipmentPartsDialog } from '../Dialogs'
import { EquipmentTasks } from './EquipmentTasks'
import { HolderCard } from './HolderCard'
import { MemberCard } from './MemberCard'

const fmtDate = (d: string | null) => (d ? new Date(`${d}T00:00:00`).toLocaleDateString('ru-RU') : '—')

const labelSx = { color: '#9ca3af', fontSize: 12, fontWeight: 600, textTransform: 'uppercase' as const }

type Props = {
	unit: IEquipment
	onOpenUnit: (id: string) => void
}

// Вкладка «Обзор». Держателя, состав и диалоги состава вкладка держит сама: это её
// собственный контекст, контейнеру модалки они не нужны.
export const EquipmentOverviewTab: FC<Props> = ({ unit, onOpenUnit }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const [compositionOpen, setCompositionOpen] = useState(false)
	const [partsOpen, setPartsOpen] = useState(false)
	const isKit = unit.isKit
	const isUnit = !isKit && !canBePart(unit.typeKind)
	// Установленность — это физический родитель (parent_id) или членство в
	// комплекте (kit_id), а не категория типа: членом комплекта может быть
	// единица любого типа (мышь, сумка), и она тоже показывает, где находится.
	const installed = !isKit && Boolean(unit.parentId || unit.kitId)

	// Для логического члена комплекта физического родителя нет — хозяина ищем по kit_id.
	const { data: parentData } = useGetEquipmentQuery(unit.parentId ?? '', { skip: !unit.parentId })
	const { data: kitHostData } = useGetEquipmentQuery(unit.kitId ?? '', { skip: !unit.kitId })
	const host = parentData?.data ?? kitHostData?.data ?? null

	// У комплектующего/расходника состава быть не может — детей не запрашиваем.
	const { data: childrenData } = useGetEquipmentChildrenQuery(unit.id, { skip: !isUnit })
	const { data: compositionData } = useGetEquipmentCompositionQuery(unit.id, { skip: !isKit })
	const members = isKit ? (compositionData?.data ?? []) : (childrenData?.data ?? [])

	const holderTitle = installed ? (unit.parentId ? 'Установлен в' : 'В составе комплекта') : 'Держатель'

	return (
		<>
			<Stack spacing={3}>
				<SectionBlock title={holderTitle}>
					{installed && host ? <MemberCard unit={host} onOpen={onOpenUnit} /> : <HolderCard unit={unit} />}
				</SectionBlock>

				<SectionBlock title='Характеристики'>
					<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1 }}>
						{(
							[
								['Производитель', unit.vendor || '—'],
								['Модель', unit.model || '—'],
								['Серийный номер', unit.serialNumber ?? '—'],
								['Инвентарный номер', unit.inventoryNumber ?? '—'],
								['Площадка', unit.siteName || '—'],
								['Локация', unit.location ?? '—'],
								['Дата покупки', fmtDate(unit.purchaseDate)],
								['Гарантия до', fmtDate(unit.warrantyUntil)],
								['Стоимость', unit.cost != null ? `${unit.cost.toLocaleString('ru-RU')} ₽` : '—'],
							] as [string, string][]
						).map(([label, value]) => (
							<Box
								key={label}
								sx={{
									display: 'flex',
									justifyContent: 'space-between',
									gap: 2,
									py: 0.75,
									borderBottom: '1px solid #f3f4f6',
								}}
							>
								<Typography sx={{ color: '#6b7280', fontSize: 14 }}>{label}</Typography>
								<Typography sx={{ color: '#1f2937', fontWeight: 500, fontSize: 14 }}>{value}</Typography>
							</Box>
						))}
					</Box>
				</SectionBlock>

				{unit.notes && (
					<SectionBlock title='Примечание'>
						<Typography sx={{ color: '#374151', whiteSpace: 'pre-wrap', fontSize: 14 }}>{unit.notes}</Typography>
					</SectionBlock>
				)}

				{(isKit || isUnit) && (
					<SectionBlock
						title={isKit ? 'Состав комплекта' : 'Физический состав'}
						action={
							canWrite && (
								<Button
									variant='outlined'
									size='small'
									sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
									onClick={() => (isKit ? setCompositionOpen(true) : setPartsOpen(true))}
								>
									<PlusIcon sx={{ fontSize: 16 }} />
									Изменить состав
								</Button>
							)
						}
					>
						{members.length === 0 ? (
							<EmptyHint>
								{isKit ? 'Комплект пуст — состав пока не добавлен' : 'Составных частей нет'}
							</EmptyHint>
						) : (
							<Stack spacing={1}>
								{members.map(m => (
									<MemberCard key={m.id} unit={m} onOpen={onOpenUnit} />
								))}
							</Stack>
						)}
					</SectionBlock>
				)}

				<SectionBlock title='Заявки по оборудованию'>
					<EquipmentTasks unitId={unit.id} />
				</SectionBlock>
			</Stack>

			{/* Диалоги состава открываются из вкладки — её собственный контекст. */}
			{isKit && compositionOpen && (
				<EquipmentCompositionDialog open onClose={() => setCompositionOpen(false)} unit={unit} />
			)}
			{!isKit && partsOpen && (
				<EquipmentPartsDialog open onClose={() => setPartsOpen(false)} unit={unit} />
			)}
		</>
	)
}

const SectionBlock: FC<{ title: string; action?: ReactNode; children: ReactNode }> = ({
	title,
	action,
	children,
}) => (
	<Box>
		<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
			<Typography sx={labelSx}>{title}</Typography>
			{action}
		</Box>
		{children}
	</Box>
)

const EmptyHint: FC<{ children: ReactNode }> = ({ children }) => (
	<Box sx={{ p: 2, color: '#6b7280', fontSize: 14, border: '1px dashed #e5e7eb', borderRadius: '12px' }}>
		{children}
	</Box>
)
