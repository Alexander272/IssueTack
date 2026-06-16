import { Box, Typography, Grid, Paper, Skeleton } from '@mui/material'

import { useGetRolesQuery } from '@/features/user/roleApiSlice'
import { useGetAllUsersQuery } from '@/features/user/usersApiSlice'
import { LastActions } from './LastActions'

const statCards: { label: string; key: 'users' | 'active' | 'roles'; color: string }[] = [
	{ label: 'Пользователи', key: 'users', color: '#2196f3' },
	{ label: 'Активные', key: 'active', color: '#4caf50' },
	{ label: 'Роли', key: 'roles', color: '#ff9800' },
]

export const Dashboard = () => {
	const { data: users, isFetching: isFetchingUsers } = useGetAllUsersQuery(null)
	const { data: roles, isFetching: isFetchingRoles } = useGetRolesQuery(null)

	const stats = {
		users: users?.data.length,
		active: users?.data.filter(u => u.isActive).length,
		roles: roles?.data.length,
	}
	const isLoading = isFetchingUsers || isFetchingRoles

	return (
		<Box id='section-dashboard' className='section active' sx={{ p: 3 }}>
			{/* Page Header */}
			<Box className='page-header' sx={{ mb: 4 }}>
				<Typography variant='h4' component='h1' gutterBottom sx={{ fontWeight: 'bold' }}>
					Дашборд
				</Typography>
				<Typography variant='body1' color='text.secondary'>
					Состояние системы доступов
				</Typography>
			</Box>

			{/* Stats Grid */}
			<Grid container spacing={3} sx={{ mb: 4 }}>
				{statCards.map(card => (
					<Grid key={card.key} size={{ xs: 12, sm: 4 }}>
						<Paper sx={{ p: 3, border: '1px solid #eee', borderRadius: 2 }} elevation={0}>
							<Typography variant='caption' color='text.secondary' sx={{ display: 'block', mb: 1 }}>
								{card.label}
							</Typography>

							{/* Если загрузка — показываем скелетон, если нет — значение */}
							{isLoading ? (
								<Skeleton variant='text' width='60%' height={40} />
							) : (
								<Typography variant='h4' sx={{ color: card.color, fontWeight: 'bold' }}>
									{stats?.[card.key] || 0}
								</Typography>
							)}
						</Paper>
					</Grid>
				))}
			</Grid>

			{/* Activity Card */}
			<LastActions />
		</Box>
	)
}
