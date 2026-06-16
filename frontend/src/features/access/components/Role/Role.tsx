import { Fragment, useState } from 'react'
import {
	Box,
	Button,
	Chip,
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
import { ArrowRightIcon } from '@mui/x-date-pickers'

import { PlusIcon } from '@/components/Icons/PlusIcon'
import { ModifyIcon } from '@/components/Icons/ModifyIcon'
import { StatusBadge } from '../StatusBadge'
import { useGetRolesWithStatsQuery } from '@/features/user/roleApiSlice'
import { RoleDialog } from '@/features/user/components/RoleDialog/RoleDialog'
import type { IRoleWithStats } from '@/features/user/types/role'

export const Role = () => {
	const { palette } = useTheme()
	const [role, setRole] = useState<string | null>(null)

	const { data, isFetching } = useGetRolesWithStatsQuery(null)

	const createHandler = () => {
		setRole('')
	}

	const editHandler = (id: string) => (e: React.MouseEvent) => {
		e.stopPropagation()
		e.preventDefault()

		setRole(id)
	}

	const clearRole = () => {
		setRole(null)
	}

	const roleMap = new Map<string, IRoleWithStats>()
	data?.data.forEach(role => {
		roleMap.set(role.slug, role)
	})

	return (
		<Box sx={{ p: 3 }}>
			{/* Page Header */}
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
				<Box>
					<Typography variant='h4' sx={{ fontWeight: 'bold' }}>
						Роли
					</Typography>
					<Typography variant='body1' color='text.secondary'>
						Управление ролями и их правами
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

			{/* {data?.data.length && <UpdateRole roleId={data?.data[1].id} />} */}
			<RoleDialog roleId={role} onClose={clearRole} />

			<TableContainer
				component={Paper}
				elevation={0}
				sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden' }}
			>
				<Table sx={{ minWidth: 800 }}>
					<TableHead>
						<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
							<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
								Роль
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Статус
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Наследование
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Разрешения
							</TableCell>
							<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
								Пользователей
							</TableCell>
							<TableCell align='right' sx={{ p: 0, width: 64 }}></TableCell>
						</TableRow>
					</TableHead>

					<TableBody sx={{ '& tr:not(:last-child)': { borderBottom: '1px solid #f3f4f6' } }}>
						{data?.data.map(role => (
							<TableRow key={role.id} hover sx={{ cursor: 'pointer', '&:hover': { bgcolor: '#fafafa' } }}>
								{/* Роль */}
								<TableCell sx={{ py: 2, px: 4 }}>
									<Box>
										<Typography sx={{ fontWeight: 600, color: '#111827' }}>
											{role.name}

											<Typography
												component={'span'}
												sx={{ fontSize: '0.9rem', color: '#9ca3af', ml: 1 }}
											>
												({role.slug})
											</Typography>
										</Typography>

										<Typography sx={{ fontSize: '0.875rem', color: '#6b7280', mt: 0.5 }}>
											{role.description}
										</Typography>
									</Box>
								</TableCell>

								{/* Статус */}
								<TableCell sx={{ px: 3 }}>
									<StatusBadge
										active={role.isActive}
										label={role.isActive ? 'Активна' : 'Неактивна'}
									/>
								</TableCell>

								{/* Наследование */}
								<TableCell sx={{ px: 3 }}>
									{role.children && role.children.length > 0 ? (
										<Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
											{role.children.map((name, index) => (
												<Fragment key={name}>
													<Typography
														variant='caption'
														sx={{
															bgcolor: '#f3f4f6',
															px: 1,
															py: 0.2,
															borderRadius: '4px',
															color: '#6b7280',
														}}
													>
														{roleMap.get(name)?.name || name}
													</Typography>
													{/* Если это не последний родитель, можно поставить какой-то разделитель, но обычно в множественном наследовании они равноправны */}
													{index < role.children.length - 1 && (
														<Typography variant='caption' sx={{ color: '#d1d5db' }}>
															+
														</Typography>
													)}
												</Fragment>
											))}

											<ArrowRightIcon sx={{ fontSize: 14, color: '#d1d5db', mx: 0.5 }} />

											<Typography
												variant='caption'
												sx={{
													fontWeight: 600,
													color: 'primary.main',
													bgcolor: 'rgba(79, 70, 229, 0.08)',
													px: 1,
													py: 0.2,
													borderRadius: '4px',
												}}
											>
												{role.name}
											</Typography>
										</Box>
									) : (
										<Chip
											label='Нет наследования'
											size='small'
											variant='outlined'
											sx={{ borderStyle: 'dashed', color: '#9ca3af', fontSize: '11px' }}
										/>
									)}
								</TableCell>

								{/* Разрешения */}
								<TableCell sx={{ px: 3 }}>
									<Typography sx={{ fontSize: '0.875rem', fontWeight: 500 }}>
										Всего:{' '}
										<Box component='span' sx={{ color: 'primary.main' }}>
											{role.perms.total?.count}
										</Box>
									</Typography>
									<Box sx={{ gap: 1, fontSize: '0.75rem', mt: 0.2 }}>
										<Typography variant='inherit' sx={{ color: '#059669' }}>
											Собственные: {role.perms.own?.count}
										</Typography>
										{role.perms.inherited?.count > 0 && (
											<Typography variant='inherit' sx={{ color: '#2563eb' }}>
												Наследованные: {role.perms.inherited?.count}
											</Typography>
										)}
									</Box>
								</TableCell>

								{/* Кол-во пользователей */}
								<TableCell sx={{ px: 3, fontWeight: 600, color: '#374151' }}>
									{role.userCount}
								</TableCell>

								{/* Действия */}
								<TableCell align='center' sx={{ p: 0, pr: 1 }}>
									{role.isEditable && (
										<Tooltip title='Редактировать роль'>
											<Button
												onClick={editHandler(role.id)}
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
									)}
								</TableCell>
							</TableRow>
						))}
						{!data?.data.length && !isFetching ? (
							<TableRow>
								<TableCell colSpan={6} align='center' sx={{ py: 3, color: 'text.secondary' }}>
									Роли не найдены.
								</TableCell>
							</TableRow>
						) : null}
					</TableBody>
				</Table>
			</TableContainer>
		</Box>
	)
}
