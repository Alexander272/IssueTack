import { Box, Button, Stack, TextField, Typography } from '@mui/material'
import { Paperclip, ArrowRight, AtSign, MessageSquare } from 'lucide-mui'
import { useState } from 'react'

interface Comment {
	id: string
	author: string
	authorRole: string
	avatar: string
	text: string
	timestamp: string
	type: 'comment' | 'system' | 'question' | 'answer'
}

interface Props {
	comments: Comment[]
	taskId: string
}

export const Comments = ({ comments }: Props) => {
	const [text, setText] = useState('')

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
				<Typography sx={{ fontWeight: 700, color: '#1f2937', fontSize: '0.9375rem', display: 'flex', alignItems: 'center', gap: 1 }}>
					<MessageSquare sx={{ fontSize: 16 }} />
					Комментарии и история
					{comments.length > 0 && (
						<Typography component='span' sx={{ fontSize: '0.75rem', bgcolor: '#e5e7eb', color: '#374151', px: 1, py: 0.25, borderRadius: '999px' }}>
							{comments.length}
						</Typography>
					)}
				</Typography>

				<Box sx={{ display: 'flex', gap: 0.5 }}>
					{['Все', 'Комментарии', 'История'].map(filter => (
						<Button
							key={filter}
							size='small'
							sx={{
								textTransform: 'none',
								fontSize: '0.75rem',
								color: '#6b7280',
								minWidth: 0,
								px: 1.5,
								py: 0.25,
								borderRadius: '6px',
								'&:hover': { bgcolor: '#f3f4f6', color: 'primary.main' },
							}}
						>
							{filter}
						</Button>
					))}
				</Box>
			</Box>

			<Box sx={{ p: 2.5 }}>
				<Stack spacing={3}>
					{comments.map(comment => (
						<Box key={comment.id} sx={{ display: 'flex', gap: 1.5 }}>
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
									...(comment.type === 'system'
										? { bgcolor: '#dbeafe', color: '#2563eb', fontSize: '0.875rem' }
										: { bgcolor: comment.avatar }),
								}}
							>
								{comment.type === 'system' ? <ArrowRight sx={{ fontSize: 14 }} /> : comment.author.charAt(0)}
							</Box>

							<Box sx={{ flex: 1, minWidth: 0 }}>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
									<Typography sx={{ fontSize: '0.8125rem', fontWeight: 600, color: '#1f2937' }}>
										{comment.author}
									</Typography>
									{comment.authorRole && (
										<Typography
											sx={{
												fontSize: '0.6875rem',
												px: 1,
												py: 0.125,
												borderRadius: '4px',
												fontWeight: 500,
												...(comment.authorRole === 'Исполнитель'
													? { bgcolor: '#dbeafe', color: '#1e40af' }
													: comment.authorRole === 'Заявитель'
														? { bgcolor: '#fef3c7', color: '#92400e' }
														: { bgcolor: '#f3f4f6', color: '#374151' }),
											}}
										>
											{comment.authorRole}
										</Typography>
									)}
									<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', ml: 'auto', flexShrink: 0 }}>
										{comment.timestamp}
									</Typography>
								</Box>

								{comment.type === 'question' ? (
									<Box sx={{ bgcolor: '#fffbeb', borderLeft: '4px solid #f59e0b', borderRadius: '8px', p: 2, fontSize: '0.8125rem', color: '#374151' }}>
										<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5, color: '#d97706', fontWeight: 600, fontSize: '0.75rem' }}>
											Вопрос заявителю
										</Box>
										{comment.text}
									</Box>
								) : comment.type === 'system' ? (
									<Box sx={{ fontSize: '0.8125rem', color: '#6b7280' }}>
										{comment.text}
									</Box>
								) : (
									<Box sx={{ bgcolor: '#f9fafb', borderRadius: '8px', p: 2, fontSize: '0.8125rem', color: '#374151' }}>
										{comment.text}
										{comment.type === 'answer' && (
											<Box sx={{ mt: 1.5, display: 'inline-flex', alignItems: 'center', gap: 0.75, bgcolor: 'white', border: '1px solid #e5e7eb', borderRadius: '6px', px: 1.5, py: 0.75, fontSize: '0.75rem' }}>
												<Paperclip sx={{ fontSize: 14 }} />
												<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>printer_label.jpg</Typography>
											</Box>
										)}
									</Box>
								)}
							</Box>
						</Box>
					))}
				</Stack>
			</Box>

			<Box sx={{ borderTop: '1px solid #e5e7eb', p: 2.5, bgcolor: '#f9fafb' }}>
				<Box sx={{ display: 'flex', gap: 1.5 }}>
					<Box
						sx={{
							width: 36,
							height: 36,
							borderRadius: '50%',
							bgcolor: '#3b82f6',
							flexShrink: 0,
							display: 'flex',
							alignItems: 'center',
							justifyContent: 'center',
							fontSize: '0.75rem',
							fontWeight: 700,
							color: 'white',
						}}
					>
						Я
					</Box>
					<Box sx={{ flex: 1 }}>
						<TextField
							multiline
							rows={3}
							placeholder='Напишите комментарий...'
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
							<Box sx={{ display: 'flex', gap: 0.5 }}>
								<Button
									size='small'
									sx={{
										minWidth: 0,
										p: 1,
										borderRadius: '6px',
										color: '#9ca3af',
										'&:hover': { color: 'primary.main', bgcolor: 'white' },
									}}
								>
									<Paperclip sx={{ fontSize: 18 }} />
								</Button>
								<Button
									size='small'
									sx={{
										minWidth: 0,
										p: 1,
										borderRadius: '6px',
										color: '#9ca3af',
										'&:hover': { color: 'primary.main', bgcolor: 'white' },
									}}
								>
									<AtSign sx={{ fontSize: 18 }} />
								</Button>
							</Box>
							<Button
								variant='contained'
								size='small'
								disabled={!text.trim()}
								sx={{
									borderRadius: '8px',
									textTransform: 'none',
									fontSize: '0.8125rem',
									boxShadow: 'none',
									'&:hover': { boxShadow: 'none' },
								}}
							>
								Отправить
							</Button>
						</Box>
					</Box>
				</Box>
			</Box>
		</Box>
	)
}
