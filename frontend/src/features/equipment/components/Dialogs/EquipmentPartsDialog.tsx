import { type FC, useState } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	List,
	ListItem,
	ListItemText,
	TextField,
	Typography,
} from '@mui/material'
import { PlusIcon, Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import { canBePart } from '../../types/equipment'
import {
	useGetAllEquipmentQuery,
	useGetEquipmentChildrenQuery,
	useUpdateEquipmentMutation,
} from '../../equipmentApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

// toPayload переводит единицу в DTO для смены физического родителя. Отдельной
// ручки «прикрепить/открепить» нет: состав меняется через PUT, а он пишет все
// поля, поэтому переносим текущие значения и меняем только parentId.
const toPayload = (child: IEquipment, parentId: string | null): IEquipmentDTO => ({
	id: child.id,
	typeId: child.typeId,
	name: child.name,
	vendor: child.vendor,
	model: child.model,
	serialNumber: child.serialNumber,
	inventoryNumber: child.inventoryNumber,
	specs: child.specs,
	status: child.status,
	parentId,
	siteId: child.siteId,
	holderUserId: child.holderUserId,
	holderName: child.holderName,
	location: child.location,
	purchaseDate: child.purchaseDate,
	warrantyUntil: child.warrantyUntil,
	cost: child.cost,
	notes: child.notes,
	isKit: child.isKit,
})

// Физический состав единицы (parent_id): прикрепить можно только уже заведённую
// единицу — комплектующее или расходник (то же ограничение на бэкенде).
export const EquipmentPartsDialog: FC<Props> = ({ open, onClose, unit }) => {
	const { data } = useGetEquipmentChildrenQuery(unit.id, { skip: !open })
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })
	const [update, { isLoading }] = useUpdateEquipmentMutation()

	const [selected, setSelected] = useState<IEquipment | null>(null)

	const members = data?.data ?? []
	const memberIds = new Set(members.map(m => m.id))
	const candidates = (unitsData?.data ?? []).filter(
		u => u.id !== unit.id && !u.isKit && canBePart(u.typeKind) && !memberIds.has(u.id),
	)

	const changeParent = async (child: IEquipment, parentId: string | null) => {
		try {
			await update(toPayload(child, parentId)).unwrap()
			toast.success(parentId ? 'Добавлено в состав' : 'Исключено из состава')
			return true
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при изменении состава', { autoClose: false })
			return false
		}
	}

	const handleAttach = async () => {
		if (!selected) return
		if (await changeParent(selected, unit.id)) setSelected(null)
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{ paper: { sx: { borderRadius: '16px', p: 1 } } }}
		>
			<DialogTitle
				sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
			>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Физический состав
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent
				dividers
				sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
			>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
					<List dense sx={{ py: 0 }}>
						{members.length === 0 ? (
							<Typography variant='body2' sx={{ color: '#6b7280', py: 2 }}>
								Составных частей нет
							</Typography>
						) : (
							members.map(m => (
								<ListItem
									key={m.id}
									secondaryAction={
										<IconButton
											color='error'
											disabled={isLoading}
											onClick={() => changeParent(m, null)}
										>
											<Trash2Icon sx={{ fontSize: 18 }} />
										</IconButton>
									}
								>
									<ListItemText
										primary={m.name}
										secondary={`${m.typeName} · Инв. №${m.inventoryNumber ?? '—'}`}
									/>
								</ListItem>
							))
						)}
					</List>

					<Box
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: '1fr', sm: '1fr auto' },
							gap: 2,
							alignItems: 'start',
						}}
					>
						<Autocomplete
							options={candidates}
							getOptionLabel={u => u.name}
							value={selected}
							onChange={(_e, value) => setSelected(value)}
							renderInput={params => (
								<TextField {...params} size='small' placeholder='Прикрепить существующее' />
							)}
						/>
						<Button
							variant='outlined'
							startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
							disabled={!selected || isLoading}
							onClick={handleAttach}
							sx={{ textTransform: 'none', height: 40, whiteSpace: 'nowrap' }}
						>
							Добавить
						</Button>
					</Box>

					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						В состав можно прикрепить только комплектующее или расходник
					</Typography>
				</Box>
			</DialogContent>

			<DialogActions sx={{ p: 2 }}>
				<Button
					onClick={onClose}
					variant='outlined'
					sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
				>
					Закрыть
				</Button>
			</DialogActions>
		</Dialog>
	)
}
