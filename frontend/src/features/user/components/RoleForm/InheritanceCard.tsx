import { useState, useMemo, type FC } from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
import { Box, TextField, Typography, IconButton, InputAdornment, ClickAwayListener, Paper } from '@mui/material'
import { SearchIcon } from '@/components/Icons/SearchIcon'

import type { IRoleWithStats } from '../../types/role'
import type { IForm } from './UpdateRole'
import { useGetRolesWithStatsQuery } from '../../roleApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { CheckIcon } from '@/components/Icons/CheckSimpleIcon'
import { LockIcon } from '@/components/Icons/LockIcon'
import { LeftArrowIcon } from '@/components/Icons/LeftArrowIcon'
import { HyperlinkIcon } from '@/components/Icons/HyperlinkIcon'

type Props = {
	roleId?: string
}

export const InheritanceCard: FC<Props> = ({ roleId }) => {
	const { control, setValue } = useFormContext<IForm>()

	const { data: rolesData } = useGetRolesWithStatsQuery()
	const selectedSlugs = useWatch({ control, name: 'inherits' })
	const currentPerms = useWatch({ control, name: 'perms' })
	const [searchTerm, setSearchTerm] = useState('')
	const [isOpen, setIsOpen] = useState(false)

	const selectedRoles = useMemo(() => {
		if (!rolesData?.data) return []
		return selectedSlugs
			.map((slug: string) => rolesData.data.find(r => r.slug === slug))
			.filter(Boolean) as IRoleWithStats[]
	}, [selectedSlugs, rolesData])

	const ancestorsMap = useMemo(() => {
		const map = new Map<string, Set<string>>()

		const getAncestors = (slug: string): Set<string> => {
			if (map.has(slug)) return map.get(slug)!

			const res = new Set<string>()
			const role = rolesData?.data.find(r => r.slug === slug)

			role?.children.forEach(parentSlug => {
				res.add(parentSlug)
				getAncestors(parentSlug).forEach(s => res.add(s))
			})

			map.set(slug, res)
			return res
		}

		rolesData?.data.forEach(r => getAncestors(r.slug))
		return map
	}, [rolesData])

	const canSelectRole = (role: IRoleWithStats) => {
		for (const selected of selectedRoles) {
			// Проверка: не является ли текущая роль предком уже выбранной?
			if (ancestorsMap.get(selected.slug)?.has(role.slug)) {
				return { can: false, reason: `Роль "${role.name}" уже наследуется через "${selected.name}"` }
			}

			// Проверка: не является ли выбранная роль предком текущей? (нужно удалить старую)
			if (ancestorsMap.get(role.slug)?.has(selected.slug)) {
				return { can: true, willRemove: selected }
			}
		}
		return { can: true }
	}

	const toggleRole = (role: IRoleWithStats) => {
		const check = canSelectRole(role)
		if (!check.can) return

		let newSlugs = [...selectedSlugs]
		if (check.willRemove) {
			newSlugs = newSlugs.filter((s: string) => s !== check.willRemove!.slug)
		}

		const index = newSlugs.indexOf(role.slug)
		if (index === -1) {
			newSlugs.push(role.slug)
		} else {
			newSlugs.splice(index, 1)
		}

		foundInherits(newSlugs)
		setValue('inherits', newSlugs)
	}

	const foundInherits = (slugs: string[]) => {
		if (!rolesData?.data || !currentPerms) return

		const allRelatedSlugs = new Set<string>()

		slugs.forEach(slug => {
			allRelatedSlugs.add(slug)
			ancestorsMap.get(slug)?.forEach(ancestor => allRelatedSlugs.add(ancestor))
		})

		const inheritedPermIds = new Set<string>()
		rolesData.data.forEach(role => {
			if (allRelatedSlugs.has(role.slug)) {
				role.perms.own.items?.forEach(id => inheritedPermIds.add(id))
			}
		})

		const mergedPerms = currentPerms.map(group => ({
			...group,
			resources: group.resources.map(resource => {
				const existsInCurrentSelection = inheritedPermIds.has(resource.permissionId)

				return {
					...resource,
					isInherited: existsInCurrentSelection,
				}
			}),
		}))

		setValue('perms', mergedPerms)
	}

	const removeRole = (slug: string) => {
		const newSlugs = selectedSlugs.filter((s: string) => s !== slug)
		foundInherits(newSlugs)
		setValue('inherits', newSlugs)
	}

	const filteredRoles = useMemo(() => {
		if (!rolesData?.data) return []
		const allRoles = rolesData.data
		if (!searchTerm) return allRoles.filter(r => r.isEditable && r.id !== roleId)
		const term = searchTerm.toLowerCase()
		return allRoles.filter(
			r =>
				r.isEditable &&
				r.id !== roleId &&
				(r.name.toLowerCase().includes(term) || r.slug.toLowerCase().includes(term)),
		)
	}, [rolesData, searchTerm, roleId])

	return (
		<Box
			sx={{
				mb: 1.5,
				borderRadius: '16px',
				border: '1px solid #f3f4f6',
				p: 2,
				bgcolor: 'background.paper',
			}}
		>
			<Typography variant='h6' sx={{ mb: 1, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 1 }}>
				<HyperlinkIcon fontSize={16} /> Наследование от других ролей
			</Typography>
			<Typography variant='body2' color='text.secondary' sx={{ mb: 2 }}>
				Выберите родительские роли.
			</Typography>

			<Box sx={{ position: 'relative' }}>
				<Box
					onClick={() => setIsOpen(!isOpen)}
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						px: 2.5,
						py: 1.8,
						border: '1px solid',
						borderColor: isOpen ? '#6366f1' : '#d1d5db',
						borderRadius: '12px',
						cursor: 'pointer',
						transition: 'all 0.2s ease',
						'&:hover': { borderColor: '#6366f1' },
					}}
				>
					<Typography sx={{ color: selectedSlugs.length ? '#1f2937' : '#9ca3af', fontSize: '0.875rem' }}>
						{selectedSlugs.length
							? `${selectedSlugs.length} рол${selectedSlugs.length > 1 ? 'и' : 'ь'} выбран${selectedSlugs.length == 1 ? 'а' : 'ы'}`
							: 'Выберите роли для наследования...'}
					</Typography>
					<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
						<LeftArrowIcon
							sx={{
								fontSize: 16,
								fill: '#9ca3af',
								transition: 'transform 0.2s ease',
								transform: isOpen ? 'rotate(90deg)' : 'rotate(270deg)',
							}}
						/>
					</Box>
				</Box>

				{isOpen && (
					<ClickAwayListener onClickAway={() => setIsOpen(false)}>
						<Paper
							elevation={8}
							sx={{
								position: 'absolute',
								mt: 1,
								left: 0,
								right: 0,
								borderRadius: '12px',
								overflow: 'hidden',
								zIndex: 50,
							}}
						>
							<Box sx={{ p: 2, borderBottom: '1px solid #f3f4f6' }}>
								<TextField
									fullWidth
									size='small'
									placeholder='Поиск роли...'
									value={searchTerm}
									onChange={e => setSearchTerm(e.target.value)}
									autoFocus
									slotProps={{
										input: {
											startAdornment: (
												<InputAdornment position='start'>
													<SearchIcon sx={{ fontSize: 16, fill: '#9ca3af' }} />
												</InputAdornment>
											),
										},
									}}
									sx={{ '& .MuiOutlinedInput-root': { borderRadius: '12px' } }}
								/>
							</Box>

							<Box sx={{ maxHeight: 280, overflow: 'auto', py: 1 }}>
								{filteredRoles.map(role => {
									const isSelected = selectedSlugs.includes(role.slug)
									const check = canSelectRole(role)
									const chainText = role.children.length
										? `Наследуется от: ${role.children.map(s => rolesData?.data.find(r => r.slug === s)?.name || s).join(' → ')}`
										: ''

									return (
										<Box
											key={role.slug}
											onClick={() => {
												if (check.can) toggleRole(role)
											}}
											sx={{
												display: 'flex',
												alignItems: 'center',
												justifyContent: 'space-between',
												mx: 2,
												px: 2.5,
												py: 1.5,
												borderRadius: '12px',
												cursor: check.can ? 'pointer' : 'not-allowed',
												bgcolor: isSelected ? '#f0f9ff' : 'transparent',
												opacity: check.can ? 1 : 0.5,
												transition: 'all 0.2s ease',
												'&:hover': {
													bgcolor: check.can
														? isSelected
															? '#e0f2fe'
															: '#f8fafc'
														: 'transparent',
												},
											}}
										>
											<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
												<Box
													sx={{
														width: 36,
														height: 36,
														borderRadius: '12px',
														bgcolor: '#e0e7ff',
														color: '#4f46e5',
														display: 'flex',
														alignItems: 'center',
														justifyContent: 'center',
														fontWeight: 600,
														fontSize: '1rem',
													}}
												>
													{role.name[0]}
												</Box>
												<Box>
													<Typography
														sx={{
															fontWeight: 500,
															fontSize: '0.875rem',
															color: '#1f2937',
														}}
													>
														{role.name}
													</Typography>
													<Typography
														sx={{
															fontSize: '0.75rem',
															color: '#9ca3af',
														}}
													>
														{role.slug} • lvl {role.level}
													</Typography>
													{chainText && (
														<Typography
															sx={{ fontSize: '0.7rem', color: '#64748b', mt: 0.25 }}
														>
															{chainText}
														</Typography>
													)}
												</Box>
											</Box>

											<Box>
												{(isSelected || !check.can) && (
													<Box
														sx={{
															width: 24,
															height: 24,
															borderRadius: '50%',
															display: 'flex',
															alignItems: 'center',
															justifyContent: 'center',
															bgcolor: isSelected ? '#22c55e' : '#71b8ff',
														}}
													>
														{isSelected ? (
															<CheckIcon sx={{ fontSize: 14, fill: '#fff' }} />
														) : (
															<LockIcon sx={{ fontSize: 14, fill: '#fff' }} />
														)}
													</Box>
												)}
											</Box>
										</Box>
									)
								})}
							</Box>
						</Paper>
					</ClickAwayListener>
				)}
			</Box>

			<Box sx={{ mt: 3 }}>
				<Typography
					variant='body2'
					sx={{
						mb: 1.5,
						fontWeight: 500,
						color: '#374151',
						display: 'flex',
						alignItems: 'center',
						gap: 1,
					}}
				>
					<HyperlinkIcon fontSize={16} mr={1} />
					Выбранные родительские роли
				</Typography>

				{selectedRoles.length === 0 ? (
					<Box
						sx={{
							textAlign: 'center',
							py: 4,
							color: '#9ca3af',
							border: '1px dashed #d1d5db',
							borderRadius: '16px',
						}}
					>
						<Box sx={{ width: 32, height: 32, mx: 'auto', mb: 1 }}>
							<HyperlinkIcon fontSize={22} fill={'#9ca3af'} />
						</Box>
						<Typography sx={{ fontSize: '0.875rem' }}>Родительские роли не выбраны</Typography>
					</Box>
				) : (
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
						{selectedRoles.map(role => (
							<Box
								key={role.slug}
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'space-between',
									px: 2.5,
									py: 1.5,
									borderRadius: '12px',
									border: '1px solid #e2e8f0',
									bgcolor: '#f8fafc',
								}}
							>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
									<Box>
										<HyperlinkIcon fontSize={16} fill={'#4f46e5'} />
									</Box>
									<Box>
										<Typography sx={{ fontWeight: 500, fontSize: '0.875rem' }}>
											{role.name}
										</Typography>
										<Typography sx={{ fontSize: '0.75rem', color: '#64748b' }}>
											{role.slug} • уровень {role.level}
										</Typography>
									</Box>
								</Box>
								<IconButton
									onClick={() => removeRole(role.slug)}
									sx={{
										width: 32,
										height: 32,
										borderRadius: '8px',
										'&:hover': { bgcolor: '#fef2f2' },
									}}
								>
									<TimesIcon fontSize={14} fill={'#ef4444'} />
								</IconButton>
							</Box>
						))}
					</Box>
				)}
			</Box>
		</Box>
	)
}
