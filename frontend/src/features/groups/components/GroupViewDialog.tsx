import { type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	Stack,
	Typography,
} from '@mui/material'

import type { IGroup, IGroupDTO } from '../types/group'
import { getSmartDate } from '@/utils/date'
import { GroupsIcon } from '@/components/Icons/GroupsIcon'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { GroupMemberRow } from './GroupMemberRow'
import { AssignedUser } from './AssignedUser'
import { getMemberRoleInfo } from '../utils/memberRole'

type Props = {
	group: IGroup | null
	onClose: () => void
	onEdit: (group: IGroupDTO) => void
}

export const GroupViewDialog: FC<Props> = ({ group, onClose, onEdit }) => {
	const memberCount = group?.members?.length ?? 0

	return (
		<Dialog
			open={Boolean(group)}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{
				paper: {
					sx: { borderRadius: '16px', p: 1 },
				},
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Просмотр группы
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>
			{group && (
				<>
					<DialogContent
						dividers
						sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
					>
						<Stack spacing={3}>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
								<Box
									sx={{
										width: 64,
										height: 64,
										borderRadius: '12px',
										bgcolor: '#dbeafe',
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'center',
									}}
								>
									<GroupsIcon sx={{ fontSize: 32, color: '#3b82f6' }} />
								</Box>
								<Box>
									<Typography variant='h5' sx={{ fontWeight: 'bold' }}>
										{group.name}
									</Typography>
									<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.5 }}>
										{group.description || '—'}
									</Typography>
									<Box sx={{ display: 'flex', gap: 2, mt: 1 }}>
										<Typography variant='caption' sx={{ color: '#9ca3af' }}>
											Создана: {getSmartDate(group.createdAt)}
										</Typography>
										<Typography variant='caption' sx={{ color: '#9ca3af' }}>
											Обновлена: {getSmartDate(group.updatedAt)}
										</Typography>
									</Box>
								</Box>
							</Box>

							<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
								<Box sx={{ bgcolor: '#f9fafb', borderRadius: '8px', p: 2 }}>
									<Typography
										variant='caption'
										sx={{ fontWeight: 600, color: '#9ca3af', textTransform: 'uppercase' }}
									>
										Руководитель группы
									</Typography>
									<Box sx={{ mt: 1.5 }}>
										<AssignedUser user={group.manager} />
									</Box>
								</Box>

								<Box sx={{ bgcolor: '#f9fafb', borderRadius: '8px', p: 2 }}>
									<Typography
										variant='caption'
										sx={{ fontWeight: 600, color: '#9ca3af', textTransform: 'uppercase' }}
									>
										Исполнитель по умолчанию
									</Typography>
									<Box sx={{ mt: 1.5 }}>
										<AssignedUser user={group.defaultAssignee} />
									</Box>
								</Box>
							</Box>

							<Box>
								<Box
									sx={{
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'space-between',
										mb: 1.5,
									}}
								>
									<Typography variant='subtitle1' sx={{ fontWeight: 'bold' }}>
										Состав группы ({memberCount})
									</Typography>
								</Box>
								<Stack spacing={1}>
									{group.members?.map(member => {
										const role = getMemberRoleInfo(member.id, group.managerId, group.defaultAssigneeId)
										return (
											<GroupMemberRow
												key={member.id}
												user={member}
												roleLabel={role?.label}
												roleColor={role?.color}
											/>
										)
									})}
									{!group.members?.length && (
										<Typography
											variant='body2'
											sx={{ color: '#9ca3af', py: 2, textAlign: 'center' }}
										>
											Нет участников
										</Typography>
									)}
								</Stack>
							</Box>

							<Box sx={{ pt: 1 }}>
								<Typography variant='caption' sx={{ color: '#9ca3af' }}>
									<strong>ID группы:</strong>{' '}
									<Typography
										component='code'
										sx={{
											fontSize: '0.75rem',
											bgcolor: '#f3f4f6',
											px: 1,
											py: 0.5,
											borderRadius: '4px',
											color: '#6b7280',
										}}
									>
										{group.id}
									</Typography>
								</Typography>
							</Box>
						</Stack>
					</DialogContent>
					<DialogActions sx={{ p: 2, gap: 1 }}>
						<Button
							onClick={onClose}
							variant='outlined'
							sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
						>
							Закрыть
						</Button>
						<Button
							onClick={() => {
								onEdit({
									id: group.id,
									name: group.name,
									description: group.description,
									managerId: group.managerId ?? null,
									defaultAssigneeId: group.defaultAssigneeId ?? null,
									memberIds: group.members?.map(m => m.id) ?? [],
								})
								onClose()
							}}
							variant='contained'
							sx={{ textTransform: 'none', px: 3 }}
						>
							Редактировать
						</Button>
					</DialogActions>
				</>
			)}
		</Dialog>
	)
}
