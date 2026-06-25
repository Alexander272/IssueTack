import { type FC } from 'react'
import { Box, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import dayjs from 'dayjs'
import 'dayjs/locale/ru'

import type { ICategory } from '../types/category'
import { TaskPriorityBadge } from '@/features/tasks/components/TaskPriorityBadge'
import { StatusBadge } from '@/features/access/components/StatusBadge'
import { EditIcon } from '@/components/Icons/EditIcon'
import { EyeIcon } from '@/components/Icons/EyeIcon'
import { GroupsIcon } from '@/components/Icons/GroupsIcon'

const statusLabel = (active: boolean) => (active ? 'Активна' : 'Неактивна')

const getInitials = (name: string) => {
	const words = name.trim().split(/\s+/)
	if (words.length >= 2) {
		return (words[0][0] + words[1][0]).toUpperCase()
	}
	return name.slice(0, 2).toUpperCase()
}

type Props = {
	categories: ICategory[]
	groupsMap: Map<string, string>
	onView: (cat: ICategory) => void
	onEdit: (cat: ICategory) => void
}

export const CategoryCardList: FC<Props> = ({ categories, groupsMap, onView, onEdit }) => {
	if (!categories.length) {
		return (
			<Typography align='center' sx={{ py: 3, color: 'text.secondary' }}>
				Категории не найдены.
			</Typography>
		)
	}

	return (
		<Stack direction='row' sx={{ gap: 2, flexWrap: 'wrap', justifyContent: 'center', alignItems: 'center' }}>
			{categories.map(cat => {
				const groupName = groupsMap.get(cat.groupId) || 'Неизвестная группа'
				return (
					<Box
						key={cat.id}
						sx={{
							bgcolor: '#fff',
							borderRadius: '12px',
							border: '1px solid #e5e7eb',
							boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)',
							overflow: 'hidden',
							opacity: cat.isActive ? 1 : 0.6,
							flexGrow: 1,
							minWidth: 300,
							maxWidth: 400,
						}}
					>
						<Box sx={{ p: 2 }}>
							<Box sx={{ display: 'flex', gap: 1.5, mb: 1.5 }}>
								<Box
									sx={{
										width: 48,
										height: 48,
										borderRadius: '10px',
										bgcolor: '#f3e8ff',
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'center',
										flexShrink: 0,
									}}
								>
									<Typography sx={{ fontWeight: 700, fontSize: 18, color: '#9333ea' }}>
										{getInitials(cat.name)}
									</Typography>
								</Box>
								<Box sx={{ flex: 1, minWidth: 0 }}>
									<Typography
										sx={{
											fontWeight: 700,
											color: '#111827',
											overflow: 'hidden',
											textOverflow: 'ellipsis',
											whiteSpace: 'nowrap',
										}}
									>
										{cat.name}
									</Typography>
									<Typography
										sx={{
											fontSize: '0.875rem',
											color: '#6b7280',
											mt: 0.5,
											overflow: 'hidden',
											textOverflow: 'ellipsis',
											display: '-webkit-box',
											WebkitLineClamp: 2,
											WebkitBoxOrient: 'vertical',
										}}
									>
										{cat.description}
									</Typography>
								</Box>
							</Box>

							<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mb: 1.5 }}>
								<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
									<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
										Группа-владелец
									</Typography>
									<Box
										sx={{
											display: 'inline-flex',
											alignItems: 'center',
											gap: 0.75,
											px: 1.5,
											py: 0.5,
											borderRadius: '6px',
											fontSize: '0.75rem',
											fontWeight: 500,
											bgcolor: '#eff6ff',
											color: '#1d4ed8',
											border: '1px solid #bfdbfe',
										}}
									>
										<GroupsIcon sx={{ fontSize: 16, mr: 0.5, fill: '#1d4ed8' }} />
										{groupName}
									</Box>
								</Box>
								<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
									<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>Приоритет</Typography>
									<TaskPriorityBadge priority={cat.priority} />
								</Box>
								<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
									<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>Статус</Typography>
									<StatusBadge active={cat.isActive} label={statusLabel(cat.isActive)} />
								</Box>
							</Box>

							<Box
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'space-between',
									pt: 1.5,
									borderTop: '1px solid #f3f4f6',
								}}
							>
								<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
									{dayjs(cat.updatedAt).locale('ru').format('DD MMMM YYYY')}
								</Typography>
								<Box sx={{ display: 'flex', gap: 0.5 }}>
									<Tooltip title='Просмотр'>
										<IconButton
											onClick={() => onView(cat)}
											size='large'
											sx={{
												fill: '#9ca3af',
												'&:hover': { fill: '#f59e0b', bgcolor: '#f3f4f6' },
											}}
										>
											<EyeIcon sx={{ fontSize: 16 }} />
										</IconButton>
									</Tooltip>
									<Tooltip title='Редактировать'>
										<IconButton
											onClick={() => onEdit(cat)}
											size='large'
											sx={{
												fill: '#9ca3af',
												'&:hover': { fill: '#3b82f6', bgcolor: '#f3f4f6' },
											}}
										>
											<EditIcon sx={{ fontSize: 16 }} />
										</IconButton>
									</Tooltip>
								</Box>
							</Box>
						</Box>
					</Box>
				)
			})}
		</Stack>
	)
}
