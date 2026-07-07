import { Box, Typography } from '@mui/material'

import type { IUserShort } from '@/features/user/types/user'
// import type { IGroupShort } from '../types/task'
// import { UsersIcon } from 'lucide-mui'
import { UserIcon } from 'lucide-mui'

interface Props {
	assignee: IUserShort | null
	// group: IGroupShort | null
}

export const TaskAssignmentChip = ({ assignee }: Props) => {
	if (assignee) {
		return (
			<Box
				sx={{
					display: 'inline-flex',
					alignItems: 'center',
					gap: 0.75,
					px: 1.25,
					py: 0.5,
					borderRadius: '6px',
					fontSize: '0.75rem',
					fontWeight: 500,
					bgcolor: '#f3f4f6',
					color: '#374151',
					border: '1px solid #e5e7eb',
				}}
			>
				<Typography component='span' sx={{ fontSize: '0.75rem', lineHeight: 1 }}>
					<UserIcon sx={{ fontSize: 18, mr: 0.5 }} />
				</Typography>
				{/* Fontawesome User https://fontawesome.com/icons/classic/solid/user */}
				<Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
					{assignee.lastName} {assignee.firstName}
				</Typography>
			</Box>
		)
	}

	// if (group) {
	// 	return (
	// 		<Box
	// 			sx={{
	// 				display: 'inline-flex',
	// 				alignItems: 'center',
	// 				gap: 0.75,
	// 				px: 1.25,
	// 				py: 0.5,
	// 				borderRadius: '6px',
	// 				fontSize: '0.75rem',
	// 				fontWeight: 500,
	// 				bgcolor: '#eff6ff',
	// 				color: '#1d4ed8',
	// 				border: '1px solid #bfdbfe',
	// 			}}
	// 		>
	// 			<GroupsIcon sx={{ fontSize: '0.75rem', fill: '#1d4ed8' }} />
	// 			<Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
	// 				{group.name}
	// 			</Typography>
	// 		</Box>
	// 	)
	// }

	return <Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>—</Typography>
}
