import { Box, Typography } from '@mui/material'
import { UserPlus, User, Users, Wrench, Briefcase } from 'lucide-mui'
import type { ReactNode } from 'react'

import { Avatar } from '@/components/Avatar'
import { getAvatarColor, getInitials } from '@/utils/avatar'

import type { ITask } from '../../types/task'

interface Props {
	task: ITask
}

const UserRow = ({ label, icon, name, sub, userId }: { label: string; icon: ReactNode; name: string; sub?: string; userId?: string }) => (
	<Box>
		<Typography
			sx={{ fontSize: '0.75rem', color: '#9ca3af', mb: 0.5, display: 'flex', alignItems: 'center', gap: 0.5 }}
		>
			{icon} {label}
		</Typography>
		{name ? (
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
				<Avatar size={28} bgcolor={getAvatarColor(userId ?? name)}>
					{getInitials(name)}
				</Avatar>
				<Box sx={{ minWidth: 0 }}>
					<Typography
						sx={{
							fontSize: '0.8125rem',
							fontWeight: 500,
							color: '#1f2937',
							overflow: 'hidden',
							textOverflow: 'ellipsis',
							whiteSpace: 'nowrap',
						}}
					>
						{name}
					</Typography>
					{sub && (
						<Typography
							sx={{
								fontSize: '0.6875rem',
								color: '#9ca3af',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{sub}
						</Typography>
					)}
				</Box>
			</Box>
		) : (
			<Typography sx={{ fontSize: '0.8125rem', color: '#d1d5db' }}>—</Typography>
		)}
	</Box>
)

const getUserName = (user: { firstName: string; lastName: string } | null | undefined) => {
	if (!user) return ''
	return `${user.lastName} ${user.firstName}`
}

export const Participants = ({ task }: Props) => {
	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ px: 2.5, py: 1.5, borderBottom: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
				<Typography
					sx={{
						fontSize: '0.75rem',
						fontWeight: 700,
						color: '#374151',
						textTransform: 'uppercase',
						letterSpacing: '0.05em',
						display: 'flex',
						alignItems: 'center',
						gap: 0.75,
					}}
				>
					<Users sx={{ fontSize: 14 }} />
					Участники
				</Typography>
			</Box>
			<Box sx={{ p: 2.5, display: 'flex', flexDirection: 'column', gap: 2 }}>
				{task.owner?.id !== task.creator.id ? (
					<UserRow
						label='Создатель'
						icon={<UserPlus sx={{ fontSize: 14 }} />}
						name={getUserName(task.creator)}
						sub={task.site?.name}
						userId={task.creator.id}
					/>
				) : null}
				{task.owner && (
					<UserRow
						label='Заявитель'
						icon={<User sx={{ fontSize: 14 }} />}
						name={getUserName(task.owner)}
						sub={task.site?.name}
						userId={task.owner.id}
					/>
				)}
				{task.group && (
					<Box>
						<Typography
							sx={{
								fontSize: '0.75rem',
								color: '#9ca3af',
								mb: 0.5,
								display: 'flex',
								alignItems: 'center',
								gap: 0.5,
							}}
						>
							<Users sx={{ fontSize: 14 }} /> Группа
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
								bgcolor: '#dbeafe',
								color: '#1e40af',
							}}
						>
							{task.group.name}
						</Box>
					</Box>
				)}
				{task.assignee && (
					<UserRow
						label='Исполнитель'
						icon={<Wrench sx={{ fontSize: 14 }} />}
						name={getUserName(task.assignee)}
						sub={task.group?.name}
						userId={task.assignee.id}
					/>
				)}
				{task.manager && (
					<UserRow
						label='Руководитель'
						icon={<Briefcase sx={{ fontSize: 14 }} />}
						name={getUserName(task.manager)}
						userId={task.manager.id}
					/>
				)}
			</Box>
		</Box>
	)
}
