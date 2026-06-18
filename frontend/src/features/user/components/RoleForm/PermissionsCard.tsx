import { type FC } from 'react'
import {
	Box,
	Card,
	CardContent,
	Stack,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
	Tooltip,
	type SxProps,
} from '@mui/material'
import { useFormContext, useWatch } from 'react-hook-form'

import type { IRolePermissionItem, IRoleWithPerms } from '../../types/role'
import type { IForm } from './UpdateRole'
import { getBadge } from './utils'
import { useGetPermissionsQuery } from '@/features/access/permApiSlice'
import { Fallback } from '@/components/Fallback/Fallback'

type Props = {
	role?: IRoleWithPerms
}

const actions = [
	{ slug: 'read' as const, title: 'Чтение' },
	{ slug: 'write' as const, title: 'Создание/Обновление' },
	{ slug: 'delete' as const, title: 'Удаление' },
	// { slug: '*' as const, title: 'Все' },
]

export const PermissionsCard: FC<Props> = ({ role }) => {
	const { data, isFetching } = useGetPermissionsQuery()

	const { control, setValue } = useFormContext<IForm>()
	const selected = useWatch({ control, name: 'perms' })

	const resMap = new Map<string, Map<string, IRolePermissionItem>>()
	role?.perms.forEach(p => {
		const actionMap = new Map<string, IRolePermissionItem>()
		p.resources.forEach(r => {
			actionMap.set(r.action, r)
		})
		resMap.set(p.group, actionMap)
	})
	const selectedMap = new Map<string, Map<string, IRolePermissionItem>>()
	selected?.forEach(p => {
		const actionMap = new Map<string, IRolePermissionItem>()
		p.resources.forEach(r => {
			actionMap.set(r.action, r)
		})
		selectedMap.set(p.group, actionMap)
	})

	const togglePermissionHandler = (group: string, perm: IRolePermissionItem) => {
		console.log('toggle')
		perm.isAssigned = !perm.isAssigned

		const orig = resMap.get(group)?.get(perm.action)
		perm.status = (orig?.isAssigned ?? false) !== perm.isAssigned ? 'changed' : 'original'

		const newSelected = Array.from(selectedMap, ([group, actionMap]) => ({
			group,
			resources: Array.from(actionMap.values()),
		}))

		setValue('perms', newSelected)
	}

	const toggleAllPermissionsHandler = (state: boolean) => {
		console.log('toggle all', state)

		data?.data.forEach(row => {
			const groupName = row.group
			const groupResMap = resMap.get(groupName)
			let actionMap: Map<string, IRolePermissionItem> | null = null

			row.items.forEach(item => {
				const baseItem = groupResMap?.get(item.action)
				if (baseItem?.isInherited) return
				if (!actionMap) {
					actionMap = new Map(selectedMap.get(groupName) || [])
				}
				const currentItem = actionMap.get(item.action)
				if (currentItem?.isAssigned === state) return

				actionMap.set(item.action, {
					permissionId: currentItem?.permissionId || '',
					object: groupName,
					action: item.action,
					isInherited: false,
					isAssigned: state,
					status: (baseItem?.isAssigned ?? false) == state ? 'original' : 'changed',
				})
			})
			if (actionMap) {
				selectedMap.set(groupName, actionMap)
			}
		})

		const newSelected = Array.from(selectedMap, ([group, actionMap]) => ({
			group,
			resources: Array.from(actionMap.values()),
		}))
		setValue('perms', newSelected)
	}

	const groupToggleHandler = (group: string, item: IRolePermissionItem) => {
		console.log('toggle group', group, item)

		const actions = selectedMap.get(group)
		if (!actions) return

		const actionMap = new Map(actions)
		const groupResMap = resMap.get(group)

		for (const a of actions.values()) {
			const existingItem = actionMap.get(a.action)
			const orig = groupResMap?.get(a.action)

			// Если состояние уже совпадает с целевым, пропускаем итерацию
			if (existingItem?.isAssigned !== item.isAssigned) continue

			actionMap.set(a.action, {
				permissionId: existingItem?.permissionId ?? '',
				object: a.object,
				action: a.action,
				isInherited: existingItem?.isInherited ?? false,
				isAssigned: !item.isAssigned,
				status: (orig?.isAssigned ?? false) !== item.isAssigned ? 'original' : 'changed',
			})
		}

		selectedMap.set(group, actionMap)

		const newSelected = Array.from(selectedMap, ([group, map]) => ({
			group,
			resources: Array.from(map.values()),
		}))

		setValue('perms', newSelected)
	}

	if (isFetching) return <Fallback />
	return (
		<Card sx={{ mb: 1.5, borderRadius: '16px', border: '1px solid #f3f4f6' }} elevation={0}>
			<CardContent>
				<Stack direction='row' sx={{ flexWrap: 'wrap', gap: 1, px: 2, mb: 1, alignItems: 'center' }}>
					<Typography variant='h6' sx={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: 1 }}>
						🔐 Права доступа
					</Typography>

					<Box
						onClick={() => toggleAllPermissionsHandler(true)}
						sx={{
							display: 'flex',
							alignItems: 'center',
							px: 2,
							py: 1,
							ml: 'auto',
							cursor: 'pointer',
							borderRadius: '8px',
							border: '1px solid #adffca',
							transition: 'all 0.3s ease-in-out',
							':hover': { borderColor: '#4ac979', color: '#096029' },
						}}
					>
						<PermissionBadge
							allowed
							item={{
								permissionId: '',
								object: '*',
								action: '*',
								isAssigned: true,
								isInherited: false,
							}}
							rowName='all'
							sx={{ width: 24, height: 24, borderRadius: '6px' }}
						/>
						<Typography sx={{ ml: 1 }}>Выбрать все</Typography>
					</Box>
					<Box
						onClick={() => toggleAllPermissionsHandler(false)}
						sx={{
							display: 'flex',
							alignItems: 'center',
							px: 2,
							py: 1,
							cursor: 'pointer',
							borderRadius: '8px',
							border: '1px solid #ffd1d9',
							transition: 'all 0.3s ease-in-out',
							':hover': { borderColor: '#9c5353', color: '#760a0a' },
						}}
					>
						<PermissionBadge
							allowed
							item={{
								permissionId: '',
								object: '*',
								action: '*',
								isAssigned: false,
								isInherited: false,
							}}
							rowName='all'
							sx={{ width: 24, height: 24, borderRadius: '6px' }}
						/>
						<Typography sx={{ ml: 1 }}>Сбросить все</Typography>
					</Box>
				</Stack>

				<TableContainer component={Paper} elevation={0} sx={{ borderRadius: '10px', overflow: 'hidden' }}>
					<Table>
						<TableHead>
							<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
								<TableCell sx={{ py: 2.5, px: 2, color: 'text.secondary', fontSize: '0.875rem' }}>
									Ресурс / Действие
								</TableCell>
								{actions.map(a => (
									<TableCell
										key={a.title}
										align='center'
										sx={{
											py: 2.5,
											px: 4,
											color: 'text.secondary',
											fontSize: '0.875rem',
											width: 290,
										}}
									>
										{a.title}
									</TableCell>
								))}
								<TableCell
									align='center'
									sx={{
										py: 2.5,
										px: 4,
										color: 'text.secondary',
										fontSize: '0.875rem',
										width: 290,
									}}
								>
									Все
								</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{data?.data.map(row => {
								let changed = false
								let allAllowed = true
								actions.forEach(a => {
									const item = selectedMap.get(row.group)?.get(a.slug)
									allAllowed = (allAllowed && (item?.isAssigned || item?.isInherited)) ?? true
									changed = changed || (!!item?.status && item?.status !== 'original')
								})

								return (
									<TableRow key={row.group} sx={{ '&:hover': { bgcolor: '#fafafa' } }}>
										<TableCell sx={{ py: '18px' }}>
											<Typography sx={{ fontWeight: 600, fontSize: '14px', color: '#2c3e50' }}>
												{row.title}
											</Typography>
											<Typography sx={{ fontSize: '12px', color: '#9aa1a9', mt: '4px' }}>
												{row.group}
											</Typography>
										</TableCell>

										{actions.map(a => {
											return (
												<TableCell key={a.slug} align='center'>
													<PermissionBadge
														allowed={row.items.some(i => i.action === a.slug) || false}
														item={selectedMap.get(row.group)?.get(a.slug)}
														rowName={row.group}
														onClick={togglePermissionHandler}
													/>
												</TableCell>
											)
										})}

										<TableCell align='center'>
											<PermissionBadge
												allowed
												rowName={row.group}
												item={{
													permissionId: '',
													object: '',
													action: '*',
													isAssigned: allAllowed,
													isInherited: false,
													status: changed ? 'changed' : 'original',
												}}
												onClick={groupToggleHandler}
											/>
										</TableCell>
									</TableRow>
								)
							})}
						</TableBody>
					</Table>
				</TableContainer>
			</CardContent>
		</Card>
	)
}

type BadgeProps = {
	allowed: boolean
	item?: IRolePermissionItem
	rowName: string
	onClick?: (row: string, perm: IRolePermissionItem) => void
	sx?: SxProps
}

const PermissionBadge: FC<BadgeProps> = ({ allowed, item, rowName, onClick, sx }) => {
	const data = getBadge(allowed, item)

	const selectHandler = () => {
		if (item?.isInherited || !allowed) return
		if (onClick && item) onClick(rowName, item)
	}

	return (
		<Tooltip title={data.label} enterDelay={300}>
			<Box
				onClick={selectHandler}
				sx={{
					display: 'inline-flex',
					alignItems: 'center',
					justifyContent: 'center',
					width: 34,
					height: 34,
					borderRadius: '10px',
					fontSize: '18px',
					fontWeight: 'bold',
					bgcolor: data.bg,
					color: data.color,
					cursor: 'pointer',
					border: '1px solid ' + data.bg,
					transition: 'all 0.3s ease-in-out',
					':hover': {
						borderColor: data.color,
					},
					...sx,
				}}
			>
				{data.text}
			</Box>
		</Tooltip>
	)
}
