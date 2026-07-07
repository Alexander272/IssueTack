import { type FC } from 'react'
import { Box, Typography } from '@mui/material'

import type { IUserShort } from '@/features/user/types/user'
import { Avatar } from '@/components/Avatar'

type Props = {
	user: IUserShort
	roleLabel?: string
	roleColor?: string
	avatarColor: string
}

export const GroupMemberRow: FC<Props> = ({ user, roleLabel, roleColor, avatarColor }) => (
	<Box
		sx={{
			display: 'flex',
			alignItems: 'center',
			justifyContent: 'space-between',
			p: 1.5,
			bgcolor: '#f9fafb',
			borderRadius: '8px',
			'&:hover': { bgcolor: '#f3f4f6' },
		}}
	>
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
			<Avatar size={36} bgcolor={avatarColor}>
				{user.lastName[0]}{user.firstName[0]}
			</Avatar>
			<Box>
				<Typography variant='body2' sx={{ fontWeight: 600, color: '#1f2937' }}>
					{user.lastName} {user.firstName}
				</Typography>
				<Typography variant='caption' sx={{ color: '#9ca3af' }}>
					{user.email}
				</Typography>
			</Box>
		</Box>
		{roleLabel && (
			<Box
				sx={{
					px: 1.5,
					py: 0.5,
					bgcolor: roleColor ? `${roleColor}15` : '#f3f4f6',
					color: roleColor || '#6b7280',
					fontSize: '0.7rem',
					fontWeight: 600,
					borderRadius: '12px',
				}}
			>
				{roleLabel}
			</Box>
		)}
	</Box>
)
