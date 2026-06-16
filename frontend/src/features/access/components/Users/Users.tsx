import { useMemo, useState, type FC } from 'react'
import {
	Box,
	Typography,
	Button,
	TextField,
	MenuItem,
	Select,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Paper,
	InputAdornment,
	type SelectChangeEvent,
	useTheme,
	Avatar,
	Chip,
	Tooltip,
	CircularProgress,
} from '@mui/material'
import dayjs from 'dayjs'

import type { IUserData } from '@/features/user/types/user'
import { getAvatarColor, getInitials } from './utils'
import { stringToHSLA } from '@/utils/colors'
import { getSmartDate } from '@/utils/date'
import { useDebounce } from '@/hooks/useDebounce'
import { useGetAllUsersQuery, useSyncUsersMutation } from '@/features/user/usersApiSlice'
import { useGetRolesQuery } from '@/features/user/roleApiSlice'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { SyncIcon } from '@/components/Icons/SyncIcon'
import { ModifyIcon } from '@/components/Icons/ModifyIcon'
import { StatusBadge } from '../StatusBadge'
import { UpdateModal } from '../../../user/components/Update'
import { LoginsModal } from './LoginsModal'

export const Users = () => {
	const { palette } = useTheme()
	const [search, setSearch] = useState('')
	const [roleFilter, setRoleFilter] = useState<string[]>([''])
	const [statusFilter, setStatusFilter] = useState('')

	const [modalType, setModalType] = useState<'edit' | 'logins'>('edit')
	const [user, setUser] = useState<IUserData | null>(null)

	const debouncedSearch = useDebounce(search, 300)

	const { data, isFetching } = useGetAllUsersQuery(null)
	const { data: roles, isFetching: isFetchingRoles } = useGetRolesQuery(null)
	const [sync, { isLoading }] = useSyncUsersMutation()

	const filteredUsers = useMemo(() => {
		if (!data) return []

		const lowSearch = (debouncedSearch as string).toLowerCase().trim()

		return data?.data.filter(user => {
			// 1. Поиск (по имени или почте)
			const matchesSearch =
				!lowSearch ||
				[user.lastName, user.firstName, user.email].some(field => field?.toLowerCase().includes(lowSearch))

			// 2. Фильтр по ролям (проверяем, есть ли роль пользователя в массиве выбранных)
			// Если массив пустой, обычно показывают всех (или никого, зависит от вашей логики)
			const matchesRole = !roleFilter.includes('') ? roleFilter.includes(user.role) : true

			// 3. Фильтр по статусу
			const matchesStatus =
				statusFilter && statusFilter !== ''
					? statusFilter === 'active'
						? user.isActive
						: !user.isActive
					: true

			return matchesSearch && matchesRole && matchesStatus
		})
	}, [data, debouncedSearch, roleFilter, statusFilter])

	const syncHandler = async () => {
		await sync(null)
	}

	const roleHandler = (event: SelectChangeEvent<string[]>) => {
		const value = event.target.value
		let newValue = typeof value === 'string' ? value.split(',') : value

		// 1. Если список стал пустым — возвращаем ''
		if (newValue.length === 0) {
			newValue = ['']
		}
		// 2. Если в списке больше одного элемента
		else if (newValue.length > 1) {
			// Если только что добавили что-то к '', то убираем ''
			if (newValue.includes('')) {
				newValue = newValue.filter(v => v !== '')
			}

			// 3. Если выбраны все доступные опции (кроме ''),
			// тут можно добавить условие сравнения с длиной исходного массива ролей
			if (newValue.length === roles?.data.length) {
				newValue = ['']
			}
		}

		setRoleFilter(newValue)
	}

	const userHandler = (user: IUserData | null, type: 'edit' | 'logins') => {
		setUser(user)
		setModalType(type)
	}

	return (
		<Box sx={{ p: 3 }}>
			{/* Page Header */}
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
				<Box>
					<Typography variant='h4' sx={{ fontWeight: 'bold' }}>
						Пользователи
					</Typography>
					<Typography variant='body1' color='text.secondary'>
						Управление учётными записями
					</Typography>
				</Box>
				<Button
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff' }}
					onClick={syncHandler}
				>
					{isLoading ? (
						<CircularProgress size={16} sx={{ mr: 1.5 }} />
					) : (
						<SyncIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
					)}
					Синхронизировать
				</Button>
			</Box>

			{isFetching || isFetchingRoles || isLoading ? <BoxFallback /> : null}

			{/* Toolbar */}
			<Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
				<TextField
					placeholder='Поиск по имени или email…'
					size='small'
					value={search}
					onChange={e => setSearch(e.target.value)}
					slotProps={{
						input: {
							startAdornment: (
								<InputAdornment position='start'>
									<SearchIcon fontSize='small' />
								</InputAdornment>
							),
						},
					}}
					sx={{ flexGrow: 1, minWidth: '200px', background: '#fff' }}
				/>

				<Select
					size='small'
					displayEmpty
					multiple
					value={roleFilter}
					onChange={roleHandler}
					sx={{ width: '400px', background: '#fff' }}
				>
					<MenuItem value='' disabled>
						Все роли
					</MenuItem>
					{roles?.data.map(role => (
						<MenuItem key={role.id} value={role.name}>
							{role.name}
						</MenuItem>
					))}
				</Select>

				<Select
					size='small'
					displayEmpty
					value={statusFilter}
					onChange={e => setStatusFilter(e.target.value)}
					sx={{ width: '300px', background: '#fff' }}
				>
					<MenuItem value=''>Все статусы</MenuItem>
					<MenuItem value='active'>Активные</MenuItem>
					<MenuItem value='inactive'>Неактивные</MenuItem>
				</Select>
			</Box>

			<UpdateModal user={modalType == 'edit' ? user : null} onClose={() => setUser(null)} />
			<LoginsModal user={modalType == 'logins' ? user : null} onClose={() => setUser(null)} />

			{/* Table Container */}
			<TableContainer component={Paper} elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
				<Table>
					<TableHead>
						<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
							<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>Пользователь</TableCell>
							<TableCell
								align='center'
								sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 250 }}
							>
								Роль
							</TableCell>
							<TableCell
								align='center'
								sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 200 }}
							>
								Статус
							</TableCell>
							<TableCell
								align='center'
								sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 250 }}
							>
								Создан
							</TableCell>
							<TableCell
								align='center'
								sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 250 }}
							>
								Последний вход
							</TableCell>
							<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 100 }}>
								Действия
							</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{filteredUsers.map(user => (
							<UserRow key={user.id} u={user} setUser={userHandler} />
						))}
						{!data?.data.length && !isFetching ? (
							<TableRow>
								<TableCell colSpan={6} align='center' sx={{ py: 3, color: 'text.secondary' }}>
									Пользователи не найдены.
								</TableCell>
							</TableRow>
						) : null}
					</TableBody>
				</Table>
			</TableContainer>
		</Box>
	)
}

