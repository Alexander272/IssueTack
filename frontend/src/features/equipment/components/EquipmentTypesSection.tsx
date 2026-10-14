import { useMemo, useState, type FC } from 'react'
import { Box, Button, IconButton, Tooltip, Typography } from '@mui/material'
import {
	BatteryIcon,
	BriefcaseIcon,
	CableIcon,
	CpuIcon,
	DropletIcon,
	HardDriveIcon,
	KeyboardIcon,
	LaptopIcon,
	MemoryStickIcon,
	MonitorIcon,
	MouseIcon,
	PackageIcon,
	PcCaseIcon,
	PencilIcon,
	PhoneCallIcon,
	PlusIcon,
	PrinterIcon,
	Trash2Icon,
} from 'lucide-mui'

import type { EquipmentTypeKind, IEquipmentType } from '../types/equipment'
import { EQUIPMENT_TYPE_KIND_LABELS } from '../types/equipment'
import { useGetAllEquipmentQuery, useGetAllEquipmentTypesQuery } from '../equipmentApiSlice'
import { EquipmentTypeDialog } from './EquipmentTypeDialog'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'

type GroupKey = EquipmentTypeKind

const GROUP_ORDER: GroupKey[] = ['device', 'component', 'periphery', 'consumable']

const GROUP_LABELS: Record<GroupKey, string> = {
	device: 'Устройства',
	component: 'Комплектующие',
	periphery: 'Периферия',
	consumable: 'Расходники',
}

const BADGE_STYLES: Record<GroupKey, { bgcolor: string; color: string }> = {
	device: { bgcolor: '#f3f4f6', color: '#374151' },
	component: { bgcolor: '#fff7ed', color: '#9a3412' },
	periphery: { bgcolor: '#eff6ff', color: '#1e40af' },
	consumable: { bgcolor: '#ecfdf5', color: '#065f46' },
}

const getIcon = (type: IEquipmentType) => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут') || name.includes('laptop')) return <LaptopIcon sx={{ color: '#0432a5' }} />
	if (name.includes('монит') || name.includes('диспл') || name.includes('monitor'))
		return <MonitorIcon sx={{ color: '#6b7280' }} />
	if (name.includes('принт') || name.includes('мфу') || name.includes('print'))
		return <PrinterIcon sx={{ color: '#6b7280' }} />
	if (
		name.includes('накопит') ||
		name.includes('диск') ||
		name.includes('ssd') ||
		name.includes('hdd') ||
		name.includes('nvme') ||
		name.includes('hard')
	)
		return <HardDriveIcon sx={{ color: '#9333ea' }} />
	if (name.includes('тел') || name.includes('смартф') || name.includes('phone'))
		return <PhoneCallIcon sx={{ color: '#6b7280' }} />
	if (name.includes('ram') || name.includes('памя') || name.includes('memory') || name.includes('озу'))
		return <MemoryStickIcon sx={{ color: '#ea580c' }} />

	if (name.includes('cpu') || name.includes('проц') || name.includes('микро'))
		return <CpuIcon sx={{ color: '#ea580c' }} />
	if (name.includes('мыш') || name.includes('mouse')) return <MouseIcon sx={{ color: '#6b7280' }} />
	if (name.includes('клав') || name.includes('keyboard')) return <KeyboardIcon sx={{ color: '#6b7280' }} />
	if (name.includes('сумк') || name.includes('bag')) return <BriefcaseIcon sx={{ color: '#a16207' }} />
	if (name.includes('системн') || name.includes('корпус') || name.includes('блок'))
		return <PcCaseIcon sx={{ color: '#374151' }} />
	if (name.includes('картридж') || name.includes('тонер') || name.includes('чернил'))
		return <DropletIcon sx={{ color: '#0891b2' }} />
	if (name.includes('кабел') || name.includes('провод')) return <CableIcon sx={{ color: '#6b7280' }} />
	if (name.includes('батаре') || name.includes('аккумул')) return <BatteryIcon sx={{ color: '#16a34a' }} />
	return <PackageIcon sx={{ color: '#6b7280' }} />
}

