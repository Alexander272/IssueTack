import { useState } from 'react'
import { Box, IconButton, Stack, Typography } from '@mui/material'
import { Download, ExternalLink } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IAttachment } from '../../../types/task'
import { useLazyGetAttachmentContentQuery } from '../../../modules/attachments/attachmentsApiSlice'
import { saveAs } from '@/utils/saveAs'
import { formatSize } from '../../../utils/size'
import { isImage, isPdf, isText, getFileIcon } from '../../../utils/fileIcon'
import { PreviewDialog } from '../Preview'

export const CommentAttachments = ({ attachments }: { attachments: IAttachment[] }) => {
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
								<Box
									sx={{
										width: 20,
										height: 20,
										display: 'flex',
										alignItems: 'center',
										justifyContent: 'center',
									}}
								>
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
							<IconButton
								size='small'
								onClick={() => handleOpen(file)}
								sx={{ p: 0.25, color: '#6b7280' }}
							>
								<ExternalLink sx={{ fontSize: 14 }} />
							</IconButton>
							<IconButton
								size='small'
								onClick={() => handleDownload(file)}
								sx={{ p: 0.25, color: '#6b7280' }}
							>
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
