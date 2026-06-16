import type { FC } from 'react'
import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from '@mui/material'

import { Action } from '../types/resource'
import { useGetResourcesQuery } from '../permApiSlice'

const actions = [
	'Чтение',
	'Создание/Обновление',
	'Удаление',
	// 'Все',
]

export const Permissions = () => {
	const { data, isFetching } = useGetResourcesQuery(null)

	const permissions = data?.data || []

	return (
		<Box sx={{ p: 3 }}>
			{/* Page Header */}
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
				<Box>
					<Typography variant='h4' sx={{ fontWeight: 'bold' }}>
						Права доступа
					</Typography>
					<Typography variant='body1' color='text.secondary'>
						Матрица прав
					</Typography>
				</Box>
			</Box>

			<TableContainer component={Paper} elevation={0} sx={{ borderRadius: '10px', overflow: 'hidden' }}>
				<Table>
					<TableHead>
						<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
							<TableCell sx={{ py: 2.5, px: 2, color: 'text.secondary', fontSize: '0.875rem' }}>
								Ресурс / Действие
							</TableCell>
							{actions.map(head => (
								<TableCell
									key={head}
									align='center'
									sx={{ py: 2.5, px: 2, color: 'text.secondary', fontSize: '0.875rem', width: 290 }}
								>
									{head}
								</TableCell>
							))}
						</TableRow>
					</TableHead>
					<TableBody>
						{permissions.map(row => (
							<TableRow key={row.slug} sx={{ '&:hover': { bgcolor: '#fafafa' } }}>
								<TableCell sx={{ pl: 2, pr: '18px' }}>
									<Typography sx={{ fontWeight: 600, fontSize: '14px', color: '#2c3e50' }}>
										{row.name}
									</Typography>
									<Typography sx={{ fontSize: '12px', color: '#9aa1a9', mt: '4px' }}>
										{row.slug}
									</Typography>
								</TableCell>
								<TableCell align='center'>
									<PermissionBadge allowed={row.actions[Action.Read] || false} />
								</TableCell>
								<TableCell align='center'>
									<PermissionBadge allowed={row.actions[Action.Write] || false} />
								</TableCell>
								<TableCell align='center'>
									<PermissionBadge allowed={row.actions[Action.Delete] || false} />
								</TableCell>
								{/* <TableCell align='center'>
									<PermissionBadge allowed={row.actions[Action.All] || false} />
								</TableCell> */}
							</TableRow>
						))}
						{!permissions.length && !isFetching ? (
							<TableRow>
								<TableCell colSpan={6} align='center' sx={{ py: 3, color: 'text.secondary' }}>
									Права не найдены.
								</TableCell>
							</TableRow>
						) : null}
					</TableBody>
				</Table>
			</TableContainer>
		</Box>
	)
}

const PermissionBadge: FC<{ allowed: boolean }> = ({ allowed }) => (
	<Box
		sx={{
			display: 'inline-flex',
			alignItems: 'center',
			justifyContent: 'center',
			width: 34,
			height: 34,
			borderRadius: '10px',
			fontSize: '18px',
			fontWeight: 'bold',
			bgcolor: allowed ? '#d1f5dd' : '#e5e7eb',
			color: allowed ? '#22c55e' : '#9ca3af',
		}}
	>
		{allowed ? '✓' : '–'}
	</Box>
)
