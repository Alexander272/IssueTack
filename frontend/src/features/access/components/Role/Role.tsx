import { Fragment, useMemo, useState } from 'react'
import {
	Box,
	Button,
	Chip,
	Collapse,
	IconButton,
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
import { pluralize } from '@/utils/plural'
import { useGetRolesWithStatsQuery } from '@/features/user/roleApiSlice'
import { useGetAllRealmsQuery } from '@/features/realms/realmsApiSlice'
import { RoleDialog } from '@/features/user/components/RoleDialog/RoleDialog'
import type { IRoleWithStats } from '@/features/user/types/role'
import type { IRealm } from '@/features/realms/types/realm'
import { stringToHSLA } from '@/utils/colors'
import { BottomArrowIcon } from '@/components/Icons/BottomArrowIcon'

export const Role = () => {
	const { palette } = useTheme()
	const [role, setRole] = useState<string | null>(null)

	const { data, isFetching } = useGetRolesWithStatsQuery()
	const { data: realms } = useGetAllRealmsQuery()

	const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

	const toggleRealm = (realmId: string) => {
		setCollapsed(prev => {
			const next = new Set(prev)
			if (next.has(realmId)) next.delete(realmId)
			else next.add(realmId)
			return next
		})
	}

	const realmMap = useMemo(() => {
		const map = new Map<string, IRealm>()
		realms?.data.forEach(r => map.set(r.id, r))
		return map
	}, [realms])

	const groupedRoles = useMemo(() => {
		if (!data?.data) return []
		const groups = new Map<string, IRoleWithStats[]>()
		data.data.forEach(role => {
			const list = groups.get(role.realm) || []
			list.push(role)
			groups.set(role.realm, list)
		})
		return Array.from(groups, ([realmId, roles]) => ({
			realmId,
			realm: realmMap.get(realmId) || null,
			roles,
			totalUsers: roles.reduce((sum, r) => sum + r.userCount, 0),
		}))
	}, [data, realmMap])

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

			{groupedRoles.map(group => {
				const hsl = stringToHSLA(group.realmId)
				return (
					<Box key={group.realmId} sx={{ mb: 4 }}>
						<Box
							onClick={() => toggleRealm(group.realmId)}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1.5,
								cursor: 'pointer',
								py: 1.5,
								px: 2,
							}}
						>
							<Box
								sx={{
									px: 1.5,
									py: 0.5,
									borderRadius: 1.5,
									fontSize: 12,
									fontWeight: 600,
									textTransform: 'uppercase',
									bgcolor: hsl.bg,
									color: hsl.text,
								}}
							>
								{group.realm?.slug || group.realmId.slice(0, 8)}
							</Box>
							<Typography sx={{ fontWeight: 600, fontSize: 14, color: '#495057' }}>
								{group.realm?.name}
							</Typography>

							<Typography sx={{ color: '#6c757d', fontSize: 13 }}>
								({group.roles.length} {pluralize(group.roles.length, ['роль', 'роли', 'ролей'])},{' '}
								{group.totalUsers}{' '}
								{pluralize(group.totalUsers, ['пользователь', 'пользователя', 'пользователей'])})
							</Typography>

							<IconButton sx={{ ml: 'auto' }}>
								<BottomArrowIcon
									fontSize={16}
									fill='#6c757d'
									transform={collapsed.has(group.realmId) ? 'rotate(180)' : ''}
									transition={'all .3s ease-in-out'}
								/>
							</IconButton>
						</Box>
						<Collapse in={!collapsed.has(group.realmId)}>
							<TableContainer
								component={Paper}
								elevation={0}
								sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden' }}
							>
								<Table sx={{ minWidth: 800 }}>
									<TableHead>
										<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
											<TableCell
												sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}
											>
												Роль
											</TableCell>
											<TableCell
												sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}
											>
												Статус
											</TableCell>
											<TableCell
												sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}
											>
												Наследование
											</TableCell>
											<TableCell
												sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}
											>
												Разрешения
											</TableCell>
											<TableCell
												sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}
											>
												Пользователей
											</TableCell>
											<TableCell align='right' sx={{ p: 0, width: 64 }}></TableCell>
										</TableRow>
									</TableHead>

									<TableBody sx={{ '& tr:not(:last-child)': { borderBottom: '1px solid #f3f4f6' } }}>
										{group.roles.map(role => (
											<TableRow
												key={role.id}
												hover
												sx={{ cursor: 'pointer', '&:hover': { bgcolor: '#fafafa' } }}
											>
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

														<Typography
															sx={{ fontSize: '0.875rem', color: '#6b7280', mt: 0.5 }}
														>
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
													<Box
														sx={{
															display: 'flex',
															alignItems: 'center',
															flexWrap: 'wrap',
															gap: 0.5,
														}}
													>
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
																{index < role.children.length - 1 && (
																	<Typography
																		variant='caption'
																		sx={{ color: '#d1d5db' }}
																	>
																		+
																	</Typography>
																)}
															</Fragment>
														))}

														<ArrowRightIcon
															sx={{ fontSize: 14, color: '#d1d5db', mx: 0.5 }}
														/>

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
														sx={{
															borderStyle: 'dashed',
															color: '#9ca3af',
															fontSize: '11px',
														}}
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
									{!group.roles.length && !isFetching ? (
										<TableRow>
											<TableCell
												colSpan={6}
												align='center'
												sx={{ py: 3, color: 'text.secondary' }}
											>
												Роли не найдены.
											</TableCell>
										</TableRow>
									) : null}
								</TableBody>
							</Table>
						</TableContainer>
					</Collapse>
				</Box>
			)})}
		</Box>
	)
}
