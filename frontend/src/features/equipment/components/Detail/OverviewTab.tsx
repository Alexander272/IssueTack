import { useState, type FC } from 'react'
import { Box, Button, Stack, Typography } from '@mui/material'
import { PackageIcon, PlusIcon } from 'lucide-mui'

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
import { DetailCard } from './DetailCard'
import { softButtonSx } from './consts'
import { EmptyState } from './EmptyState'
import { EquipmentTasks } from './EquipmentTasks'
import { HolderCard } from './HolderCard'
import { MemberCard } from './MemberCard'

const fmtDate = (d: string | null) => (d ? new Date(`${d}T00:00:00`).toLocaleDateString('ru-RU') : '—')

// Характеристики — плоский список пар; артикулы показываем моноширинной плашкой.
const specsOf = (unit: IEquipment) => [
	{ label: 'Производитель', value: unit.vendor || '—' },
	{ label: 'Модель', value: unit.model || '—' },
	{ label: 'Серийный номер', value: unit.serialNumber ?? '—', mono: true },
	{ label: 'Инвентарный номер', value: unit.inventoryNumber ?? '—', mono: true },
	{ label: 'Площадка', value: unit.siteName || '—' },
	{ label: 'Локация', value: unit.location ?? '—' },
	{ label: 'Дата покупки', value: fmtDate(unit.purchaseDate) },
	{ label: 'Гарантия до', value: fmtDate(unit.warrantyUntil) },
]

type Props = {
	unit: IEquipment
	onOpenUnit: (id: string) => void
}

// Вкладка «Обзор»: карточки держателя, характеристик, состава и заявок. Состав и
// его диалоги вкладка держит сама — это её собственный контекст, контейнеру
// модалки он не нужен.
export const OverviewTab: FC<Props> = ({ unit, onOpenUnit }) => {
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

	const holderTitle = unit.parentId ? 'Установлен в' : 'В составе комплекта'

	return (
		<>
			<Stack spacing={2}>
				{installed && host ? (
					<DetailCard title={holderTitle}>
						<MemberCard unit={host} onOpen={onOpenUnit} />
					</DetailCard>
				) : (
					<HolderCard unit={unit} />
				)}

				<DetailCard title='Технические характеристики' divided>
					<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, columnGap: 4 }}>
						{specsOf(unit).map(spec => (
							<Box
								key={spec.label}
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'space-between',
									gap: 2,
									py: 0.75,
								}}
							>
								<Typography sx={{ fontSize: 14, color: '#64748b' }}>{spec.label}</Typography>
								{spec.mono ? (
									<Box
										component='span'
										sx={{
											fontFamily: 'monospace',
											fontSize: 12,
											color: '#334155',
											bgcolor: '#f1f5f9',
											px: 0.75,
											py: 0.25,
											borderRadius: '6px',
										}}
									>
										{spec.value}
									</Box>
								) : (
									<Typography
										sx={{ fontSize: 14, fontWeight: 500, color: '#0f172a', textAlign: 'right' }}
									>
										{spec.value}
									</Typography>
								)}
							</Box>
						))}
					</Box>
				</DetailCard>

				{unit.notes && (
					<DetailCard title='Примечание'>
						<Typography sx={{ fontSize: 14, color: '#334155', whiteSpace: 'pre-wrap' }}>
							{unit.notes}
						</Typography>
					</DetailCard>
				)}

				{(isKit || isUnit) && (
					<DetailCard
						title={isKit ? 'Состав комплекта' : 'Комплектующие'}
						action={
							canWrite && (
								<Button
									size='small'
									startIcon={<PlusIcon sx={{ fontSize: 14 }} />}
									sx={softButtonSx}
									onClick={() => (isKit ? setCompositionOpen(true) : setPartsOpen(true))}
								>
									Изменить состав
								</Button>
							)
						}
					>
						{members.length === 0 ? (
							<EmptyState
								icon={PackageIcon}
								title={isKit ? 'Комплект пуст' : 'Составных частей нет'}
								hint={
									isKit
										? 'Добавьте комплектующие в состав комплекта'
										: 'Прикрепите комплектующие к устройству'
								}
							/>
						) : (
							<Stack spacing={1}>
								{members.map(m => (
									<MemberCard key={m.id} unit={m} onOpen={onOpenUnit} />
								))}
							</Stack>
						)}
					</DetailCard>
				)}

				<EquipmentTasks unit={unit} />
			</Stack>

			{/* Диалоги состава открываются из вкладки — её собственный контекст. */}
			{isKit && compositionOpen && (
				<EquipmentCompositionDialog open onClose={() => setCompositionOpen(false)} unit={unit} />
			)}
			{!isKit && partsOpen && <EquipmentPartsDialog open onClose={() => setPartsOpen(false)} unit={unit} />}
		</>
	)
}
