import { useMemo, useState, type FC } from 'react'
import {
	Box,
	Button,
	IconButton,
	Tooltip,
	Typography,
} from '@mui/material'
import {
	HardDriveIcon,
	KeyboardIcon,
	LaptopIcon,
	MemoryStickIcon,
	MicrochipIcon,
	MonitorIcon,
	MousePointer2Icon,
	PencilIcon,
	PhoneCallIcon,
	PlusIcon,
	PrinterIcon,
	Trash2Icon,
} from 'lucide-mui'

import type { IEquipmentType } from '../types/equipment'
import { useGetAllEquipmentTypesQuery } from '../equipmentApiSlice'
import { EquipmentTypeDialog } from './EquipmentTypeDialog'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

type GroupKey = 'devices' | 'components' | 'periphery'

const GROUP_LABELS: Record<GroupKey, { title: string; countLabel: string }> = {
	devices: { title: 'Устройства', countLabel: 'единиц' },
	components: { title: 'Комплектующие', countLabel: 'единиц' },
	periphery: { title: 'Периферия', countLabel: 'единиц' },
}

const getGroup = (type: IEquipmentType): GroupKey => {
	if (type.isComponent) return 'components'
	const name = type.name.toLowerCase()
	if (name.includes('мыш') || name.includes('клав') || name.includes('перифер')) return 'periphery'
	return 'devices'
}

const getIcon = (type: IEquipmentType) => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут')) return <LaptopIcon sx={{ color: '#0432a5' }} />
	if (name.includes('монит') || name.includes('диспл')) return <MonitorIcon sx={{ color: '#6b7280' }} />
	if (name.includes('принт') || name.includes('мфу')) return <PrinterIcon sx={{ color: '#6b7280' }} />
	if (name.includes('тел') || name.includes('смартф') || name.includes('phone')) return <PhoneCallIcon sx={{ color: '#6b7280' }} />
	if (name.includes('ram') || name.includes('памя') || name.includes('memory')) return <MemoryStickIcon sx={{ color: '#ea580c' }} />
	if (name.includes('ssd') || name.includes('hdd') || name.includes('диск') || name.includes('hard')) return <HardDriveIcon sx={{ color: '#9333ea' }} />
	if (name.includes('cpu') || name.includes('проц') || name.includes('микро')) return <MicrochipIcon sx={{ color: '#ea580c' }} />
	if (name.includes('мыш')) return <MousePointer2Icon sx={{ color: '#6b7280' }} />
	if (name.includes('клав')) return <KeyboardIcon sx={{ color: '#6b7280' }} />
	return <LaptopIcon sx={{ color: '#6b7280' }} />
}

const getIconBg = (type: IEquipmentType): string => {
	const name = type.name.toLowerCase()
	if (name.includes('ноут')) return '#eaeefc'
	if (name.includes('ram') || name.includes('памя') || name.includes('memory')) return '#fff7ed'
	if (name.includes('ssd') || name.includes('hdd') || name.includes('диск')) return '#f3e8ff'
	if (name.includes('cpu') || name.includes('проц') || name.includes('микро')) return '#fff7ed'
	return '#f3f4f6'
}

export const EquipmentTypesSection: FC = () => {
	const canWrite = useCheckPermissions({ anyOf: ['equipment:write'] })

	const { data } = useGetAllEquipmentTypesQuery()
		const [dialogOpen, setDialogOpen] = useState(false)
	const [editType, setEditType] = useState<IEquipmentType | null>(null)

	const grouped = useMemo(() => {
		const types = data?.data ?? []
		const map: Record<GroupKey, IEquipmentType[]> = {
			devices: [],
			components: [],
			periphery: [],
		}
		types.forEach(t => {
			map[getGroup(t)].push(t)
		})
		return map
	}, [data])

	const counts: Record<GroupKey, number> = {
		devices: grouped.devices.length,
		components: grouped.components.length,
		periphery: grouped.periphery.length,
	}

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
		const g = GROUP_LABELS[key]
		if (list.length === 0) return null
		return (
			<Box key={key} sx={{ mb: 3 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
					<Typography variant='subtitle2' sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}>
						{g.title}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						· {counts[key]} {g.countLabel}
					</Typography>
				</Box>
				<Box sx={{ bgcolor: '#fff', border: '1px solid #e5e7eb', borderRadius: '12px', boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)' }}>
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
										bgcolor: key === 'components' ? '#fff7ed' : key === 'periphery' ? '#eff6ff' : '#f3f4f6',
										color: key === 'components' ? '#9a3412' : key === 'periphery' ? '#1e40af' : '#374151',
									}}
								>
									{key === 'components' ? 'Комплектующее' : key === 'periphery' ? 'Периферия' : 'Устройство'}
								</Box>
							</Box>
							<Box sx={{ gridColumn: 'span 2' }}>
								<Typography variant='body2' sx={{ color: '#374151' }}>
									— единиц
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
										<Tooltip title={type.isComponent ? 'Нельзя удалить: используется' : 'Удалить'}>
											<span>
												<IconButton size='small' disabled >
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
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.5 }}>
						Справочник типов для вашего реалма
					</Typography>
				</Box>
				{canWrite && (
					<Button
						variant='contained'
						onClick={openCreate}
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', fontWeight: 500, borderRadius: '8px', bgcolor: '#0432a5', '&:hover': { bgcolor: '#002d91' } }}
					>
						Добавить тип
					</Button>
				)}
			</Box>

			<Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
				{renderGroup('devices')}
				{renderGroup('components')}
				{renderGroup('periphery')}
			</Box>

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
