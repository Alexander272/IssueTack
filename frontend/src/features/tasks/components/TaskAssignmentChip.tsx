import { Box, Typography } from '@mui/material'

import type { IUserShort } from '@/features/user/types/user'
import { Avatar } from '@/components/Avatar'
import { getAvatarColor, getInitials } from '@/utils/avatar'

interface Props {
	assignee: IUserShort | null
}

export const TaskAssignmentChip = ({ assignee }: Props) => {
	if (assignee) {
		return (
			<Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75, minWidth: 0, maxWidth: '100%' }}>
				<Avatar size={24} bgcolor={getAvatarColor(assignee.id)}>
					{getInitials(assignee)}
				</Avatar>
				<Typography
					component='span'
					sx={{
						fontSize: '0.75rem',
						fontWeight: 500,
						lineHeight: 1,
						color: '#374151',
						minWidth: 0,
						overflow: 'hidden',
						textOverflow: 'ellipsis',
						whiteSpace: 'nowrap',
					}}
				>
					{assignee.lastName} {assignee.firstName}
				</Typography>
			</Box>
		)
	}

	return <Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>—</Typography>
}