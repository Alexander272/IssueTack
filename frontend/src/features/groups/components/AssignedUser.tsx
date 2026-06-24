import { type FC } from 'react'
import { Box, Typography } from '@mui/material'

import type { IUserShort } from '@/features/user/types/user'
import { UserIcon } from '@/components/Icons/UserIcon'

type Props = {
	user: IUserShort | null | undefined
	fallback?: string
}

export const AssignedUser: FC<Props> = ({ user, fallback = 'Не назначен' }) => {
	if (!user) {
		return <Typography variant='body2' sx={{ color: '#9ca3af' }}>{fallback}</Typography>
	}

	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
			<Box
				sx={{
					width: 36,
					height: 36,
					borderRadius: '50%',
					bgcolor: '#e5e7eb',
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
				}}
			>
				<UserIcon sx={{ fontSize: 18, color: '#6b7280' }} />
			</Box>
			<Box>
				<Typography variant='body2' sx={{ fontWeight: 600 }}>
					{user.lastName} {user.firstName}
				</Typography>
				<Typography variant='caption' sx={{ color: '#9ca3af' }}>
					{user.email}
				</Typography>
			</Box>
		</Box>
	)
}
