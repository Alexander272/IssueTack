import { useState } from 'react'
import { Box, IconButton, Tooltip, Typography, Menu, MenuItem, ListItemIcon } from '@mui/material'
import { Star, MoreVertical, ArrowLeftIcon, Pin, Pencil, UserRound } from 'lucide-mui'
import { useLocation, useNavigate } from 'react-router'

import type { ITask, TicketStatus } from '../../types/task'
import { AppRoutes } from '@/pages/router/routes'
import {
	useGetFavoriteStateQuery,
	useAddFavoriteMutation,
	useRemoveFavoriteMutation,
} from '@/features/favorites/favoritesApiSlice'
import { useAppSelector } from '@/hooks/redux'
import { getUserId, getHasNoGroups } from '@/features/user/userSlice'

const INACTIVE_STATUSES: TicketStatus[] = ['resolved', 'closed', 'cancelled']

interface Props {
	task: ITask
	onEdit?: () => void
	onTransfer?: () => void
}

export const Header = ({ task, onEdit, onTransfer }: Props) => {
	const navigate = useNavigate()
	const location = useLocation()

	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
	const currentUserId = useAppSelector(getUserId)
	const hasNoGroups = useAppSelector(getHasNoGroups)

	const { data: favState } = useGetFavoriteStateQuery(task.id)
	const [addFavorite] = useAddFavoriteMutation()
	const [removeFavorite] = useRemoveFavoriteMutation()

	const isInactive = INACTIVE_STATUSES.includes(task.status)
	const pinned = favState?.data?.temporary ?? false
	const starred = favState?.data?.permanent ?? false
	const isAssignee = currentUserId != null && currentUserId === task.assignee?.id
	const canTransfer = !isInactive && isAssignee

	const handleMenuClose = () => setAnchorEl(null)

	const handleBack = () => {
		if (location.key !== 'default') {
			navigate(-1)
		} else {
			if (!hasNoGroups) navigate(AppRoutes.Home, { replace: true })
			else navigate(AppRoutes.Tasks, { replace: true })
		}
	}

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

	const hasAnyAction = Boolean(onEdit) || (Boolean(onTransfer) && canTransfer) || !hasNoGroups

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
						onClick={handleBack}
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
				{!isInactive && !hasNoGroups && (
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
				{hasAnyAction && (
					<>
						<Tooltip title='Ещё'>
							<IconButton
								onClick={e => setAnchorEl(e.currentTarget)}
								sx={{ color: '#9ca3af', '&:hover': { color: 'error.main' }, fontSize: 20 }}
							>
								<MoreVertical sx={{ fontSize: 20 }} />
							</IconButton>
						</Tooltip>
						<Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
							{onEdit && (
								<MenuItem
									onClick={() => {
										handleMenuClose()
										onEdit()
									}}
								>
									<ListItemIcon>
										<Pencil sx={{ fontSize: 18, color: '#9ca3af' }} />
									</ListItemIcon>
									Редактировать
								</MenuItem>
							)}
							{onTransfer && canTransfer && (
								<MenuItem
									onClick={() => {
										handleMenuClose()
										onTransfer()
									}}
								>
									<ListItemIcon>
										<UserRound sx={{ fontSize: 18, color: '#9ca3af' }} />
									</ListItemIcon>
									Передать
								</MenuItem>
							)}
							{!hasNoGroups && (
								<MenuItem onClick={toggleStar}>
									<ListItemIcon>
										<Star sx={{ fontSize: 18, color: starred ? '#f59e0b' : '#9ca3af' }} />
									</ListItemIcon>
									{starred ? 'Убрать из избранного' : 'В избранное'}
								</MenuItem>
							)}
						</Menu>
					</>
				)}
			</Box>
		</Box>
	)
}