type RowProps = {
	u: IUserData
	setUser: (user: IUserData | null, type: 'edit' | 'logins') => void
}

const UserRow: FC<RowProps> = ({ u, setUser }) => {
	const { palette } = useTheme()
	const colors = useMemo(() => stringToHSLA(u.role), [u.role])

	const editHandler = (e: React.MouseEvent) => {
		e.stopPropagation()
		e.preventDefault()

		setUser(u, 'edit')
	}

	const openLoginsHandler = (e: React.MouseEvent) => {
		e.stopPropagation()
		e.preventDefault()

		setUser(u, 'logins')
	}

	return (
		<TableRow key={u.id} hover>
			{/* Пользователь */}
			<TableCell>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
					<Avatar
						sx={{
							bgcolor: getAvatarColor(u.id), // ваша функция
							fontSize: '14px',
							width: 36,
							height: 36,
						}}
					>
						{getInitials(u)} {/* ваша функция */}
					</Avatar>
					<Box>
						<Typography variant='body2' sx={{ fontWeight: 500 }}>
							{u.firstName} {u.lastName}
						</Typography>
						<Typography variant='caption' color='text.secondary' sx={{ display: 'block' }}>
							{u.email}
						</Typography>
					</Box>
				</Box>
			</TableCell>

			{/* Роль */}
			<TableCell align='center'>
				<Chip
					label={u.role}
					size={'small'}
					style={{
						backgroundColor: colors.bg,
						color: colors.text,
						border: `1px solid ${colors.border}`,
						fontWeight: 500,
						fontSize: '0.75rem',
						height: '20px',
						borderRadius: '6px',
					}}
				/>
			</TableCell>

			{/* Статус */}
			<TableCell align='center'>
				<StatusBadge active={u.isActive} label={u.isActive ? 'Активный' : 'Неактивный'} />
			</TableCell>

			{/* Создан */}
			<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '13px' }}>
				{dayjs(u.createdAt).format('dddd, DD MMM YYYY HH:mm')}
			</TableCell>

			{/* Последний вход */}
			<TableCell
				onClick={openLoginsHandler}
				align='center'
				sx={{
					color: 'text.secondary',
					fontSize: '13px',
					cursor: 'pointer',
					transition: 'all 0.3s ease-in-out',
					':hover': { color: palette.secondary.main },
				}}
			>
				{getSmartDate(u.lastVisit)}
			</TableCell>

			{/* Действия */}
			<TableCell align='center' sx={{ p: 0 }}>
				<Tooltip title='Редактировать пользователя'>
					<Button
						onClick={editHandler}
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
	)
}
