import { type FC, useState } from 'react'
import {
	Box,
	IconButton,
	List,
	ListItem,
	ListItemText,
	TextField,
	Tooltip,
	Typography,
} from '@mui/material'
import { CheckIcon, MapPinIcon, PencilIcon, PlusIcon, Trash2Icon, XIcon } from 'lucide-mui'

import type { ISiteLocation } from '../types/site'
import {
	useCreateSiteLocationMutation,
	useDeleteSiteLocationMutation,
	useGetSiteLocationsQuery,
	useUpdateSiteLocationMutation,
} from '../sitesApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	siteId: string
	disabled?: boolean
}

const labelSx = { fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }

// Расположения площадки — единственное место, где они создаются, переименовываются
// и удаляются. Формы оборудования только выбирают готовые записи из селекта.
// Ошибки бэкенда (в том числе EQ025 при удалении используемого расположения)
// показывает onQueryStarted мутации; состояние здесь остаётся неизменным, чтобы
// пользователь мог поправить значение или отменить действие.
export const SiteLocationsSection: FC<Props> = ({ siteId, disabled = false }) => {
	const { data } = useGetSiteLocationsQuery(siteId, { skip: !siteId })
	const locations = data?.data ?? []

	const [newName, setNewName] = useState('')
	const [editingId, setEditingId] = useState<string | null>(null)
	const [editingName, setEditingName] = useState('')
	const [deleteTarget, setDeleteTarget] = useState<ISiteLocation | null>(null)

	const [create, { isLoading: isCreating }] = useCreateSiteLocationMutation()
	const [update, { isLoading: isUpdating }] = useUpdateSiteLocationMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteSiteLocationMutation()

	const busy = isCreating || isUpdating || isDeleting || disabled

	const handleCreate = async () => {
		const name = newName.trim()
		if (!name || busy) return
		try {
			await create({ siteId, body: { name } }).unwrap()
			setNewName('')
		} catch {
			// Тост с текстом ошибки уже показан в sitesApiSlice.
		}
	}

	const startEdit = (location: ISiteLocation) => {
		setEditingId(location.id)
		setEditingName(location.name)
	}

	const handleUpdate = async () => {
		const name = editingName.trim()
		if (!editingId || !name || busy) return
		try {
			await update({ siteId, locationId: editingId, body: { name } }).unwrap()
			setEditingId(null)
		} catch {
			// Тост с текстом ошибки уже показан в sitesApiSlice.
		}
	}

	const handleDelete = async () => {
		if (!deleteTarget) return
		try {
			await remove({ siteId, locationId: deleteTarget.id }).unwrap()
			setDeleteTarget(null)
		} catch {
			// При конфликте (расположение используется) диалог остаётся открытым —
			// пользователь видит причину и может отменить удаление.
		}
	}

	return (
		<Box sx={{ mt: 1 }}>
			<Typography variant='caption' sx={labelSx}>
				Расположения площадки
			</Typography>
			<Typography variant='caption' sx={{ display: 'block', color: '#9ca3af', mb: 1 }}>
				Создавайте здесь комнаты, стеллажи и склады — в оборудовании их только выбирают
			</Typography>

			{locations.length === 0 ? (
				<Typography variant='body2' sx={{ color: '#6b7280', mb: 1 }}>
					Расположения не добавлены
				</Typography>
			) : (
				<List dense sx={{ py: 0, mb: 1 }}>
					{locations.map(location =>
						editingId === location.id ? (
							<ListItem
								key={location.id}
								sx={{ border: '1px solid #e5e7eb', borderRadius: '8px', mb: 0.5, py: 0.25 }}
								secondaryAction={
									<>
										<Tooltip title='Сохранить'>
											<span>
												<IconButton
													type='button'
													size='small'
													disabled={busy || !editingName.trim()}
													onClick={handleUpdate}
												>
													<CheckIcon sx={{ fontSize: 18, color: '#16a34a' }} />
												</IconButton>
											</span>
										</Tooltip>
										<Tooltip title='Отмена'>
											<IconButton
												type='button'
												size='small'
												onClick={() => setEditingId(null)}
											>
												<XIcon sx={{ fontSize: 18, color: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
									</>
								}
							>
								<TextField
									size='small'
									fullWidth
									autoFocus
									value={editingName}
									onChange={event => setEditingName(event.target.value)}
									onKeyDown={event => {
										// Внутри формы площадки Enter не должен её отправить.
										if (event.key === 'Enter') {
											event.preventDefault()
											void handleUpdate()
										}
										if (event.key === 'Escape') setEditingId(null)
									}}
								/>
							</ListItem>
						) : (
							<ListItem
								key={location.id}
								sx={{ borderBottom: '1px solid #f3f4f6', py: 0.5 }}
								secondaryAction={
									<>
										<Tooltip title='Переименовать'>
											<IconButton type='button' size='small' disabled={busy} onClick={() => startEdit(location)}>
												<PencilIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
										<Tooltip title='Удалить'>
											<IconButton
												type='button'
												size='small'
												disabled={busy}
												onClick={() => setDeleteTarget(location)}
											>
												<Trash2Icon sx={{ fontSize: 16, color: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
									</>
								}
							>
								<ListItemText
									primary={
										<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
											<MapPinIcon sx={{ fontSize: 16, color: '#6b7280' }} />
											{location.name}
										</Box>
									}
								/>
							</ListItem>
						),
					)}
				</List>
			)}

			<Box sx={{ display: 'flex', gap: 1 }}>
				<TextField
					size='small'
					fullWidth
					placeholder='Новое расположение'
					value={newName}
					disabled={busy}
					onChange={event => setNewName(event.target.value)}
					onKeyDown={event => {
						if (event.key === 'Enter') {
							event.preventDefault()
							void handleCreate()
						}
					}}
				/>
				<IconButton
					type='button'
					color='primary'
					disabled={busy || !newName.trim()}
					onClick={handleCreate}
					sx={{ border: '1px solid #c7d2fe', borderRadius: '8px' }}
				>
					<PlusIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</Box>

			<ConfirmDialog
				open={Boolean(deleteTarget)}
				title='Удаление расположения'
				message={`Удалить расположение «${deleteTarget?.name ?? ''}»? Если оно указано у оборудования, в выдачах или актах, удаление будет отказано.`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteTarget(null)}
				loading={isDeleting}
			/>
		</Box>
	)
}
