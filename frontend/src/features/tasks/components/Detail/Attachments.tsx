import { Box, Typography, type SvgIconProps } from '@mui/material'
import { Image, FileText, Paperclip } from 'lucide-mui'
import type { ComponentType } from 'react'

import type { IAttachment } from '../../types/task'

interface Props {
	attachments: IAttachment[] | undefined
}

const getFileIcon = (mimeType: string): { icon: ComponentType<SvgIconProps>; bg: string; color: string } => {
	if (mimeType.startsWith('image/')) return { icon: Image, bg: 'linear-gradient(135deg, #f3e8ff, #e9d5ff)', color: '#9333ea' }
	if (mimeType === 'application/pdf') return { icon: FileText, bg: 'linear-gradient(135deg, #fee2e2, #fecaca)', color: '#dc2626' }
	if (mimeType.startsWith('text/')) return { icon: FileText, bg: 'linear-gradient(135deg, #f3f4f6, #e5e7eb)', color: '#6b7280' }
	return { icon: Paperclip, bg: 'linear-gradient(135deg, #dbeafe, #bfdbfe)', color: '#2563eb' }
}

const formatSize = (bytes: number) => {
	if (bytes < 1024) return `${bytes} Б`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`
	return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

export const Attachments = ({ attachments }: Props) => {
	if (!attachments || attachments.length === 0) return null

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2.5, py: 2, borderBottom: '1px solid #e5e7eb' }}>
				<Typography sx={{ fontWeight: 700, color: '#1f2937', fontSize: '0.9375rem', display: 'flex', alignItems: 'center', gap: 1 }}>
					<Paperclip sx={{ fontSize: 16 }} />
					Вложения
					<Typography component='span' sx={{ fontSize: '0.75rem', bgcolor: '#e5e7eb', color: '#374151', px: 1, py: 0.25, borderRadius: '999px' }}>
						{attachments.length}
					</Typography>
				</Typography>
			</Box>

			<Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 1.5, p: 2.5 }}>
				{attachments.map(file => {
					const info = getFileIcon(file.mimeType)
					const IconComponent = info.icon
					return (
						<Box
							key={file.id}
							sx={{
								border: '1px solid #e5e7eb',
								borderRadius: '8px',
								overflow: 'hidden',
								cursor: 'pointer',
								transition: 'box-shadow 0.2s',
								'&:hover': { boxShadow: '0 4px 12px rgba(0,0,0,0.08)' },
							}}
						>
							<Box
								sx={{
									height: 80,
									background: info.bg,
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
									color: info.color,
								}}
							>
								<IconComponent sx={{ fontSize: 28, color: info.color }} />
							</Box>
							<Box sx={{ p: 1.5 }}>
								<Typography sx={{ fontSize: '0.75rem', fontWeight: 500, color: '#1f2937', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
									{file.fileName}
								</Typography>
								<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af' }}>{formatSize(file.fileSize)}</Typography>
							</Box>
						</Box>
					)
				})}
			</Box>
		</Box>
	)
}
