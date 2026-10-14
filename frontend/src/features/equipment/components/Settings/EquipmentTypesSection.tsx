import { useMemo, useState, type FC } from 'react'
import { Box, Button, Typography } from '@mui/material'
import { PlusIcon } from 'lucide-mui'

import type { IEquipmentType } from '../../types/equipment'
import { useGetAllEquipmentQuery, useGetAllEquipmentTypesQuery } from '../../equipmentApiSlice'
import { EquipmentTypeDialog } from '../Dialogs'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { EquipmentTypeRow } from './EquipmentTypeRow'
import { GROUP_LABELS, GROUP_ORDER, type GroupKey } from './typeIcons'

// Секция настроек «Типы оборудования»: справочник типов, разбитый по категориям,
// с числом заведённых единиц и диалогом создания/правки.
export const EquipmentTypesSection: FC = () => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })

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
		if (list.length === 0) return null
		return (
			<Box key={key} sx={{ mb: 3 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
					<Typography
						variant='subtitle2'
						sx={{ color: '#374151', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.3 }}
					>
						{GROUP_LABELS[key]}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						· {list.length} {pluralize(list.length, ['тип', 'типа', 'типов'])}
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
						<EquipmentTypeRow
							key={type.id}
							type={type}
							count={unitCounts[type.id] ?? 0}
							canWrite={canWrite}
							last={idx === list.length - 1}
							onEdit={openEdit}
						/>
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
