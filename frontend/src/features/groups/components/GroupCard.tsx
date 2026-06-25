import { type FC } from 'react'
import { Box, Button, Typography } from '@mui/material'

import type { IGroup } from '../types/group'
import { getSmartDate } from '@/utils/date'
import { GroupsIcon } from '@/components/Icons/GroupsIcon'
import { EyeIcon } from '@/components/Icons/EyeIcon'
import { EditIcon } from '@/components/Icons/EditIcon'

const AVATAR_BG = ['#3b82f6', '#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6']

interface Props {
	group: IGroup
	onView: (group: IGroup) => void
	onEdit: (group: IGroup) => void
}

export const GroupCard: FC<Props> = ({ group, onView, onEdit }) => {
	const memberCount = group.members?.length ?? 0

	return (
		<Box
			sx={{
				bgcolor: '#fff',
				borderRadius: '12px',
				border: '1px solid #e5e7eb',
				overflow: 'hidden',
				transition: 'box-shadow 0.2s',
				'&:hover': { boxShadow: '0 4px 12px rgba(0,0,0,0.08)' },
			}}
		>
			<Box sx={{ p: 3 }}>
				<Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 2 }}>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
						<Box
							sx={{
								width: 48,
								height: 48,
								borderRadius: '10px',
								bgcolor: `${AVATAR_BG[memberCount % AVATAR_BG.length]}15`,
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
							}}
						>
							<GroupsIcon
								sx={{
									fontSize: 24,
									color: AVATAR_BG[memberCount % AVATAR_BG.length],
								}}
							/>
						</Box>
						<Box>
							<Typography variant='subtitle1' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
								{group.name}
							</Typography>
							<Typography variant='caption' sx={{ color: '#9ca3af' }}>
								{memberCount}{' '}
								{memberCount === 1 ? 'участник' : memberCount < 5 ? 'участника' : 'участников'}
							</Typography>
						</Box>
					</Box>
					<Box sx={{ display: 'flex', gap: 0.5 }}>
						<Box
							component='button'
							onClick={() => onView(group)}
							sx={{
								color: '#9ca3af',
								'&:hover': { bgcolor: '#f3f4f6', svg: { fill: '#f59e0b' } },
								p: 1,
								borderRadius: '6px',
								border: 'none',
								cursor: 'pointer',
								display: 'flex',
								background: 'none',
							}}
							title='Просмотр'
						>
							<EyeIcon sx={{ fontSize: 20, fill: '#9ca3af' }} />
						</Box>
						<Box
							component='button'
							onClick={() => onEdit(group)}
							sx={{
								color: '#9ca3af',
								'&:hover': { bgcolor: '#f3f4f6', svg: { fill: '#3b82f6' } },
								p: 1,
								borderRadius: '6px',
								border: 'none',
								cursor: 'pointer',
								display: 'flex',
								background: 'none',
							}}
							title='Редактировать'
						>
							<EditIcon sx={{ fontSize: 20, fill: '#9ca3af' }} />
						</Box>
					</Box>
				</Box>

				<Typography
					variant='body2'
					sx={{
						color: '#6b7280',
						mb: 2,
						overflow: 'hidden',
						textOverflow: 'ellipsis',
						display: '-webkit-box',
						WebkitLineClamp: 2,
						WebkitBoxOrient: 'vertical',
					}}
				>
					{group.description || '—'}
				</Typography>

				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
						<Box
							sx={{
								width: 16,
								height: 16,
								borderRadius: '50%',
								bgcolor: group.manager ? '#3b82f6' : '#e5e7eb',
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
								fontSize: 8,
								color: '#fff',
								fontWeight: 'bold',
								flexShrink: 0,
							}}
						>
							{group.manager ? `${group.manager?.lastName[0]}${group.manager?.firstName[0]}` : '—'}
						</Box>
						<Typography variant='caption' sx={{ color: '#6b7280' }}>
							Руководитель:{' '}
							{group.manager ? `${group.manager.lastName} ${group.manager.firstName}` : 'Не назначен'}
						</Typography>
					</Box>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
						<Box
							sx={{
								width: 16,
								height: 16,
								borderRadius: '50%',
								bgcolor: group.defaultAssignee ? '#10b981' : '#e5e7eb',
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'center',
								fontSize: 8,
								color: '#fff',
								fontWeight: 'bold',
								flexShrink: 0,
							}}
						>
							{group.defaultAssignee
								? `${group.defaultAssignee.lastName[0]}${group.defaultAssignee.firstName[0]}`
								: '—'}
						</Box>
						<Typography variant='caption' sx={{ color: '#6b7280' }}>
							По умолчанию:{' '}
							{group.defaultAssignee
								? `${group.defaultAssignee.lastName} ${group.defaultAssignee.firstName}`
								: 'Не назначен'}
						</Typography>
					</Box>
				</Box>
			</Box>

			<Box
				sx={{
					px: 3,
					py: 1.5,
					bgcolor: '#f9fafb',
					borderTop: '1px solid #e5e7eb',
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
				}}
			>
				<Typography variant='caption' sx={{ color: '#9ca3af' }}>
					Обновлена: {getSmartDate(group.updatedAt)}
				</Typography>
				<Button
					variant='text'
					size='small'
					onClick={() => onView(group)}
					sx={{ textTransform: 'none', fontWeight: 500, fontSize: '0.75rem', p: 0, minWidth: 'auto' }}
				>
					Состав группы →
				</Button>
			</Box>
		</Box>
	)
}
