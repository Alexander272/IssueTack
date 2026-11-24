import { useState } from 'react'
import { Box, IconButton, Tooltip, Typography, Menu, MenuItem, ListItemIcon } from '@mui/material'
import { Star, MoreVertical, ArrowLeftIcon, Pin, Pencil, UserRound, ListTree, CopyPlusIcon } from 'lucide-mui'
import { useLocation, useNavigate } from 'react-router'

import type { ITask } from '../../types/task'
import { INACTIVE_STATUSES } from '../../constants/taskMaps'
import { AppRoutes } from '@/pages/router/routes'
import {
	useGetFavoriteStateQuery,
	useAddFavoriteMutation,
	useRemoveFavoriteMutation,
} from '@/features/favorites/favoritesApiSlice'
import { useAppSelector } from '@/hooks/redux'
import { getUserId, getHasNoGroups } from '@/features/user/userSlice'
import { TransferModal } from './TransferModal'
import { LinkDuplicateDialog } from './LinkDuplicateDialog'
import { ModifierPopover } from './ModifierPopover'

// Те же статусы, что пропускает LinkDuplicate (ErrDuplicateStatusNotAllowed).
// Закрытая заявка добавляется ниже отдельно — но только если уже является дублем
// (перенаправление), произвольную closed связывать нельзя.
const LINKABLE_STATUSES = ['open', 'in_progress', 'pending', 'on_hold', 'resolved']

interface Props {
	task: ITask
	onEdit?: () => void
	onSetModifier: (categoryModifierId: string | null) => Promise<void>
}

export const Header = ({ task, onEdit, onSetModifier }: Props) => {
	const navigate = useNavigate()
	const location = useLocation()

	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
	const currentUserId = useAppSelector(getUserId)
	const hasNoGroups = useAppSelector(getHasNoGroups)

	const [transferOpen, setTransferOpen] = useState(false)
	const [duplicateOpen, setDuplicateOpen] = useState(false)

	const [modifierAnchor, setModifierAnchor] = useState<HTMLElement | null>(null)

	const { data: favState } = useGetFavoriteStateQuery(task.id)
	const [addFavorite] = useAddFavoriteMutation()
	const [removeFavorite] = useRemoveFavoriteMutation()

	const isInactive = INACTIVE_STATUSES.includes(task.status)
	const pinned = favState?.data?.temporary ?? false
	const starred = favState?.data?.permanent ?? false
	const isAssignee = currentUserId != null && currentUserId === task.assignee?.id
	const canTransfer = !isInactive && isAssignee
	const canLink =
		Boolean(task.access?.canLinkDuplicate) &&
		(LINKABLE_STATUSES.includes(task.status) || (task.status === 'closed' && Boolean(task.duplicateOf)))

	const handleMenuClose = () => setAnchorEl(null)

	const handleBack = () => {
		// 1. Проверяем, что это не первая страница в текущей вкладке
		// (window.history.state.idx > 0 означает, что сзади точно есть страница нашего приложения)
		const canGoBack = window.history.state && window.history.state.idx > 0

		// 2. Дополнительно страхуемся: если мы пришли с /auth, назад идти точно не нужно
		const cameFromAuth = location.state?.from?.pathname === AppRoutes.Auth

		if (canGoBack && !cameFromAuth) {
			navigate(-1)
		} else {
			if (hasNoGroups) navigate(AppRoutes.Home, { replace: true })
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

	// Подкатегорию меняют из меню «Ещё», только когда она уже выбрана: пока её нет,
	// в InfoBar видна отдельная кнопка-триггер. Право берём из готового флага бэкенда.
	const canEditModifier = Boolean(task.access?.canPickModifier && task.categoryModifier)
	const hasAnyAction = Boolean(onEdit) || canTransfer || !hasNoGroups || canEditModifier

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
							{canLink && (
								<MenuItem
									onClick={() => {
										handleMenuClose()
										setDuplicateOpen(true)
									}}
								>
									<ListItemIcon>
										<CopyPlusIcon sx={{ fontSize: 18, color: '#9ca3af' }} />
									</ListItemIcon>
									{task.duplicateOf ? 'Изменить основную' : 'Отметить дублем'}
								</MenuItem>
							)}
							{canEditModifier && (
								<MenuItem
									onClick={() => {
										// Якорь берём у кнопки «Ещё»: сам MenuItem после закрытия
										// меню исчезнет, и поповер остался бы без якоря.
										const anchor = anchorEl
										handleMenuClose()
										setModifierAnchor(anchor)
									}}
								>
									<ListItemIcon>
										<ListTree sx={{ fontSize: 18, color: '#9ca3af' }} />
									</ListItemIcon>
									Изменить подкатегорию
								</MenuItem>
							)}
							{canTransfer && (
								<MenuItem
									onClick={() => {
										handleMenuClose()
										setTransferOpen(true)
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

			{task.access?.canPickModifier && task.categoryModifier && (
				<ModifierPopover
					task={task}
					onSelect={onSetModifier}
					hideTrigger
					open={Boolean(modifierAnchor)}
					anchorEl={modifierAnchor}
					onClose={() => setModifierAnchor(null)}
				/>
			)}

			<TransferModal open={transferOpen} onClose={() => setTransferOpen(false)} task={task} />
			<LinkDuplicateDialog open={duplicateOpen} task={task} onClose={() => setDuplicateOpen(false)} />
		</Box>
	)
}
