import { type FC, useMemo, useState } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableRow,
	TextField,
	Typography,
} from '@mui/material'
import { FileTextIcon, PlusIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment, IEquipmentAct } from '../../types/equipment'
import {
	useCreateEquipmentActMutation,
	useGetAllEquipmentActsQuery,
	useGetAllEquipmentQuery,
} from '../../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'

const fmtDateTime = (d: string | null) => (d ? new Date(d).toLocaleString('ru-RU') : '—')

const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

type Props = {
	unit: IEquipment
}

export const EquipmentItemActs: FC<Props> = ({ unit }) => {
	const { data } = useGetAllEquipmentActsQuery({ limit: 100 })
	const acts = useMemo(
		() =>
			(data?.data ?? []).filter(
				a => a.kitEquipmentId === unit.id || (a.content ?? []).some(p => p.equipmentId === unit.id),
			),
		[data, unit.id],
	)

	const [detail, setDetail] = useState<IEquipmentAct | null>(null)
	const [createOpen, setCreateOpen] = useState(false)

	return (
		<Box>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
				<Typography variant='subtitle1' sx={{ fontWeight: 600, color: '#1f2937' }}>
					Акты {unit.isKit ? 'комплекта' : 'по единице'}
				</Typography>
				{unit.isKit && (
					<Button
						variant='outlined'
						size='small'
						startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
						onClick={() => setCreateOpen(true)}
						sx={{ textTransform: 'none', borderRadius: '8px' }}
					>
						Сформировать акт
					</Button>
				)}
			</Box>

			<Box sx={{ border: '1px solid #eef2f6', borderRadius: '12px', overflow: 'hidden', bgcolor: '#fff' }}>
				<Table size='small'>
					<TableHead>
						<TableRow sx={{ bgcolor: '#fafbfc' }}>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Номер</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Дата</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Получатель</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Позиций</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{acts.length === 0 ? (
							<TableRow>
								<TableCell colSpan={4} align='center' sx={{ py: 4, color: '#6b7280' }}>
									Связанных актов нет
								</TableCell>
							</TableRow>
						) : (
							acts.map(act => (
								<TableRow key={act.id} hover sx={{ cursor: 'pointer' }} onClick={() => setDetail(act)}>
									<TableCell sx={{ fontWeight: 500 }}>{act.number}</TableCell>
									<TableCell sx={{ whiteSpace: 'nowrap' }}>{fmtDateTime(act.issuedAt)}</TableCell>
									<TableCell>{act.recipientName || '—'}</TableCell>
									<TableCell>{act.content?.length ?? 0}</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</Box>

			{unit.isKit && createOpen && (
				<CreateActDialog open onClose={() => setCreateOpen(false)} defaultKit={unit} />
			)}

			<Dialog open={Boolean(detail)} onClose={() => setDetail(null)} fullWidth maxWidth='sm'>
				<DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 2 }}>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
						<FileTextIcon sx={{ fontSize: 20, color: '#2563eb' }} />
						<Typography variant='h6' sx={{ fontWeight: 'bold' }}>
							{detail?.number}
						</Typography>
					</Box>
					<IconButton onClick={() => setDetail(null)}>
						<XIcon sx={{ fontSize: 16 }} />
					</IconButton>
				</DialogTitle>
				<DialogContent dividers>
					<Typography variant='body2' sx={{ color: '#6b7280', mb: 1 }}>
						Выдан: {fmtDateTime(detail?.issuedAt ?? null)}
						{detail?.recipientName ? ` · ${detail.recipientName}` : ''}
					</Typography>
					<Table size='small'>
						<TableHead>
							<TableRow>
								<TableCell>Наименование</TableCell>
								<TableCell>Тип</TableCell>
								<TableCell>Инв. №</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{(detail?.content ?? []).map(p => (
								<TableRow key={p.equipmentId}>
									<TableCell>{p.name}</TableCell>
									<TableCell>{p.typeName}</TableCell>
									<TableCell>{p.inventoryNumber ?? '—'}</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</DialogContent>
			</Dialog>
		</Box>
	)
}

const CreateActDialog: FC<{ open: boolean; onClose: () => void; defaultKit?: IEquipment }> = ({
	open,
	onClose,
	defaultKit,
}) => {
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 }, { skip: !open })
	const { data: usersData } = useGetRealmUsersQuery('executors', { skip: !open })
	const { data: sitesData } = useGetAllSitesQuery(undefined, { skip: !open })
	const [create, { isLoading }] = useCreateEquipmentActMutation()

	const kits = useMemo(() => (unitsData?.data ?? []).filter(u => u.isKit), [unitsData])
	const users = usersData?.data ?? []
	const sites = sitesData?.data ?? []

	const [kit, setKit] = useState<IEquipment | null>(defaultKit ?? null)
	const [userId, setUserId] = useState<string | null>(null)
	const [siteId, setSiteId] = useState<string | null>(null)
	const [location, setLocation] = useState('')
	const [comment, setComment] = useState('')
	const [error, setError] = useState('')

	const handleCreate = async () => {
		if (!kit) {
			setError('Выберите комплект')
			return
		}
		try {
			await create({
				kitEquipmentId: kit.id,
				recipientUserId: userId,
				siteId,
				location: location || null,
				comment,
			}).unwrap()
			toast.success('Акт сформирован')
			setKit(defaultKit ?? null)
			setUserId(null)
			setSiteId(null)
			setLocation('')
			setComment('')
			setError('')
			onClose()
		} catch (err) {
			const fetchError = err as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при создании акта', { autoClose: false })
		}
	}

	return (
		<Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
			<DialogTitle>Формирование акта</DialogTitle>
			<DialogContent dividers>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
					<Autocomplete
						options={kits}
						getOptionLabel={u => u.name}
						value={kit}
						onChange={(_e, value) => setKit(value)}
						renderInput={params => (
							<TextField {...params} size='small' label='Комплект' required error={Boolean(error)} helperText={error} />
						)}
					/>
					<Autocomplete
						options={users}
						getOptionLabel={userLabel}
						value={users.find(u => u.id === userId) ?? null}
						onChange={(_e, value) => setUserId(value?.id ?? null)}
						renderInput={params => <TextField {...params} size='small' label='Получатель' />}
					/>
					<Autocomplete
						options={sites}
						getOptionLabel={s => s.name}
						value={sites.find(s => s.id === siteId) ?? null}
						onChange={(_e, value) => setSiteId(value?.id ?? null)}
						renderInput={params => <TextField {...params} size='small' label='Площадка' />}
					/>
					<TextField size='small' label='Локация' value={location} onChange={e => setLocation(e.target.value)} />
					<TextField
						size='small'
						label='Комментарий'
						value={comment}
						onChange={e => setComment(e.target.value)}
						multiline
						minRows={2}
					/>
				</Box>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose} sx={{ textTransform: 'none' }}>
					Отмена
				</Button>
				<Button onClick={handleCreate} variant='contained' disabled={isLoading} sx={{ textTransform: 'none' }}>
					Сформировать
				</Button>
			</DialogActions>
		</Dialog>
	)
}
