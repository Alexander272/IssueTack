import type { FC } from 'react'
import { Link as RouterLink } from 'react-router'
import { Box, IconButton, Link, Tooltip, Typography } from '@mui/material'
import { Link2Off, MessageSquare, Paperclip } from 'lucide-mui'

import type { IDuplicateRef } from '../../../types/task'
import dayjs from 'dayjs'

type Props = {
	duplicate: IDuplicateRef
	canUnlink: boolean
	unlinking: boolean
	onUnlink: (duplicate: IDuplicateRef) => void
}

export const DuplicateRow: FC<Props> = ({ duplicate, canUnlink, unlinking, onUnlink }) => {
	const number = duplicate.ticketNumber ?? '—'

	return (
		<Box
			className='duplicate-row'
			sx={{
				position: 'relative',
				display: 'flex',
				alignItems: 'flex-start',
				gap: 1,
				px: 2.5,
				py: 1.25,
				'& + .duplicate-row': { borderTop: '1px solid', borderColor: 'divider' },
				'&:hover, &:focus-within': { bgcolor: 'action.hover' },
				'&:hover .duplicate-unlink, &:focus-within .duplicate-unlink': { opacity: 1 },
			}}
		>
			<Box sx={{ minWidth: 0, flex: 1 }}>
				<Box sx={{ display: 'flex', alignItems: 'baseline', gap: 0.75 }}>
					<Link
						component={RouterLink}
						to={`/tasks/${duplicate.id}`}
						underline='hover'
						sx={{
							flex: 'none',
							color: 'primary.main',
							fontSize: 14,
							// Вся строка кликабельна, но кнопка отвязки лежит выше неё.
							'&::after': { content: '""', position: 'absolute', inset: 0 },
						}}
					>
						№{number}
					</Link>
					<Typography
						title={duplicate.title}
						sx={{
							fontSize: 14,
							color: 'text.primary',
							whiteSpace: 'nowrap',
							overflow: 'hidden',
							textOverflow: 'ellipsis',
						}}
					>
						{duplicate.title}
					</Typography>
				</Box>

				<Box
					sx={{
						display: 'flex',
						flexWrap: 'wrap',
						alignItems: 'center',
						gap: '2px 10px',
						mt: 0.4,
						fontSize: 12,
						color: 'text.secondary',
					}}
				>
					{duplicate.creatorName && (
						<Box component='span' sx={{ flexBasis: '100%' }}>
							{duplicate.creatorName}
						</Box>
					)}
					<Box component='span'>{dayjs(duplicate.createdAt).format('D MMM, HH:mm')}</Box>
					<Tooltip title='Комментарии'>
						<Box component='span' sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.4 }}>
							<MessageSquare sx={{ fontSize: 13 }} />
							{duplicate.commentsCount || '-'}
						</Box>
					</Tooltip>
					<Tooltip title='Вложения'>
						<Box component='span' sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.4 }}>
							<Paperclip sx={{ fontSize: 13 }} />
							{duplicate.attachmentsCount || '-'}
						</Box>
					</Tooltip>
				</Box>
			</Box>

			{canUnlink && (
				<Tooltip title='Отвязать дубль'>
					<IconButton
						className='duplicate-unlink'
						size='small'
						aria-label={`Отвязать дубль №${number}`}
						disabled={unlinking}
						onClick={() => onUnlink(duplicate)}
						sx={{
							position: 'relative',
							zIndex: 1,
							flex: 'none',
							width: 28,
							height: 28,
							opacity: 0,
							color: 'text.secondary',
							'&:hover': { bgcolor: 'background.paper', color: 'text.primary' },
							'&:focus-visible': { opacity: 1 },
							'@media (hover: none)': { opacity: 1 },
						}}
					>
						<Link2Off sx={{ fontSize: 16 }} />
					</IconButton>
				</Tooltip>
			)}
		</Box>
	)
}
