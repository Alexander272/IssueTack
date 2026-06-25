import { useState, useMemo, type FC } from 'react'
import { Box, ClickAwayListener, IconButton, InputAdornment, Paper, TextField, Typography } from '@mui/material'

import type { IUserData } from '@/features/user/types/user'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { CheckIcon } from '@/components/Icons/CheckSimpleIcon'

type Props = {
	value: string[]
	onChange: (value: string[]) => void
	users: IUserData[]
}

export const MemberPicker: FC<Props> = ({ value, onChange, users }) => {
	const [searchTerm, setSearchTerm] = useState('')
	const [isOpen, setIsOpen] = useState(false)

	const selectedCount = value?.length ?? 0

	const selectedUsers = useMemo(() => {
		const ids = value ?? []
		return ids.map(id => users.find(u => u.id === id)).filter(Boolean) as IUserData[]
	}, [value, users])

	const filteredUsers = useMemo(() => {
		if (!searchTerm) return users
		const term = searchTerm.toLowerCase()
		return users.filter(
			u =>
				u.lastName.toLowerCase().includes(term) ||
				u.firstName.toLowerCase().includes(term) ||
				u.email.toLowerCase().includes(term),
		)
	}, [users, searchTerm])

	const toggleMember = (userId: string) => {
		const current = value ?? []
		if (current.includes(userId)) {
			onChange(current.filter((id: string) => id !== userId))
		} else {
			onChange([...current, userId])
		}
	}

	return (
		<Box sx={{ position: 'relative' }}>
			<Box
				onClick={() => setIsOpen(!isOpen)}
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					px: 2,
					py: 1.5,
					border: '1px solid',
					borderColor: isOpen ? '#6366f1' : '#d1d5db',
					borderRadius: '8px',
					cursor: 'pointer',
					transition: 'all 0.2s ease',
					'&:hover': { borderColor: '#6366f1' },
				}}
			>
				<Typography sx={{ color: selectedCount ? '#1f2937' : '#9ca3af', fontSize: '0.875rem' }}>
					{selectedCount
						? `${selectedCount} участник${getEnding(selectedCount)} выбран${selectedCount === 1 ? '' : 'ы'}`
						: 'Выберите участников...'}
				</Typography>
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
								placeholder='Поиск участника...'
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
							{filteredUsers.length === 0 ? (
								<Box sx={{ py: 4, textAlign: 'center' }}>
									<Typography sx={{ color: '#9ca3af', fontSize: '0.875rem' }}>
										Пользователи не найдены
									</Typography>
								</Box>
							) : (
								filteredUsers.map(user => {
									const isSelected = (value ?? []).includes(user.id)

									return (
										<Box
											key={user.id}
											onClick={() => toggleMember(user.id)}
											sx={{
												display: 'flex',
												alignItems: 'center',
												justifyContent: 'space-between',
												mx: 2,
												px: 2.5,
												py: 1.5,
												borderRadius: '12px',
												cursor: 'pointer',
												bgcolor: isSelected ? '#f0f9ff' : 'transparent',
												transition: 'all 0.2s ease',
												'&:hover': {
													bgcolor: isSelected ? '#e0f2fe' : '#f8fafc',
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
													{user.firstName[0]}
													{user.lastName[0]}
												</Box>
												<Box>
													<Typography
														sx={{ fontWeight: 500, fontSize: '0.875rem', color: '#1f2937' }}
													>
														{user.lastName} {user.firstName}
													</Typography>
													<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
														{user.email}
													</Typography>
												</Box>
											</Box>

											{isSelected && (
												<Box
													sx={{
														width: 24,
														height: 24,
														borderRadius: '50%',
														display: 'flex',
														alignItems: 'center',
														justifyContent: 'center',
														bgcolor: '#22c55e',
													}}
												>
													<CheckIcon sx={{ fontSize: 14, fill: '#fff' }} />
												</Box>
											)}
										</Box>
									)
								})
							)}
						</Box>
					</Paper>
				</ClickAwayListener>
			)}

			{selectedCount > 0 && (
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mt: 2 }}>
					{selectedUsers.map(user => (
						<Box
							key={user.id}
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
									<Typography sx={{ fontWeight: 500, fontSize: '0.875rem' }}>
										{user.lastName} {user.firstName}
									</Typography>
									<Typography sx={{ fontSize: '0.75rem', color: '#64748b' }}>{user.email}</Typography>
								</Box>
							</Box>
							<IconButton
								onClick={() => toggleMember(user.id)}
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
	)
}

function getEnding(n: number): string {
	if (n % 10 === 1 && n % 100 !== 11) return ''
	if (n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 10 || n % 100 >= 20)) return 'а'
	return 'ов'
}
