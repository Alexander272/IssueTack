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
import type { IEquipment } from '../../types/equipment'
import {
	useAddEquipmentComponentsMutation,
	useGetAllEquipmentQuery,
	useGetEquipmentCompositionQuery,
	useRemoveEquipmentComponentsMutation,
} from '../../equipmentApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

export const EquipmentCompositionDialog: FC<Props> = ({ open, onClose, unit }) => {
	const { data } = useGetEquipmentCompositionQuery(unit.id, { skip: !open })
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })

	const [add, { isLoading: isAdding }] = useAddEquipmentComponentsMutation()
	const [remove, { isLoading: isRemoving }] = useRemoveEquipmentComponentsMutation()

	const [reason, setReason] = useState('')
	const [selected, setSelected] = useState<IEquipment | null>(null)

	const members = data?.data ?? []
	const memberIds = new Set(members.map(m => m.id))
	const candidates = (unitsData?.data ?? []).filter(u => u.id !== unit.id && !memberIds.has(u.id))

	const handleAdd = async () => {
		if (!selected) return
		try {
			await add({ id: unit.id, body: { components: [{ componentId: selected.id }], reason } }).unwrap()
			toast.success('Компонент добавлен')
			setSelected(null)
			setReason('')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при добавлении', { autoClose: false })
		}
	}

	const handleRemove = async (componentId: string) => {
		try {
			await remove({ id: unit.id, body: { components: [{ componentId }], reason } }).unwrap()
			toast.success('Компонент исключён')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при исключении', { autoClose: false })
		}
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
					Состав комплекта
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
								Состав пуст
							</Typography>
						) : (
							members.map(m => (
								<ListItem
									key={m.id}
									secondaryAction={
										<IconButton
											color='error'
											disabled={isRemoving}
											onClick={() => handleRemove(m.id)}
										>
											<Trash2Icon sx={{ fontSize: 18 }} />
										</IconButton>
									}
								>
									<ListItemText primary={m.name} secondary={m.typeName} />
								</ListItem>
							))
						)}
					</List>

					<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr auto' }, gap: 2, alignItems: 'start' }}>
						<Autocomplete
							options={candidates}
							getOptionLabel={u => u.name}
							value={selected}
							onChange={(_e, value) => setSelected(value)}
							renderInput={params => (
								<TextField {...params} size='small' placeholder='Добавить компонент' />
							)}
						/>
						<Button
							variant='outlined'
							startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
							disabled={!selected || isAdding}
							onClick={handleAdd}
							sx={{ textTransform: 'none', height: 40, whiteSpace: 'nowrap' }}
						>
							Добавить
						</Button>
					</Box>

					<Box>
						<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
							Причина (в историю)
						</Typography>
						<TextField
							value={reason}
							onChange={e => setReason(e.target.value)}
							size='small'
							fullWidth
							placeholder='Например: замена комплектующей'
						/>
					</Box>
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
