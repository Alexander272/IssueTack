import { useState, type FC } from 'react'
import { Box, Button, IconButton, Stack, Typography } from '@mui/material'
import { PackageCheckIcon, PackageMinusIcon, PencilIcon, Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IEquipment } from '../../types/equipment'
import { getEquipmentStatus } from '../../types/equipment'
import { useDeleteEquipmentMutation, useUnassignEquipmentMutation } from '../../equipmentApiSlice'
import { equipmentAccent } from '../../utils/typeKind'
import { EquipmentAssignDialog, EquipmentDialog } from '../Dialogs'
import { EquipmentKindIcon } from '../EquipmentKindIcon'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

type Props = {
	unit: IEquipment
	onClose: () => void
}

// Шапка модалки единицы: иконка типа и название со статусом, действия
// (выдать/вернуть, редактировать, удалить) и кнопка закрытия. Диалоги действий
// открываются здесь же, рядом со своими кнопками, поэтому наружу уходят только
// единица и закрытие модалки — мостиков-колбэков в контейнер нет. Прав на запись
// нужны только здесь, поэтому компонент берёт их сам.
export const EquipmentDetailHeader: FC<Props> = ({ unit, onClose }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const [assignOpen, setAssignOpen] = useState(false)
	const [editOpen, setEditOpen] = useState(false)
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()
	const [unassign, { isLoading: isUnassigning }] = useUnassignEquipmentMutation()

	const accent = equipmentAccent(unit)
	const status = getEquipmentStatus(unit.status)
	const hasHolder = Boolean(unit.holderUserId || unit.holderName)
	const installed = Boolean(unit.parentId || unit.kitId)

	const handleDelete = async () => {
		try {
			await remove(unit.id).unwrap()
			toast.success('Оборудование удалено')
			setDeleteOpen(false)
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	const handleUnassign = async () => {
		try {
			await unassign({ id: unit.id, body: { comment: 'Возврат со страницы оборудования' } }).unwrap()
			toast.success('Оборудование возвращено')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	return (
		<>
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
									onClick={handleUnassign}
									sx={{ borderRadius: '8px', textTransform: 'none' }}
								>
									Возврат
								</Button>
							) : (
								<Button
									variant='outlined'
									size='small'
									startIcon={<PackageCheckIcon sx={{ fontSize: 16 }} />}
									onClick={() => setAssignOpen(true)}
									sx={{ borderRadius: '8px', textTransform: 'none' }}
								>
									Выдать
								</Button>
							))}
						<Button
							variant='outlined'
							size='small'
							sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
							onClick={() => setEditOpen(true)}
						>
							<PencilIcon sx={{ fontSize: 16 }} />
							Редактировать
						</Button>
						<IconButton size='small' color='error' onClick={() => setDeleteOpen(true)}>
							<Trash2Icon sx={{ fontSize: 18 }} />
						</IconButton>
					</Stack>
				)}
				<IconButton size='small' onClick={onClose} sx={{ color: '#9ca3af' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Box>

			{/* Условный рендер — форма монтируется заново при каждом открытии,
			    чтобы не показывать значения предыдущего элемента. */}
			{editOpen && <EquipmentDialog open onClose={() => setEditOpen(false)} unit={unit} />}
			{assignOpen && <EquipmentAssignDialog open onClose={() => setAssignOpen(false)} unit={unit} />}

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление оборудования'
				message={`Вы уверены, что хотите удалить «${unit.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
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
