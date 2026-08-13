import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { Star, MoreVertical, ArrowLeftIcon } from 'lucide-mui'
import { useNavigate } from 'react-router'

import type { ITask } from '../../types/task'

interface Props {
	task: ITask
}

export const Header = ({ task }: Props) => {
	const navigate = useNavigate()

	return (
		<Box
			sx={{
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				pb: 1,
				borderBottom: '1px solid #e5e7eb',
			}}
		>
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1, minWidth: 0 }}>
				<Tooltip title='К списку заявок'>
					<IconButton
						onClick={() => navigate(-1)}
						sx={{ color: '#6b7280', '&:hover': { svg: { color: 'primary.main' } } }}
					>
						<ArrowLeftIcon sx={{ fontSize: 20 }} />
					</IconButton>
				</Tooltip>
				<Box sx={{ width: '1px', height: 24, bgcolor: '#e5e7eb' }} />
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
					{task.ticketNumber && (
						<Typography
							sx={{
								fontSize: '0.9375rem',
								fontFamily: 'monospace',
								color: '#9ca3af',
								fontWeight: 500,
								flexShrink: 0,
							}}
						>
							{task.ticketNumber}.
						</Typography>
					)}
					<Typography
						sx={{
							fontSize: '0.9375rem',
							fontWeight: 600,
							color: '#1f2937',
							overflow: 'hidden',
							textOverflow: 'ellipsis',
							whiteSpace: 'nowrap',
						}}
					>
						{task.title}
					</Typography>
				</Box>
			</Box>

			<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, flexShrink: 0 }}>
				<Tooltip title='В избранное'>
					<IconButton sx={{ color: '#9ca3af', '&:hover': { color: '#f59e0b' }, fontSize: 20 }}>
						<Star sx={{ fontSize: 20 }} />
					</IconButton>
				</Tooltip>
				<Tooltip title='Ещё'>
					<IconButton sx={{ color: '#9ca3af', '&:hover': { color: 'error.main' }, fontSize: 20 }}>
						<MoreVertical sx={{ fontSize: 20 }} />
					</IconButton>
				</Tooltip>
			</Box>
		</Box>
	)
}
