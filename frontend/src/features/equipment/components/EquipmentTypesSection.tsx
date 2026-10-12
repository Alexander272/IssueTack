import { useState, type FC } from 'react'
import {
	Box,
	Button,
	IconButton,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Tooltip,
	Typography,
	useTheme,
} from '@mui/material'
import { PencilIcon, PlusIcon, Trash2Icon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IEquipmentType } from '../types/equipment'
import { useDeleteEquipmentTypeMutation, useGetAllEquipmentTypesQuery } from '../equipmentApiSlice'
import { EquipmentTypeDialog } from './EquipmentTypeDialog'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IFetchError } from '@/app/types/error'

export const EquipmentTypesSection: FC = () => {
	const { palette } = useTheme()
	const canWrite = useCheckPermissions({ anyOf: ['equipment:write'] })

	const { data } = useGetAllEquipmentTypesQuery()
	const types = data?.data ?? []

	const [dialogOpen, setDialogOpen] = useState(false)
	const [editType, setEditType] = useState<IEquipmentType | null>(null)
	const [deleteType, setDeleteType] = useState<IEquipmentType | null>(null)
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentTypeMutation()

	const openCreate = () => {
		setEditType(null)
		setDialogOpen(true)
	}

	const openEdit = (type: IEquipmentType) => {
		setEditType(type)
		setDialogOpen(true)
	}

	const closeDialog = () => {
		setEditType(null)
		setDialogOpen(false)
	}

	const handleDelete = async () => {
		if (!deleteType) return
		try {
			await remove(deleteType.id).unwrap()
			toast.success('Тип удалён')
			setDeleteType(null)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	return (
		<Box>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
					mb: 2,
				}}
			>
				<Typography variant='body2' sx={{ color: '#6b7280' }}>
					Типы, добавляемые в карточку оборудования: ноутбук, RAM, SSD, принтер…
				</Typography>
				{canWrite && (
					<Button
						variant='outlined'
						sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500, gap: 1 }}
						onClick={openCreate}
					>
						<PlusIcon sx={{ color: palette.primary.main, fontSize: 16 }} />
						Добавить тип
					</Button>
				)}
			</Box>

			<Box
				sx={{
					border: '1px solid #eef2f6',
					borderRadius: '16px',
					overflow: 'hidden',
					bgcolor: '#fff',
				}}
			>
				<TableContainer>
					<Table size='small'>
						<TableHead>
							<TableRow sx={{ bgcolor: '#fafbfc' }}>
								<TableCell sx={{ fontWeight: 600, color: '#6b7280', width: 60 }}>
									Порядок
								</TableCell>
								<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Название</TableCell>
								<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Вид</TableCell>
								<TableCell align='right' sx={{ width: 100 }} />
							</TableRow>
						</TableHead>
						<TableBody>
							{types.length === 0 ? (
								<TableRow>
									<TableCell colSpan={4} align='center' sx={{ py: 6, color: '#6b7280' }}>
										Типы оборудования не добавлены
									</TableCell>
								</TableRow>
							) : (
								types.map(type => (
									<TableRow key={type.id} hover>
										<TableCell sx={{ color: '#6b7280', whiteSpace: 'nowrap' }}>
											{type.sortOrder}
										</TableCell>
										<TableCell sx={{ fontWeight: 500 }}>{type.name}</TableCell>
										<TableCell sx={{ color: '#9ca3af', fontSize: 13 }}>
											{type.isComponent ? 'Компонент' : 'Единица'}
										</TableCell>
										<TableCell align='right'>
											{canWrite && (
												<>
													<Tooltip title='Редактировать'>
														<IconButton size='small' onClick={() => openEdit(type)}>
															<PencilIcon sx={{ fontSize: 16 }} />
														</IconButton>
													</Tooltip>
													<Tooltip title='Удалить'>
														<IconButton
															size='small'
															onClick={() => setDeleteType(type)}
														>
															<Trash2Icon
																sx={{ fontSize: 16, color: palette.error.main }}
															/>
														</IconButton>
													</Tooltip>
												</>
											)}
										</TableCell>
									</TableRow>
								))
							)}
						</TableBody>
					</Table>
				</TableContainer>
			</Box>

			<EquipmentTypeDialog open={dialogOpen} onClose={closeDialog} type={editType} />

			<ConfirmDialog
				open={Boolean(deleteType)}
				title='Удаление типа'
				message={`Вы уверены, что хотите удалить тип «${deleteType?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteType(null)}
				loading={isDeleting}
			/>
		</Box>
	)
}