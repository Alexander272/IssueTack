import { Box, Typography, type SvgIconProps } from '@mui/material'
import { Image, FileText, Paperclip } from 'lucide-mui'
import { toast } from 'react-toastify'
import type { ComponentType } from 'react'

import { API } from '@/app/api'
import type { IFetchError } from '@/app/types/error'
import type { IAttachment } from '../../types/task'
import { useUploadAttachmentMutation } from '../../modules/attachments/attachmentsApiSlice'
import { FileDropZone } from '../../modules/attachments/FileDropZone'

interface Props {
	attachments: IAttachment[] | undefined
	canWork?: boolean
	taskId?: string
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

export const Attachments = ({ attachments, canWork, taskId }: Props) => {
	const [uploadAttachment] = useUploadAttachmentMutation()

	const files = attachments ?? []
	const showUpload = canWork && taskId

	if (files.length === 0 && !showUpload) return null

	const handleUpload = async (list: FileList) => {
		if (!taskId) return
		try {
			await Promise.all(
				Array.from(list).map(file => uploadAttachment({ entityType: 'ticket', entityId: taskId, file }).unwrap())
			)
			if (list.length > 1) {
				toast.success(`Загружено файлов: ${list.length}`)
			}
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка загрузки файла', { autoClose: false })
		}
	}

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			{(files.length > 0 || showUpload) && (
				<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2.5, py: 2, borderBottom: '1px solid #e5e7eb' }}>
					<Typography sx={{ fontWeight: 700, color: '#1f2937', fontSize: '0.9375rem', display: 'flex', alignItems: 'center', gap: 1 }}>
						<Paperclip sx={{ fontSize: 16 }} />
						Вложения
						{files.length > 0 && (
							<Typography component='span' sx={{ fontSize: '0.75rem', bgcolor: '#e5e7eb', color: '#374151', px: 1, py: 0.25, borderRadius: '999px' }}>
								{files.length}
							</Typography>
						)}
					</Typography>
				</Box>
			)}

			{files.length > 0 && (
				<Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 1.5, p: 2.5 }}>
					{files.map(file => {
						const info = getFileIcon(file.mimeType)
						const IconComponent = info.icon
						return (
							<Box
								key={file.id}
								component='a'
								href={API.attachments.content(file.id)}
								target='_blank'
								rel='noopener noreferrer'
								sx={{
									border: '1px solid #e5e7eb',
									borderRadius: '8px',
									overflow: 'hidden',
									cursor: 'pointer',
									transition: 'box-shadow 0.2s',
									textDecoration: 'none',
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
			)}

			{showUpload && (
				<Box sx={{ p: 2.5 }}>
					<FileDropZone onUpload={handleUpload} />
				</Box>
			)}
		</Box>
	)
}
