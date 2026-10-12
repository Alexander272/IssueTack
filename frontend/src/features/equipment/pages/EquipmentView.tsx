import { useState, type FC } from 'react'
import {
	Box,
	Button,
	IconButton,
	MenuItem,
	Pagination as MuiPagination,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	TextField,
	Tooltip,
	Typography,
	useTheme,
} from '@mui/material'
import { PencilIcon, PlusIcon, SearchIcon, Trash2Icon } from 'lucide-mui'
import { toast } from 'react-toastify'
import { useNavigate } from 'react-router'

import type { IEquipment, IEquipmentFilter } from '../types/equipment'
import { EQUIPMENT_STATUS_OPTIONS, getEquipmentStatus } from '../types/equipment'
import {
	useDeleteEquipmentMutation,
	useGetAllEquipmentQuery,
	useGetAllEquipmentTypesQuery,
} from '../equipmentApiSlice'
import { EquipmentDialog } from '../components/EquipmentDialog'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import type { IFetchError } from '@/app/types/error'

const PAGE_SIZE = 20

export const EquipmentView: FC = () => {
	const { palette } = useTheme()
	const navigate = useNavigate()
	const canWrite = useCheckPermissions({ anyOf: ['equipment:write'] })

	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const { data: sitesData } = useGetAllSitesQuery()
	const types = typesData?.data ?? []
	const sites = sitesData?.data ?? []

	const [filter, setFilter] = useState<IEquipmentFilter>({})
	const [search, setSearch] = useState('')
	const [page, setPage] = useState(1)

	const { data } = useGetAllEquipmentQuery({ ...filter, search: search || undefined, limit: PAGE_SIZE, offset: (page - 1) * PAGE_SIZE })
	const units = data?.data ?? []
	const total = data?.total ?? 0
	const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

	const [dialogOpen, setDialogOpen] = useState(false)
	const [editUnit, setEditUnit] = useState<IEquipment | null>(null)
	const [deleteUnit, setDeleteUnit] = useState<IEquipment | null>(null)
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()

	const submitSearch = () => {
		setPage(1)
		setFilter(f => ({ ...f, search: search || undefined }))
	}

	const setFilterField = <K extends keyof IEquipmentFilter>(key: K, value: IEquipmentFilter[K]) => {
		setPage(1)
		setFilter(f => ({ ...f, [key]: value }))
	}

	const openCreate = () => {
		setEditUnit(null)
		setDialogOpen(true)
	}

	const openEdit = (unit: IEquipment) => {
		setEditUnit(unit)
		setDialogOpen(true)
	}

	const closeDialog = () => {
		setEditUnit(null)
		setDialogOpen(false)
	}

	const handleDelete = async () => {
		if (!deleteUnit) return
		try {
			await remove(deleteUnit.id).unwrap()
			toast.success('Оборудование удалено')
			setDeleteUnit(null)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message, { autoClose: false })
		}
	}

	const holderLabel = (unit: IEquipment) =>
		unit.holderFullName || unit.holderName || (unit.location ? `Локация: ${unit.location}` : '—')

	return (
		<Box sx={{ p: 3 }}>
			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 0 },
					mb: 3,
				}}
			>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Оборудование
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						Инвентарь, комплекты, история выдач
					</Typography>
				</Box>
				{canWrite && (
					<Button
						variant='outlined'
						sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500, gap: 1 }}
						onClick={openCreate}
					>
						<PlusIcon sx={{ color: palette.primary.main, fontSize: 16 }} />
						Добавить единицу
					</Button>
				)}
			</Box>

			<Stack
				direction={{ xs: 'column', sm: 'row' }}
				spacing={{ xs: 1.5, sm: 2 }}
				sx={{ mb: 3, flexWrap: 'wrap' }}
			>
				<TextField
					size='small'
					placeholder='Поиск: название, серийник…'
					value={search}
					onChange={e => setSearch(e.target.value)}
					onKeyDown={e => {
						if (e.key === 'Enter') submitSearch()
					}}
					slotProps={{
						input: {
							startAdornment: <SearchIcon sx={{ fontSize: 16, color: '#9ca3af', mr: 1 }} />,
						},
					}}
					sx={{ minWidth: 240 }}
				/>
				<TextField
					size='small'
					select
					label='Тип'
					value={filter.typeId ?? ''}
					onChange={e => setFilterField('typeId', e.target.value || undefined)}
					sx={{ minWidth: 180 }}
				>
					<MenuItem value=''>Все типы</MenuItem>
					{types.map(t => (
						<MenuItem key={t.id} value={t.id}>
							{t.name}
						</MenuItem>
					))}
				</TextField>
				<TextField
					size='small'
					select
					label='Статус'
					value={filter.status ?? ''}
					onChange={e => setFilterField('status', (e.target.value || undefined) as IEquipmentFilter['status'])}
					sx={{ minWidth: 180 }}
				>
					<MenuItem value=''>Все статусы</MenuItem>
					{EQUIPMENT_STATUS_OPTIONS.map(s => (
						<MenuItem key={s.value} value={s.value}>
							{s.label}
						</MenuItem>
					))}
				</TextField>
				<TextField
					size='small'
					select
					label='Площадка'
					value={filter.siteId ?? ''}
					onChange={e => setFilterField('siteId', e.target.value || undefined)}
					sx={{ minWidth: 180 }}
				>
					<MenuItem value=''>Все площадки</MenuItem>
					{sites.map(s => (
						<MenuItem key={s.id} value={s.id}>
							{s.name}
						</MenuItem>
					))}
				</TextField>
			</Stack>

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
								<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Держатель / локация</TableCell>
								<TableCell align='right' sx={{ width: 100 }} />
							</TableRow>
						</TableHead>
						<TableBody>
							{units.length === 0 ? (
								<TableRow>
									<TableCell colSpan={6} align='center' sx={{ py: 6, color: '#6b7280' }}>
										Оборудование не найдено
									</TableCell>
								</TableRow>
							) : (
								units.map(unit => {
									const status = getEquipmentStatus(unit.status)
									return (
										<TableRow
											key={unit.id}
											hover
											sx={{ cursor: 'pointer' }}
											onClick={() => navigate(`/equipment/${unit.id}`)}
										>
											<TableCell sx={{ fontWeight: 500 }}>
												{unit.name}
												{unit.parentId && (
													<Typography
														component='span'
														sx={{ ml: 1, fontSize: 12, color: '#9ca3af' }}
													>
														в комплекте
													</Typography>
												)}
											</TableCell>
											<TableCell sx={{ color: '#6b7280', whiteSpace: 'nowrap' }}>
												{unit.typeName}
											</TableCell>
											<TableCell sx={{ color: '#6b7280' }}>
												{unit.inventoryNumber ?? '—'}
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
														bgcolor: `${status.color}1a`,
														color: status.color,
													}}
												>
													{status.label}
												</Typography>
											</TableCell>
											<TableCell sx={{ color: '#6b7280' }}>{holderLabel(unit)}</TableCell>
											<TableCell
												align='right'
												onClick={e => e.stopPropagation()}
											>
												{canWrite && (
													<>
														<Tooltip title='Редактировать'>
															<IconButton size='small' onClick={() => openEdit(unit)}>
																<PencilIcon sx={{ fontSize: 16 }} />
															</IconButton>
														</Tooltip>
														<Tooltip title='Удалить'>
															<IconButton
																size='small'
																onClick={() => setDeleteUnit(unit)}
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
									)
								})
							)}
						</TableBody>
					</Table>
				</TableContainer>
			</Box>

			{totalPages > 1 && (
				<Box sx={{ display: 'flex', justifyContent: 'center', mt: 3 }}>
					<MuiPagination
						color='primary'
						page={page}
						count={totalPages}
						onChange={(_e, value) => setPage(value)}
					/>
				</Box>
			)}

			<EquipmentDialog open={dialogOpen} onClose={closeDialog} unit={editUnit} />

			<ConfirmDialog
				open={Boolean(deleteUnit)}
				title='Удаление оборудования'
				message={`Вы уверены, что хотите удалить «${deleteUnit?.name}»? Состав комплекта, если есть, будет отсоединён.`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteUnit(null)}
				loading={isDeleting}
			/>
		</Box>
	)
}