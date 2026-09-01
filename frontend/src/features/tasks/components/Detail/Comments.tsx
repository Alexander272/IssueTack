import { useRef, useState } from 'react'
import { useSelector } from 'react-redux'
import {
	Box,
	Button,
	CircularProgress,
	IconButton,
	Stack,
	Switch,
	TextField,
	Tooltip,
	Typography,
} from '@mui/material'
import { MessageSquare, Trash2, EyeOff, ArrowRightLeft, Paperclip, Download, ExternalLink } from 'lucide-mui'
import { toast } from 'react-toastify'

import { useGetActivityLogsQuery } from '../../modules/activity/activityApiSlice'
import { ActivityEntry } from '../../modules/activity/ActivityEntry'
import {
	useGetCommentsQuery,
	useCreateCommentMutation,
	useDeleteCommentMutation,
} from '../../modules/comments/commentsApiSlice'
import { useLazyGetAttachmentContentQuery } from '../../modules/attachments/attachmentsApiSlice'
import { formatSize } from '../../utils/size'
import { isImage, isPdf, isText, getFileIcon } from '../../utils/fileIcon'
import { PreviewDialog } from './Preview'
import { saveAs } from '@/utils/saveAs'
import type { IFetchError } from '@/app/types/error'
import { getUserId } from '@/features/user/userSlice'
import { getAvatarColor, getInitials, getDisplayName } from '@/utils/avatar'
import type { IComment } from '../../types/comment'
import type { IAttachment } from '../../types/task'

interface Props {
	taskId: string
	isInactive?: boolean
}

type TabKey = 'comments' | 'history'

const CommentAttachments = ({ attachments }: { attachments: IAttachment[] }) => {
	const [fetchContent] = useLazyGetAttachmentContentQuery()
	const [previewFile, setPreviewFile] = useState<IAttachment | null>(null)

	const handleDownload = async (file: IAttachment) => {
		try {
			const { url } = await fetchContent(file.id).unwrap()
			const res = await fetch(url)
			saveAs(await res.blob(), file.fileName)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка скачивания файла')
		}
	}

	const handleOpen = async (file: IAttachment) => {
		if (isImage(file.mimeType)) {
			setPreviewFile(file)
			return
		}
		if (isPdf(file.mimeType) || isText(file.mimeType)) {
			try {
				const { url } = await fetchContent(file.id).unwrap()
				window.open(url, '_blank')
			} catch (error) {
				const fetchError = error as IFetchError
				toast.error(fetchError.data?.message || 'Ошибка открытия файла')
			}
			return
		}
		handleDownload(file)
	}

	return (
		<>
			<Stack spacing={0.5} sx={{ mt: 1 }}>
				{attachments.map(file => {
					const info = getFileIcon(file.mimeType)
					const IconComponent = info.icon
					return (
						<Box
							key={file.id}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1,
								px: 1,
								py: 0.5,
								borderRadius: '6px',
								bgcolor: 'white',
								border: '1px solid #e5e7eb',
							}}
						>
							{isImage(file.mimeType) ? (
								<IconComponent sx={{ fontSize: 20, color: info.color }} />
							) : (
								<Box sx={{ width: 20, height: 20, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
									<IconComponent sx={{ fontSize: 20, color: info.color }} />
								</Box>
							)}
							<Typography
								sx={{
									flex: 1,
									minWidth: 0,
									fontSize: '0.75rem',
									fontWeight: 500,
									color: '#1f2937',
									overflow: 'hidden',
									textOverflow: 'ellipsis',
									whiteSpace: 'nowrap',
									cursor: 'pointer',
									'&:hover': { textDecoration: 'underline' },
								}}
								onClick={() => handleOpen(file)}
							>
								{file.fileName}
							</Typography>
							<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', flexShrink: 0 }}>
								{formatSize(file.fileSize)}
							</Typography>
							<IconButton size='small' onClick={() => handleOpen(file)} sx={{ p: 0.25, color: '#6b7280' }}>
								<ExternalLink sx={{ fontSize: 14 }} />
							</IconButton>
							<IconButton size='small' onClick={() => handleDownload(file)} sx={{ p: 0.25, color: '#6b7280' }}>
								<Download sx={{ fontSize: 14 }} />
							</IconButton>
						</Box>
					)
				})}
			</Stack>
			<PreviewDialog file={previewFile} onClose={() => setPreviewFile(null)} />
		</>
	)
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
	const color = getAvatarColor(comment.userId)
	const isAuthor = comment.userId === currentUserId
	const [withinDeleteWindow] = useState(() => Date.now() - new Date(comment.createdAt).getTime() < 15 * 60 * 1000)

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

export const Comments = ({ taskId, isInactive = false }: Props) => {
	const [activeTab, setActiveTab] = useState<TabKey>('comments')
	const [text, setText] = useState('')
	const [isInternal, setIsInternal] = useState(isInactive)
	const [files, setFiles] = useState<File[]>([])
	const fileInputRef = useRef<HTMLInputElement>(null)
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
			await createComment({
				ticketId: taskId,
				text: text.trim(),
				isInternal: isInactive || isInternal,
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
										<IconButton size='small' onClick={() => setFiles(f => f.filter((_, i) => i !== index))} sx={{ p: 0.25, color: '#9ca3af', '&:hover': { color: '#ef4444' } }}>
											<Trash2 sx={{ fontSize: 14 }} />
										</IconButton>
									</Box>
								))}
							</Stack>
						)}
						<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 1.5 }}>
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
								<EyeOff sx={{ fontSize: 16, color: isInactive || isInternal ? '#92400e' : '#9ca3af' }} />
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
									<Typography sx={{ fontSize: '0.75rem', color: isInternal ? '#92400e' : '#6b7280' }}>
										Внутренний
									</Typography>
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
		</Box>
	)
}
