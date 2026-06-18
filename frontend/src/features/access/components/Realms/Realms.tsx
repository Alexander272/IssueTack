import {
	Box,
	Button,
	Paper,
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

import type { IRealm } from '@/features/realms/types/realm'
import { getSmartDate } from '@/utils/date'
import { useGetAllRealmsQuery } from '@/features/realms/realmsApiSlice'
import { PlusIcon } from '@/components/Icons/PlusIcon'
import { ModifyIcon } from '@/components/Icons/ModifyIcon'
import { RealmDialog } from '@/features/realms/components/Dialogs/RealmDialog'
import { useState } from 'react'
import { StatusBadge } from '../StatusBadge'

export const Realms = () => {
	const { palette } = useTheme()

	const [open, setOpen] = useState(false)
	const [realm, setRealm] = useState<IRealm | null>(null)

	const { data, isFetching } = useGetAllRealmsQuery()

	const createHandler = () => {
		setOpen(true)
	}

	const editHandler = (realm: IRealm) => () => {
		setRealm(realm)
		setOpen(true)
	}

	const clearHandler = () => {
		setRealm(null)
		setOpen(false)
	}

	return (
		<Box sx={{ p: 3 }}>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
				<Box>
					<Typography variant='h4' sx={{ fontWeight: 'bold' }}>
						Области
					</Typography>
					<Typography variant='body1' color='text.secondary'>
						Изолированные контуры данных для Multi-Tenancy архитектуры
					</Typography>
				</Box>
				<Button
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff' }}
					onClick={createHandler}
				>
					<PlusIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
					Добавить
				</Button>
			</Box>

			<RealmDialog realm={realm || undefined} open={open} onClose={clearHandler} />

			<TableContainer
				component={Paper}
				elevation={0}
				sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden' }}
			>
				<Table sx={{ minWidth: 800 }}>
					<TableHead>
						<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
							<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
								Область
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
								Ключ / Субдомен
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Описание
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Статус
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Пользователей
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Создано
							</TableCell>
							<TableCell align='right' sx={{ p: 0, width: 64 }}></TableCell>
						</TableRow>
					</TableHead>

					<TableBody sx={{ '& tr:not(:last-child)': { borderBottom: '1px solid #f3f4f6' } }}>
						{data?.data.map(r => (
							<TableRow key={r.id} hover sx={{ cursor: 'pointer', '&:hover': { bgcolor: '#fafafa' } }}>
								<TableCell sx={{ py: 2, px: 4 }}>
									<Typography sx={{ fontWeight: 600, color: '#111827' }}>{r.name}</Typography>
								</TableCell>

								<TableCell sx={{ py: 2, px: 4 }}>
									<Typography sx={{ fontWeight: 600, color: '#111827' }}>{r.slug}</Typography>
								</TableCell>

								<TableCell sx={{ py: 2, px: 4 }}>
									<Typography sx={{ fontSize: '0.875rem', color: '#6b7280', mt: 0.5 }}>
										{r.description}
									</Typography>
								</TableCell>

								{/* Статус */}
								<TableCell sx={{ px: 3 }}>
									<StatusBadge active={r.isActive} label={r.isActive ? 'Активна' : 'Неактивна'} />
								</TableCell>

								{/* Кол-во пользователей */}
								<TableCell sx={{ px: 3, fontWeight: 600, color: '#374151' }}>{r.userCount}</TableCell>

								{/* Дата создания */}
								<TableCell sx={{ px: 3, fontWeight: 600, color: '#374151' }}>
									{getSmartDate(r.createdAt)}
								</TableCell>

								{/* Действия */}
								<TableCell align='center' sx={{ p: 0, pr: 1 }}>
									<Tooltip title='Редактировать область'>
										<Button
											onClick={editHandler(r)}
											sx={{
												minWidth: 60,
												minHeight: 60,
												borderRadius: '6px',
												':hover': { svg: { fill: palette.secondary.main } },
											}}
										>
											<ModifyIcon sx={{ fontSize: 18 }} />
										</Button>
									</Tooltip>
								</TableCell>
							</TableRow>
						))}
						{!data?.data.length && !isFetching ? (
							<TableRow>
								<TableCell colSpan={6} align='center' sx={{ py: 3, color: 'text.secondary' }}>
									Области не найдены.
								</TableCell>
							</TableRow>
						) : null}
					</TableBody>
				</Table>
			</TableContainer>
		</Box>
	)
}
