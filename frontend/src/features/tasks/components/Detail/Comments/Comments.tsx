import { useRef, useState } from 'react'
import { useSelector } from 'react-redux'
import { Box, Button, CircularProgress, IconButton, Stack, Switch, TextField, Tooltip, Typography } from '@mui/material'
import { EyeOff, MessageSquare, Paperclip, Trash2 } from 'lucide-mui'

import { useGetActivityLogsQuery } from '../../../modules/activity/activityApiSlice'
import { ActivityTimeline } from '../../../modules/activity/ActivityTimeline'
import {
	useGetCommentsQuery,
	useCreateCommentMutation,
	useDeleteCommentMutation,
} from '../../../modules/comments/commentsApiSlice'
import { getUserId } from '@/features/user/userSlice'
import { formatSize } from '../../../utils/size'
import { CommentEntry } from './CommentEntry'

interface Props {
	taskId: string
	/** Заявка в терминальном статусе: внешние комментарии запрещены, поле форсится внутренним. */
	isInactive: boolean
	/** Есть ли у пользователя «рабочий» доступ (write или исполнитель). От него зависит
	 *  возможность писать внутренние комментарии: заказчику (без групп) они недоступны. */
	canWork: boolean
}

type TabKey = 'comments' | 'history'

export const Comments = ({ taskId, isInactive, canWork }: Props) => {
	const [activeTab, setActiveTab] = useState<TabKey>('comments')
	const [text, setText] = useState('')
	const [isInternal, setIsInternal] = useState(isInactive)
	const [files, setFiles] = useState<File[]>([])
	const fileInputRef = useRef<HTMLInputElement>(null)
	const currentUserId = useSelector(getUserId)

	const { data: commentsData, isLoading: commentsLoading } = useGetCommentsQuery(taskId, {
		skip: activeTab !== 'comments',
	})

	const { data: logsData, isLoading: logsLoading } = useGetActivityLogsQuery({ parentId: taskId }, {
		skip: activeTab !== 'history',
	})

	const [createComment, { isLoading: creating }] = useCreateCommentMutation()
	const [deleteComment] = useDeleteCommentMutation()

	const comments = commentsData?.data ?? []
	const logs = logsData?.data ?? []

	const handleSubmit = async () => {
		if (!text.trim()) return
		try {
			await createComment({
				ticketId: taskId,
				text: text.trim(),
				isInternal: canWork && (isInactive || isInternal),
				files,
			}).unwrap()
			setText('')
			setIsInternal(isInactive)
			setFiles([])
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
				<Box sx={{ maxHeight: 400, overflow: 'auto' }}>
					{logsLoading ? (
						<Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
							<CircularProgress size={24} />
						</Box>
					) : (
						<ActivityTimeline logs={logs} />
					)}
				</Box>
			)}

			{activeTab === 'comments' && (canWork || !isInactive) && (
				<Box
					sx={{
						borderTop: `1px solid ${isInternal || isInactive ? '#fde68a' : '#e5e7eb'}`,
						p: 2.5,
						bgcolor: isInternal || isInactive ? '#fffbeb' : '#f9fafb',
						transition: 'background-color 0.15s, border-color 0.15s',
					}}
				>
					<Box sx={{ display: 'flex', gap: 1.5 }}>
						<Box sx={{ flex: 1 }}>
							<TextField
								multiline
								rows={3}
								placeholder={
									isInternal
										? 'Внутренний комментарий (виден исполнителю и менеджерам)...'
										: 'Напишите комментарий...'
								}
								value={text}
								onChange={e => setText(e.target.value)}
								fullWidth
								sx={{
									'& .MuiOutlinedInput-root': {
										borderRadius: '8px',
										fontSize: '0.8125rem',
										bgcolor: 'white',
										...((isInternal || isInactive) && {
											'& fieldset': { borderColor: '#fde68a' },
											'&:hover fieldset': { borderColor: '#fcd34d' },
										}),
									},
								}}
							/>
							{files.length > 0 && (
								<Stack spacing={0.5} sx={{ mt: 1 }}>
									{files.map((file, index) => (
										<Box
											key={`${file.name}-${index}`}
											sx={{
												display: 'flex',
												alignItems: 'center',
												gap: 1,
												px: 1,
												py: 0.5,
												bgcolor: 'white',
												border: '1px solid #e5e7eb',
												borderRadius: '6px',
											}}
										>
											<Paperclip sx={{ fontSize: 14, color: '#6b7280' }} />
											<Typography
												sx={{
													flex: 1,
													minWidth: 0,
													fontSize: '0.75rem',
													color: '#374151',
													overflow: 'hidden',
													textOverflow: 'ellipsis',
													whiteSpace: 'nowrap',
												}}
											>
												{file.name}
											</Typography>
											<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', flexShrink: 0 }}>
												{formatSize(file.size)}
											</Typography>
											<IconButton
												size='small'
												onClick={() => setFiles(f => f.filter((_, i) => i !== index))}
												sx={{ p: 0.25, color: '#9ca3af', '&:hover': { color: '#ef4444' } }}
											>
												<Trash2 sx={{ fontSize: 14 }} />
											</IconButton>
										</Box>
									))}
								</Stack>
							)}

							<Box
								sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 1.5 }}
							>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
									<input
										ref={fileInputRef}
										type='file'
										multiple
										hidden
										onChange={e => {
											const list = Array.from(e.target.files ?? [])
											if (list.length) setFiles(f => [...f, ...list])
											e.target.value = ''
										}}
									/>
									<Tooltip title='Прикрепить файлы'>
										<IconButton
											size='small'
											disabled={creating}
											onClick={() => fileInputRef.current?.click()}
											sx={{ color: '#6b7280', '&:hover': { color: '#2563eb' } }}
										>
											<Paperclip sx={{ fontSize: 18 }} />
										</IconButton>
									</Tooltip>
									{canWork && (
										<>
											{/* разделитель между «прикрепить» и «внутренний» */}
											<Box sx={{ width: '1px', height: 20, bgcolor: '#e5e7eb', mx: 0.5 }} />

											<Tooltip title='Внутренний комментарий — виден исполнителю и менеджерам заявки'>
												<span style={{ display: 'inline-flex' }}>
													<EyeOff
														sx={{
															fontSize: 16,
															color: isInactive || isInternal ? '#92400e' : '#9ca3af',
														}}
													/>
												</span>
											</Tooltip>
											{isInactive ? (
												<Typography sx={{ fontSize: '0.75rem', color: '#92400e' }}>
													Внутренний (заявка закрыта)
												</Typography>
											) : (
												<Switch
													size='small'
													checked={isInternal}
													onChange={e => setIsInternal(e.target.checked)}
													sx={{
														'& .MuiSwitch-switchBase.Mui-checked': { color: '#f59e0b' },
														'& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': {
															bgcolor: '#f59e0b',
														},
													}}
												/>
											)}
											{!isInactive && (
												<Typography
													sx={{
														fontSize: '0.75rem',
														color: isInternal ? '#92400e' : '#6b7280',
													}}
												>
													Внутренний
												</Typography>
											)}
										</>
									)}
								</Box>
								<Button
									variant='outlined'
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
			)}
		</Box>
	)
}
