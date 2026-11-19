import { useEffect, useState } from 'react'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { ArrowRightLeft, EyeOff, Trash2 } from 'lucide-mui'

import type { IComment } from '../../../types/comment'
import { getAvatarColor, getInitials, getDisplayName } from '@/utils/avatar'
import { CommentAttachments } from './CommentAttachments'

// Окно, в течение которого автор может удалить свой комментарий.
const DELETE_COMMENT_WINDOW_MS = 15 * 60 * 1000

interface Props {
	comment: IComment
	currentUserId: string
	onDelete: () => void
}

export const CommentEntry = ({ comment, currentUserId, onDelete }: Props) => {
	const displayName = getDisplayName(comment.user)
	const initials = getInitials(comment.user)
	const color = getAvatarColor(comment.userId ?? 'deleted')
	const isAuthor = comment.userId !== null && comment.userId === currentUserId
	const createdAt = new Date(comment.createdAt).getTime()
	// Реактивное окно удаления: по его истечении кнопка дизейблится, а не остаётся
	// замороженной на значении из момента монтирования.
	const [deleteWindowExpired, setDeleteWindowExpired] = useState(
		() => Date.now() - createdAt >= DELETE_COMMENT_WINDOW_MS,
	)
	useEffect(() => {
		if (deleteWindowExpired) return undefined
		const remaining = createdAt + DELETE_COMMENT_WINDOW_MS - Date.now()
		if (remaining <= 0) return undefined
		const timer = setTimeout(() => setDeleteWindowExpired(true), remaining)
		return () => clearTimeout(timer)
	}, [createdAt, deleteWindowExpired])
	const withinDeleteWindow = !deleteWindowExpired

	return (
		<Box sx={{ display: 'flex', gap: 1.5 }}>
			<Box
				sx={{
					width: 36,
					height: 36,
					borderRadius: '50%',
					flexShrink: 0,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					fontSize: '0.75rem',
					fontWeight: 700,
					color: 'white',
					bgcolor: color,
				}}
			>
				{initials}
			</Box>

			<Box sx={{ flex: 1, minWidth: 0 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.25 }}>
					<Typography sx={{ fontSize: '0.8125rem', fontWeight: 600, color: '#1f2937' }}>
						{displayName}
					</Typography>
					{comment.isInternal && (
						<Tooltip title='Внутренний комментарий — виден исполнителю и менеджерам группы'>
							<Box
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									gap: 0.25,
									px: 0.75,
									py: 0.125,
									borderRadius: '4px',
									fontSize: '0.6875rem',
									fontWeight: 500,
									bgcolor: '#fef3c7',
									color: '#92400e',
								}}
							>
								<EyeOff sx={{ fontSize: 12 }} />
								Внутренний
							</Box>
						</Tooltip>
					)}
					{comment.type === 'status_change' && (
						<Box
							sx={{
								display: 'inline-flex',
								alignItems: 'center',
								gap: 0.25,
								px: 0.75,
								py: 0.125,
								borderRadius: '4px',
								fontSize: '0.6875rem',
								fontWeight: 500,
								bgcolor: '#e0f2fe',
								color: '#0369a1',
							}}
						>
							<ArrowRightLeft sx={{ fontSize: 12 }} />
							Смена статуса
						</Box>
					)}
					<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', ml: 'auto', flexShrink: 0 }}>
						{new Date(comment.createdAt).toLocaleDateString('ru-RU', {
							day: 'numeric',
							month: 'short',
							hour: '2-digit',
							minute: '2-digit',
						})}
					</Typography>
					{isAuthor && (
						<Tooltip
							title={
								withinDeleteWindow
									? 'Удалить комментарий'
									: 'Удаление доступно в течение 15 минут после создания'
							}
						>
							<span>
								<IconButton
									size='small'
									onClick={onDelete}
									disabled={!withinDeleteWindow}
									sx={{
										ml: 0.5,
										color: '#9ca3af',
										'&:hover': { color: '#ef4444' },
										'&.Mui-disabled': { color: '#d1d5db' },
									}}
								>
									<Trash2 sx={{ fontSize: 14 }} />
								</IconButton>
							</span>
						</Tooltip>
					)}
				</Box>
				<Box
					sx={{
						bgcolor: comment.isInternal ? '#fffbeb' : '#f9fafb',
						borderRadius: '8px',
						p: 1.5,
						borderLeft: `3px solid ${color}`,
					}}
				>
					<Typography sx={{ fontSize: '0.8125rem', color: '#374151', whiteSpace: 'pre-wrap' }}>
						{comment.text}
					</Typography>
					{comment.attachments && comment.attachments.length > 0 && (
						<CommentAttachments attachments={comment.attachments} />
					)}
				</Box>
			</Box>
		</Box>
	)
}
