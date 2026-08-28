import type { ComponentType } from 'react'
import { useCallback, useState } from 'react'
import { Box, Typography, type SvgIconProps } from '@mui/material'
import { Image, FileText, Paperclip } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IAttachment } from '../../types/task'
import { formatSize } from '../../utils/size'
import { saveAs } from '@/utils/saveAs'
import {
	useGetAttachmentContentQuery,
	useLazyGetAttachmentContentQuery,
	useUploadAttachmentMutation,
} from '../../modules/attachments/attachmentsApiSlice'
import { FileDropZone } from '../../modules/attachments/FileDropZone'
import { PreviewDialog } from './Preview'

interface Props {
	attachments: IAttachment[] | undefined
	canWork?: boolean
	taskId?: string
}

const isImage = (m: string) => m.startsWith('image/')
const isPdf = (m: string) => m === 'application/pdf'
const isText = (m: string) => m.startsWith('text/')

const getFileIcon = (mimeType: string): { icon: ComponentType<SvgIconProps>; bg: string; color: string } => {
	if (isImage(mimeType)) return { icon: Image, bg: 'linear-gradient(135deg, #f3e8ff, #e9d5ff)', color: '#9333ea' }
	if (isPdf(mimeType)) return { icon: FileText, bg: 'linear-gradient(135deg, #fee2e2, #fecaca)', color: '#dc2626' }
	if (isText(mimeType)) return { icon: FileText, bg: 'linear-gradient(135deg, #f3f4f6, #e5e7eb)', color: '#6b7280' }
	return { icon: Paperclip, bg: 'linear-gradient(135deg, #dbeafe, #bfdbfe)', color: '#2563eb' }
}

function ImageThumbnail({ file }: { file: IAttachment }) {
	const { data } = useGetAttachmentContentQuery(file.id)
	const src = data?.url ?? ''

	if (!src) {
		const info = getFileIcon(file.mimeType)
		const IconComponent = info.icon
		return (
			<Box
				sx={{
					height: 80,
					background: info.bg,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
				}}
			>
				<IconComponent sx={{ fontSize: 28, color: info.color }} />
			</Box>
		)
	}

	return (
		<Box sx={{ height: 80, overflow: 'hidden', bgcolor: '#f3f4f6' }}>
			<Box
				component='img'
				src={src}
				alt={file.fileName}
				sx={{ width: '100%', height: '100%', objectFit: 'contain' }}
			/>
		</Box>
	)
}

export const Attachments = ({ attachments, canWork, taskId }: Props) => {
	const [uploadAttachment] = useUploadAttachmentMutation()
	const [fetchContent] = useLazyGetAttachmentContentQuery()
	const [previewFile, setPreviewFile] = useState<IAttachment | null>(null)

	const files = attachments ?? []
	const showUpload = canWork && taskId

	const handleDownload = useCallback(
		async (file: IAttachment) => {
			try {
				const { url } = await fetchContent(file.id).unwrap()
				const res = await fetch(url)
				saveAs(await res.blob(), file.fileName)
			} catch (error) {
				const fetchError = error as IFetchError
				toast.error(fetchError.data?.message || 'Ошибка скачивания файла')
			}
		},
		[fetchContent],
	)

	const handleOpen = useCallback(
		async (file: IAttachment) => {
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
		},
		[fetchContent, handleDownload],
	)

	if (files.length === 0 && !showUpload) return null

	const handleUpload = async (list: FileList) => {
		if (!taskId) return
		try {
			await Promise.all(
				Array.from(list).map(file =>
					uploadAttachment({ entityType: 'ticket', entityId: taskId, file }).unwrap(),
				),
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
						<Paperclip sx={{ fontSize: 16 }} />
						Вложения
						{files.length > 0 && (
							<Typography
								component='span'
								sx={{
									fontSize: '0.75rem',
									bgcolor: '#e5e7eb',
									color: '#374151',
									px: 1,
									py: 0.25,
									borderRadius: '999px',
								}}
							>
								{files.length}
							</Typography>
						)}
					</Typography>
				</Box>
			)}

			{files.length > 0 && (
				<Box
					sx={{
						display: 'grid',
						gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
						gap: 1.5,
						p: 2.5,
					}}
				>
					{files.map(file => {
						const info = getFileIcon(file.mimeType)
						const IconComponent = info.icon
						return (
							<Box
								key={file.id}
								component='button'
								onClick={() => handleOpen(file)}
								sx={{
									border: '1px solid #e5e7eb',
									borderRadius: '8px',
									overflow: 'hidden',
									cursor: 'pointer',
									transition: 'box-shadow 0.2s',
									textDecoration: 'none',
									bgcolor: 'white',
									p: 0,
									textAlign: 'left',
									width: '100%',
									'&:hover': { boxShadow: '0 4px 12px rgba(0,0,0,0.08)' },
								}}
							>
								{isImage(file.mimeType) ? (
									<ImageThumbnail file={file} />
								) : (
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
								)}
								<Box sx={{ p: 1.5 }}>
									<Typography
										sx={{
											fontSize: '0.75rem',
											fontWeight: 500,
											color: '#1f2937',
											overflow: 'hidden',
											textOverflow: 'ellipsis',
											whiteSpace: 'nowrap',
										}}
									>
										{file.fileName}
									</Typography>
									<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af' }}>
										{formatSize(file.fileSize)}
									</Typography>
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

			<PreviewDialog file={previewFile} onClose={() => setPreviewFile(null)} />
		</Box>
	)
}
