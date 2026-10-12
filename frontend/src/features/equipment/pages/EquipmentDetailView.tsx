import { useState, type FC } from 'react'
import {
	Box,
	Button,
	IconButton,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
	useTheme,
} from '@mui/material'
import { ArrowLeftIcon, PencilIcon, PlusIcon, Trash2Icon } from 'lucide-mui'
import { toast } from 'react-toastify'
import { Link as RouterLink, useNavigate, useParams } from 'react-router'

import { getEquipmentStatus } from '../types/equipment'
import { useDeleteEquipmentMutation, useGetEquipmentChildrenQuery, useGetEquipmentQuery } from '../equipmentApiSlice'
import { EquipmentDialog } from '../components/EquipmentDialog'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import type { IFetchError } from '@/app/types/error'

const labelSx = { color: '#9ca3af', fontSize: 12, fontWeight: 600, textTransform: 'uppercase' as const }
const valueSx = { color: '#1f2937', fontWeight: 500 }

const fmtDate = (d: string | null) => (d ? new Date(`${d}T00:00:00`).toLocaleDateString('ru-RU') : '—')

export const EquipmentDetailView: FC = () => {
	const { palette } = useTheme()
	const navigate = useNavigate()
	const { id = '' } = useParams()
	const canWrite = useCheckPermissions({ anyOf: ['equipment:write'] })

	const { data } = useGetEquipmentQuery(id)
	const { data: childrenData } = useGetEquipmentChildrenQuery(id, { skip: !id })
	const children = childrenData?.data ?? []
	const unit = data

	const [dialogOpen, setDialogOpen] = useState(false)
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()

	const handleDelete = async () => {
		try {
			await remove(id).unwrap()
			toast.success('Оборудование удалено')
			navigate('/equipment')
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	if (!unit) return null

	const status = getEquipmentStatus(unit.status)
	const holder = unit.holderFullName || unit.holderName

	return (
		<Box sx={{ p: 3 }}>
			<Button
				startIcon={<ArrowLeftIcon sx={{ fontSize: 16 }} />}
				onClick={() => navigate('/equipment')}
				sx={{ textTransform: 'none', mb: 2, color: '#6b7280' }}
			>
				К списку оборудования
			</Button>

			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: 2,
					mb: 3,
				}}
			>
				<Box>
					<Box sx={{ display: 'flex', flexDirection: 'row', gap: 1, alignItems: 'center', mb: 0.5 }}>
						<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
							{unit.name}
						</Typography>
						<Typography
							component='span'
							sx={{
								px: 1,
								py: 0.25,
								borderRadius: '6px',
								fontSize: 12,
								fontWeight: 600,
								bgcolor: `${status.color}1a`,
								color: status.color,
							}}
						>
							{status.label}
						</Typography>
					</Box>
					<Typography variant='body2' sx={{ color: '#6b7280' }}>
						{unit.typeName}
						{unit.parentId && (
							<>
								{' · в составе '}
								<RouterLink
									to={`/equipment/${unit.parentId}`}
									style={{ color: palette.primary.main, textDecoration: 'none' }}
								>
									комплекта
								</RouterLink>
							</>
						)}
					</Typography>
				</Box>
				{canWrite && (
					<Stack direction='row' spacing={1}>
						<Button
							variant='outlined'
							sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
							onClick={() => setDialogOpen(true)}
						>
							<PencilIcon sx={{ color: palette.primary.main, fontSize: 16 }} />
							Редактировать
						</Button>
						<IconButton color='error' onClick={() => setDeleteOpen(true)}>
							<Trash2Icon sx={{ fontSize: 18 }} />
						</IconButton>
					</Stack>
				)}
			</Box>

			<Box
				sx={{
					display: 'grid',
					gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr', lg: '1fr 1fr 1fr' },
					gap: 2,
					mb: 3,
				}}
			>
				{[
					['Производитель', unit.vendor || '—'],
					['Модель', unit.model || '—'],
					['Серийный номер', unit.serialNumber ?? '—'],
					['Инвентарный номер', unit.inventoryNumber ?? '—'],
					['Площадка', unit.siteName || '—'],
					['Держатель', holder || '—'],
					['Локация', unit.location ?? '—'],
					['Дата покупки', fmtDate(unit.purchaseDate)],
					['Гарантия до', fmtDate(unit.warrantyUntil)],
					['Стоимость', unit.cost != null ? `${unit.cost.toLocaleString('ru-RU')} ₽` : '—'],
				].map(([label, value]) => (
					<Box
						key={label}
						sx={{ border: '1px solid #eef2f6', borderRadius: '12px', p: 2, bgcolor: '#fff' }}
					>
						<Typography sx={labelSx}>{label}</Typography>
						<Typography sx={valueSx}>{value}</Typography>
					</Box>
				))}
			</Box>

			{unit.notes && (
				<Box
					sx={{ border: '1px solid #eef2f6', borderRadius: '12px', p: 2, bgcolor: '#fff', mb: 3 }}
				>
					<Typography sx={labelSx}>Примечание</Typography>
					<Typography sx={{ ...valueSx, whiteSpace: 'pre-wrap' }}>{unit.notes}</Typography>
				</Box>
			)}

			<Box sx={{ mb: 4 }}>
				<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5, gap: 2 }}>
					<Typography variant='h6' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Состав комплекта
					</Typography>
					{canWrite && (
						<Button
							variant='outlined'
							size='small'
							sx={{ borderRadius: '8px', textTransform: 'none', gap: 0.5 }}
							onClick={() => setDialogOpen(true)}
						>
							<PlusIcon sx={{ color: palette.primary.main, fontSize: 16 }} />
							Добавить
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
									<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Название</TableCell>
									<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Тип</TableCell>
									<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Инв. №</TableCell>
									<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Статус</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{children.length === 0 ? (
									<TableRow>
										<TableCell colSpan={4} align='center' sx={{ py: 4, color: '#6b7280' }}>
											Комплект пуст — состав пока не добавлен
										</TableCell>
									</TableRow>
								) : (
									children.map(child => {
										const st = getEquipmentStatus(child.status)
										return (
											<TableRow
												key={child.id}
												hover
												sx={{ cursor: 'pointer' }}
												onClick={() => navigate(`/equipment/${child.id}`)}
											>
												<TableCell sx={{ fontWeight: 500 }}>{child.name}</TableCell>
												<TableCell sx={{ color: '#6b7280' }}>{child.typeName}</TableCell>
												<TableCell sx={{ color: '#6b7280' }}>
													{child.inventoryNumber ?? '—'}
												</TableCell>
												<TableCell sx={{ whiteSpace: 'nowrap' }}>
													<Typography
														component='span'
														sx={{
															px: 1,
															py: 0.25,
															borderRadius: '6px',
															fontSize: 12,
															fontWeight: 600,
															bgcolor: `${st.color}1a`,
															color: st.color,
														}}
													>
														{st.label}
													</Typography>
												</TableCell>
											</TableRow>
										)
									})
								)}
							</TableBody>
						</Table>
					</TableContainer>
				</Box>
			</Box>

			<EquipmentDialog open={dialogOpen} onClose={() => setDialogOpen(false)} unit={unit} />

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
		</Box>
	)
}