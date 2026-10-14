import { type FC, useState } from 'react'
import { Box, Dialog, DialogContent, Tab, Tabs } from '@mui/material'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { Loader } from '@/components/Fallback/Loader'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import {
	useDeleteEquipmentMutation,
	useGetEquipmentQuery,
	useUnassignEquipmentMutation,
} from '../../equipmentApiSlice'
import { EquipmentAssignDialog, EquipmentCompositionDialog, EquipmentDialog, EquipmentPartsDialog } from '../Dialogs'
import { EquipmentHistoryTimeline } from './EquipmentHistoryTimeline'
import { EquipmentItemActs } from './EquipmentItemActs'
import { EquipmentDetailHeader } from './EquipmentDetailHeader'
import { EquipmentOverviewTab } from './EquipmentOverviewTab'

type Props = {
	open: boolean
	unitId: string | null
	onClose: () => void
	// onOpenUnit — открыть карточку связанной единицы на месте (без смены URL).
	onOpenUnit: (id: string) => void
}

// Модалка единицы оборудования. Контейнер держит данные единицы, вкладки и
// диалоги уровня модалки (правка, выдача, состав, удаление); сами вкладки —
// презентационные компоненты.
export const EquipmentDetailModal: FC<Props> = ({ open, unitId, onClose, onOpenUnit }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const [tab, setTab] = useState(0)

	const { data, isLoading } = useGetEquipmentQuery(unitId ?? '', { skip: !unitId })
	const unit = data?.data

	const [editOpen, setEditOpen] = useState(false)
	const [assignOpen, setAssignOpen] = useState(false)
	const [compositionOpen, setCompositionOpen] = useState(false)
	const [partsOpen, setPartsOpen] = useState(false)
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()
	const [unassign, { isLoading: isUnassigning }] = useUnassignEquipmentMutation()

	const handleDelete = async () => {
		if (!unit) return
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
		if (!unit) return
		try {
			await unassign({ id: unit.id, body: { comment: 'Возврат со страницы оборудования' } }).unwrap()
			toast.success('Оборудование возвращено')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='lg'
			slotProps={{ paper: { sx: { borderRadius: '16px', backgroundImage: 'none' } } }}
		>
			{!unit ? (
				<Box sx={{ minHeight: 320, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
					{isLoading ? <Loader /> : null}
				</Box>
			) : (
				<>
					<Box sx={{ px: 3, pt: 2.5 }}>
						<EquipmentDetailHeader
							unit={unit}
							canWrite={canWrite}
							isUnassigning={isUnassigning}
							onAssign={() => setAssignOpen(true)}
							onUnassign={handleUnassign}
							onEdit={() => setEditOpen(true)}
							onDelete={() => setDeleteOpen(true)}
							onClose={onClose}
						/>

						<Tabs
							value={tab}
							onChange={(_e, value) => setTab(value)}
							textColor='primary'
							indicatorColor='primary'
							sx={{ mt: 2, minHeight: 40, borderBottom: '1px solid #eef2f6' }}
						>
							<Tab label='Обзор' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='История' sx={{ textTransform: 'none', minHeight: 40 }} />
							<Tab label='Акты' sx={{ textTransform: 'none', minHeight: 40 }} />
						</Tabs>
					</Box>

					<DialogContent dividers sx={{ minHeight: 360, bgcolor: '#fff' }}>
						{tab === 0 && (
							<EquipmentOverviewTab
								unit={unit}
								canWrite={canWrite}
								onComposition={() => setCompositionOpen(true)}
								onParts={() => setPartsOpen(true)}
								onOpenUnit={onOpenUnit}
							/>
						)}
						{tab === 1 && <EquipmentHistoryTimeline unit={unit} />}
						{tab === 2 && <EquipmentItemActs unit={unit} />}
					</DialogContent>

					{/* Условный рендер — форма монтируется заново при каждом открытии,
					    чтобы не показывать значения предыдущего элемента. */}
					{editOpen && <EquipmentDialog open onClose={() => setEditOpen(false)} unit={unit} />}
					{assignOpen && <EquipmentAssignDialog open onClose={() => setAssignOpen(false)} unit={unit} />}
					{unit.isKit && compositionOpen && (
						<EquipmentCompositionDialog open onClose={() => setCompositionOpen(false)} unit={unit} />
					)}
					{!unit.isKit && partsOpen && (
						<EquipmentPartsDialog open onClose={() => setPartsOpen(false)} unit={unit} />
					)}

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
			)}
		</Dialog>
	)
}
