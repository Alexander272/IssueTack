import type { FC } from 'react'
import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from '@mui/material'

import { useGetPermissionsQuery } from '../permApiSlice'

const actions = [
	{ title: 'Чтение', action: 'read' },
	{ title: 'Добавление/Изменение', action: 'write' },
	{ title: 'Удаление', action: 'delete' },
]

export const Permissions = () => {
	const { data, isFetching } = useGetPermissionsQuery()

	const permissions = data?.data || []

	return (
		<Box sx={{ p: 3 }}>
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
							{actions.map(a => (
								<TableCell
									key={a.action}
									align='center'
									sx={{ py: 2.5, px: 2, color: 'text.secondary', fontSize: '0.875rem', width: 290 }}
								>
									{a.title}
								</TableCell>
							))}
						</TableRow>
					</TableHead>
					<TableBody>
						{permissions.map(row => (
							<TableRow key={row.group} sx={{ '&:hover': { bgcolor: '#fafafa' } }}>
								<TableCell sx={{ pl: 2, pr: '18px' }}>
									<Typography sx={{ fontWeight: 600, fontSize: '14px', color: '#2c3e50' }}>
										{row.title}
									</Typography>
									<Typography sx={{ fontSize: '12px', color: '#9aa1a9', mt: '4px' }}>
										{row.group}
									</Typography>
								</TableCell>
								{actions.map(a => (
									<TableCell key={a.action} align='center'>
										<PermissionBadge
											allowed={row.items.find(i => i.action === a.action) ? true : false}
										/>
									</TableCell>
								))}
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
