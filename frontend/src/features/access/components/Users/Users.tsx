import { useMemo, useState } from 'react'
import {
	Box,
	Typography,
	Button,
	TextField,
	MenuItem,
	Select,
	InputAdornment,
	type SelectChangeEvent,
	useTheme,
	CircularProgress,
	Stack,
} from '@mui/material'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IUserData } from '@/features/user/types/user'
import { useDebounce } from '@/hooks/useDebounce'
import { useGetAllUsersQuery, useSyncUsersMutation } from '@/features/user/usersApiSlice'
import { useGetRolesQuery } from '@/features/user/roleApiSlice'
import { UpdateModal } from '@/features/user/components/Update'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { SyncIcon } from '@/components/Icons/SyncIcon'
import { UserCard } from './UserCard'

export const Users = () => {
	const { palette } = useTheme()
	const [search, setSearch] = useState('')
	const [roleFilter, setRoleFilter] = useState<string[]>([''])
	const [statusFilter, setStatusFilter] = useState('')

	const [modalType, setModalType] = useState<'edit' | 'logins'>('edit')
	const [user, setUser] = useState<IUserData | null>(null)

	const debouncedSearch = useDebounce(search, 300)

	const { data, isFetching } = useGetAllUsersQuery()
	const { data: roles, isFetching: isFetchingRoles } = useGetRolesQuery()
	const [sync, { isLoading }] = useSyncUsersMutation()

	const filteredUsers = useMemo(() => {
		if (!data?.data) return []

		const lowSearch = (debouncedSearch as string).toLowerCase().trim()

		return data.data.filter(user => {
			const matchesSearch =
				!lowSearch ||
				[user.lastName, user.firstName, user.email].some(field => field?.toLowerCase().includes(lowSearch))

			const matchesRole =
				roleFilter.length === 0 || roleFilter.includes('')
					? true
					: user.realms?.some(realm => roleFilter.includes(realm.role?.name || ''))

			const matchesStatus =
				!statusFilter || statusFilter === ''
					? true
					: statusFilter === 'active'
						? user.realms?.some(realm => realm.isActive)
						: user.realms?.every(realm => !realm.isActive)

			return matchesSearch && matchesRole && matchesStatus
		})
	}, [data, debouncedSearch, roleFilter, statusFilter])

	const syncHandler = async () => {
		try {
			await sync().unwrap()
			toast.success('Пользователи синхронизированы')
		} catch (error) {
			const err = error as IFetchError
			toast.error(err.data.message, { autoClose: false })
		}
	}

	const roleHandler = (event: SelectChangeEvent<string[]>) => {
		const value = event.target.value
		let newValue = typeof value === 'string' ? value.split(',') : value

		if (newValue.length === 0) {
			newValue = ['']
		} else if (newValue.length > 1) {
			if (newValue.includes('')) {
				newValue = newValue.filter(v => v !== '')
			}

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
			{/* <LoginsModal user={modalType == 'logins' ? user : null} onClose={() => setUser(null)} /> */}

			{/* Cards */}
			<Stack spacing={2}>
				{filteredUsers.map(u => (
					<UserCard key={u.id} user={u} onClick={userHandler} />
				))}
				{!data?.data.length && !isFetching ? (
					<Typography align='center' sx={{ py: 3, color: 'text.secondary' }}>
						Пользователи не найдены.
					</Typography>
				) : null}
			</Stack>
		</Box>
	)
}
