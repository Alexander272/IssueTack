import { useState } from 'react'
import { Box, IconButton, Tooltip, Typography, Menu, MenuItem, ListItemIcon } from '@mui/material'
import { Star, MoreVertical, ArrowLeftIcon, Pin } from 'lucide-mui'
import { useNavigate } from 'react-router'

import type { ITask, TicketStatus } from '../../types/task'
import {
	useGetFavoriteStateQuery,
	useAddFavoriteMutation,
	useRemoveFavoriteMutation,
} from '@/features/favorites/favoritesApiSlice'

const INACTIVE_STATUSES: TicketStatus[] = ['resolved', 'closed', 'cancelled']

interface Props {
	task: ITask
}

export const Header = ({ task }: Props) => {
	const navigate = useNavigate()
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)

	const { data: favState } = useGetFavoriteStateQuery(task.id)
	const [addFavorite] = useAddFavoriteMutation()
	const [removeFavorite] = useRemoveFavoriteMutation()

	const isInactive = INACTIVE_STATUSES.includes(task.status)
	const pinned = favState?.data?.temporary ?? false
	const starred = favState?.data?.permanent ?? false

	const handleMenuClose = () => setAnchorEl(null)

	const togglePin = () => {
		if (pinned) {
			removeFavorite({ id: task.id, type: 'temporary' })
		} else {
			addFavorite({ id: task.id, type: 'temporary' })
		}
	}

	const toggleStar = () => {
		handleMenuClose()
		if (starred) {
			removeFavorite({ id: task.id, type: 'permanent' })
		} else {
			addFavorite({ id: task.id, type: 'permanent' })
		}
	}

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
				{!isInactive && (
					<Tooltip title={pinned ? 'Открепить' : 'Закрепить'}>
						<IconButton
							onClick={togglePin}
							sx={{
								color: pinned ? '#f59e0b' : '#9ca3af',
								'&:hover': { color: '#f59e0b' },
								fontSize: 20,
							}}
						>
							<Pin sx={{ fontSize: 20 }} />
						</IconButton>
					</Tooltip>
				)}
				<Tooltip title='Ещё'>
					<IconButton
						onClick={e => setAnchorEl(e.currentTarget)}
						sx={{ color: '#9ca3af', '&:hover': { color: 'error.main' }, fontSize: 20 }}
					>
						<MoreVertical sx={{ fontSize: 20 }} />
					</IconButton>
				</Tooltip>
				<Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
					<MenuItem onClick={toggleStar}>
						<ListItemIcon>
							<Star sx={{ fontSize: 18, color: starred ? '#f59e0b' : '#9ca3af' }} />
						</ListItemIcon>
						{starred ? 'Убрать из избранного' : 'В избранное'}
					</MenuItem>
				</Menu>
			</Box>
		</Box>
	)
}
