import { useState } from 'react'
import { useSelector } from 'react-redux'
import { Box, Button, CircularProgress, IconButton, Stack, Switch, TextField, Tooltip, Typography } from '@mui/material'
import { MessageSquare, Trash2, EyeOff } from 'lucide-mui'

import { useGetActivityLogsQuery } from '../../modules/activity/activityApiSlice'
import { ActivityEntry } from '../../modules/activity/ActivityEntry'
import {
	useGetCommentsQuery,
	useCreateCommentMutation,
	useDeleteCommentMutation,
} from '../../modules/comments/commentsApiSlice'
import { getUserId } from '@/features/user/userSlice'
import type { IComment } from '../../types/comment'

interface Props {
	taskId: string
}

type TabKey = 'comments' | 'history'

const AVATAR_COLORS = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#ef4444', '#6366f1', '#14b8a6']

const hashCode = (str: string): number => {
	let hash = 0
	for (let i = 0; i < str.length; i++) {
		hash = str.charCodeAt(i) + ((hash << 5) - hash)
	}
	return hash
}

const getAvatarColor = (name: string): string => {
	return AVATAR_COLORS[Math.abs(hashCode(name)) % AVATAR_COLORS.length]
}

const getDisplayName = (user?: { firstName?: string; lastName?: string; username?: string }): string => {
	if (user?.firstName || user?.lastName) {
		return `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim()
	}
	return user?.username ?? 'Пользователь'
}

const getInitials = (user?: { firstName?: string; lastName?: string; username?: string }): string => {
	if (user?.firstName || user?.lastName) {
		const first = user.firstName?.[0] ?? ''
		const last = user.lastName?.[0] ?? ''
		if (first || last) return `${first}${last}`.toUpperCase()
	}
	const name = user?.username ?? '?'
	return name.slice(0, 2).toUpperCase()
}

const CommentEntry = ({
	comment,
	currentUserId,
	onDelete,
}: {
	comment: IComment
	currentUserId: string
	onDelete: () => void
}) => {
	const displayName = getDisplayName(comment.user)
	const initials = getInitials(comment.user)
	const color = getAvatarColor(displayName)
	const isAuthor = comment.userId === currentUserId

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
						<Tooltip title='Внутренний комментарий — видите только вы'>
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
					<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', ml: 'auto', flexShrink: 0 }}>
						{new Date(comment.createdAt).toLocaleDateString('ru-RU', {
							day: 'numeric',
							month: 'short',
							hour: '2-digit',
							minute: '2-digit',
						})}
					</Typography>
					{isAuthor && (
						<IconButton
							size='small'
							onClick={onDelete}
							sx={{ ml: 0.5, color: '#9ca3af', '&:hover': { color: '#ef4444' } }}
						>
							<Trash2 sx={{ fontSize: 14 }} />
						</IconButton>
					)}
				</Box>
				<Box sx={{ bgcolor: comment.isInternal ? '#fffbeb' : '#f9fafb', borderRadius: '8px', p: 1.5 }}>
					<Typography sx={{ fontSize: '0.8125rem', color: '#374151', whiteSpace: 'pre-wrap' }}>
						{comment.text}
					</Typography>
				</Box>
			</Box>
		</Box>
	)
}

export const Comments = ({ taskId }: Props) => {
	const [activeTab, setActiveTab] = useState<TabKey>('comments')
	const [text, setText] = useState('')
	const [isInternal, setIsInternal] = useState(false)
	const currentUserId = useSelector(getUserId)

	const { data: commentsData, isLoading: commentsLoading } = useGetCommentsQuery(taskId, {
		skip: activeTab !== 'comments',
	})

	const { data: logsData, isLoading: logsLoading } = useGetActivityLogsQuery(taskId, {
		skip: activeTab !== 'history',
	})

	const [createComment, { isLoading: creating }] = useCreateCommentMutation()
	const [deleteComment] = useDeleteCommentMutation()

	const comments = commentsData?.data ?? []
	const logs = logsData?.data ?? []

	const handleSubmit = async () => {
		if (!text.trim()) return
		try {
			await createComment({ ticketId: taskId, text: text.trim(), isInternal }).unwrap()
			setText('')
			setIsInternal(false)
		} catch {
			// error handled by apiSlice
		}
	}

	const handleDelete = async (commentId: string) => {
		try {
			await deleteComment({ ticketId: taskId, commentId }).unwrap()
		} catch {
			// error handled by apiSlice
		}
	}

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					px: 2.5,
					py: 2,
					borderBottom: '1px solid #e5e7eb',
				}}
			>
				<Typography
					sx={{
						fontWeight: 700,
						color: '#1f2937',
						fontSize: '0.9375rem',
						display: 'flex',
						alignItems: 'center',
						gap: 1,
					}}
				>
					<MessageSquare sx={{ fontSize: 16 }} />
					{activeTab === 'comments' ? 'Комментарии' : 'История'}
				</Typography>

				<Box sx={{ display: 'flex', gap: 0.5 }}>
					{(
						[
							['comments', 'Комментарии'],
							['history', 'История'],
						] as const
					).map(([key, label]) => (
						<Button
							key={key}
							size='small'
							onClick={() => setActiveTab(key)}
							sx={{
								textTransform: 'none',
								fontSize: '0.75rem',
								minWidth: 0,
								px: 1.5,
								py: 0.25,
								borderRadius: '6px',
								color: activeTab === key ? 'primary.main' : '#6b7280',
								fontWeight: activeTab === key ? 600 : 400,
								bgcolor: activeTab === key ? '#eff6ff' : 'transparent',
								'&:hover': { bgcolor: activeTab === key ? '#eff6ff' : '#f3f4f6' },
							}}
						>
							{label}
						</Button>
					))}
				</Box>
			</Box>

			{activeTab === 'comments' ? (
				<Box sx={{ p: 2.5, maxHeight: 400, overflow: 'auto' }}>
					{commentsLoading ? (
						<Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
							<CircularProgress size={24} />
						</Box>
					) : comments.length === 0 ? (
						<Typography sx={{ color: '#9ca3af', fontSize: '0.875rem', textAlign: 'center', py: 3 }}>
							Комментариев пока нет
						</Typography>
					) : (
						<Stack spacing={2.5}>
							{comments.map(comment => (
								<CommentEntry
									key={comment.id}
									comment={comment}
									currentUserId={currentUserId ?? ''}
									onDelete={() => handleDelete(comment.id)}
								/>
							))}
						</Stack>
					)}
				</Box>
			) : (
				<Box sx={{ p: 2.5, maxHeight: 400, overflow: 'auto' }}>
					{logsLoading ? (
						<Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
							<CircularProgress size={24} />
						</Box>
					) : logs.length === 0 ? (
						<Typography sx={{ color: '#9ca3af', fontSize: '0.875rem', textAlign: 'center', py: 3 }}>
							История пуста
						</Typography>
					) : (
						<Stack spacing={2.5}>
							{logs.map(log => (
								<ActivityEntry key={log.id} log={log} />
							))}
						</Stack>
					)}
				</Box>
			)}

			<Box sx={{ borderTop: '1px solid #e5e7eb', p: 2.5, bgcolor: '#f9fafb' }}>
				<Box sx={{ display: 'flex', gap: 1.5 }}>
					<Box sx={{ flex: 1 }}>
						<TextField
							multiline
							rows={3}
							placeholder={isInternal ? 'Внутренний комментарий (видите только вы)...' : 'Напишите комментарий...'}
							value={text}
							onChange={e => setText(e.target.value)}
							fullWidth
							sx={{
								'& .MuiOutlinedInput-root': {
									borderRadius: '8px',
									fontSize: '0.8125rem',
									bgcolor: 'white',
								},
							}}
						/>
						<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 1.5 }}>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
								<EyeOff sx={{ fontSize: 16, color: isInternal ? '#92400e' : '#9ca3af' }} />
								<Switch
									size='small'
									checked={isInternal}
									onChange={e => setIsInternal(e.target.checked)}
									sx={{
										'& .MuiSwitch-switchBase.Mui-checked': { color: '#f59e0b' },
										'& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': { bgcolor: '#f59e0b' },
									}}
								/>
								<Typography sx={{ fontSize: '0.75rem', color: isInternal ? '#92400e' : '#6b7280' }}>
									Внутренний
								</Typography>
							</Box>
							<Button
								variant='contained'
								size='small'
								disabled={!text.trim() || creating}
								onClick={handleSubmit}
								sx={{
									borderRadius: '8px',
									textTransform: 'none',
									fontSize: '0.8125rem',
									boxShadow: 'none',
									'&:hover': { boxShadow: 'none' },
								}}
							>
								{creating ? <CircularProgress size={16} /> : 'Отправить'}
							</Button>
						</Box>
					</Box>
				</Box>
			</Box>
		</Box>
	)
}