const getIconBg = (type: IEquipmentType): string => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут')) return '#eaeefc'
	if (name.includes('ram') || name.includes('памя') || name.includes('memory') || name.includes('озу'))
		return '#fff7ed'
	if (name.includes('накопит') || name.includes('диск') || name.includes('ssd') || name.includes('hdd'))
		return '#f3e8ff'
	if (name.includes('cpu') || name.includes('проц') || name.includes('микро')) return '#fff7ed'
	if (name.includes('сумк')) return '#fef3c7'
	if (name.includes('картридж') || name.includes('тонер') || name.includes('чернил')) return '#ecfeff'
	if (name.includes('батаре') || name.includes('аккумул')) return '#ecfdf5'
	return '#f3f4f6'
}

export const EquipmentTypesSection: FC = () => {
	const canWrite = useCheckPermissions({ anyOf: ['equipment:write'] })

	const { data } = useGetAllEquipmentTypesQuery()
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })
	const [dialogOpen, setDialogOpen] = useState(false)
	const [editType, setEditType] = useState<IEquipmentType | null>(null)

	const grouped = useMemo(() => {
		const types = data?.data ?? []
		const map: Record<GroupKey, IEquipmentType[]> = {
			device: [],
			component: [],
			periphery: [],
			consumable: [],
		}
		types.forEach(t => {
			map[t.kind]?.push(t)
		})
		return map
	}, [data])

	const counts: Record<GroupKey, number> = {
		device: grouped.device.length,
		component: grouped.component.length,
		periphery: grouped.periphery.length,
		consumable: grouped.consumable.length,
	}

	const unitCounts = useMemo(() => {
		const map: Record<string, number> = {}
		;(unitsData?.data ?? []).forEach(unit => {
			map[unit.typeId] = (map[unit.typeId] ?? 0) + 1
		})
		return map
	}, [unitsData])

	const openCreate = () => {
		setEditType(null)
		setDialogOpen(true)
	}

	const openEdit = (type: IEquipmentType) => {
		setEditType(type)
		setDialogOpen(true)
	}

	const renderGroup = (key: GroupKey) => {
		const list = grouped[key]
		const title = GROUP_LABELS[key]
		if (list.length === 0) return null
		return (
			<Box key={key} sx={{ mb: 3 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
					<Typography
						variant='subtitle2'
						sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}
					>
						{title}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						· {counts[key]} {pluralize(counts[key], ['тип', 'типа', 'типов'])}
					</Typography>
				</Box>
				<Box
					sx={{
						bgcolor: '#fff',
						border: '1px solid #e5e7eb',
						borderRadius: '12px',
						boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)',
					}}
				>
					{list.map((type, idx) => (
						<Box
							key={type.id}
							sx={{
								display: 'grid',
								gridTemplateColumns: 'repeat(12, 1fr)',
								gap: 2,
								px: 3,
								py: 2,
								alignItems: 'center',
								borderBottom: idx < list.length - 1 ? '1px solid #f3f4f6' : 'none',
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
										...BADGE_STYLES[key],
									}}
								>
									{EQUIPMENT_TYPE_KIND_LABELS[key]}
								</Box>
							</Box>
							<Box sx={{ gridColumn: 'span 2' }}>
								<Typography variant='body2' sx={{ color: '#374151' }}>
									{unitCounts[type.id] ?? 0}{' '}
									{pluralize(unitCounts[type.id] ?? 0, ['единица', 'единицы', 'единиц'])}
								</Typography>
							</Box>
							<Box sx={{ gridColumn: 'span 2', textAlign: 'right' }}>
								{canWrite && (
									<>
										<Tooltip title='Редактировать'>
											<IconButton size='small' onClick={() => openEdit(type)}>
												<PencilIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
										<Tooltip
											title={
												type.kind === 'component' ? 'Нельзя удалить: используется' : 'Удалить'
											}
										>
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
					))}
				</Box>
			</Box>
		)
	}

	return (
		<Box sx={{ p: 3, maxWidth: 1200, mx: 'auto' }}>
			<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
				<Box>
					<Typography variant='h6' sx={{ fontWeight: 700, color: '#111827' }}>
						Типы оборудования
					</Typography>
				</Box>
				{canWrite && (
					<Button
						variant='outlined'
						onClick={openCreate}
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', fontWeight: 500, borderRadius: '8px', gap: 1 }}
					>
						Добавить тип
					</Button>
				)}
			</Box>

			<Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>{GROUP_ORDER.map(renderGroup)}</Box>

			<EquipmentTypeDialog
				open={dialogOpen}
				onClose={() => {
					setEditType(null)
					setDialogOpen(false)
				}}
				type={editType}
			/>
		</Box>
	)
}
