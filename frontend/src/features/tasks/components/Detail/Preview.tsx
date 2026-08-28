import { useEffect, useMemo, type FC } from 'react'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { X, Download } from 'lucide-mui'

import { formatSize } from '../../utils/size'
import { saveAs } from '@/utils/saveAs'
import { useGetAttachmentContentQuery } from '../../modules/attachments/attachmentsApiSlice'
import type { IAttachment } from '../../types/task'

type ContentProps = {
	fileKey: string
	fileName: string
	fileSize: number
	onClose: () => void
}

function PreviewContent({ fileKey, fileName, fileSize, onClose }: ContentProps) {
	const { data: blob } = useGetAttachmentContentQuery(fileKey)
	const src = useMemo(() => (blob ? URL.createObjectURL(blob) : ''), [blob])

	useEffect(() => {
		return () => {
			if (src) URL.revokeObjectURL(src)
		}
	}, [src])

	const handleDownload = () => {
		if (!blob) return
		saveAs(blob, fileName)
	}

	return (
		<>
			<Box
				sx={{
					position: 'relative',
					bgcolor: '#000',
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					minHeight: 300,
				}}
			>
				<IconButton
					onClick={onClose}
					sx={{
						position: 'absolute',
						top: 8,
						right: 8,
						color: 'white',
						zIndex: 1,
						bgcolor: 'rgba(0,0,0,0.4)',
						'&:hover': { bgcolor: 'rgba(0,0,0,0.6)' },
					}}
				>
					<X sx={{ fontSize: 24 }} />
				</IconButton>
				{src ? (
					<Box component='img' src={src} sx={{ maxWidth: '100%', maxHeight: '80vh', objectFit: 'contain' }} />
				) : (
					<Typography color='white'>Загрузка...</Typography>
				)}
			</Box>
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					px: 2,
					py: 1.5,
					bgcolor: '#1f2937',
				}}
			>
				<Box>
					<Typography sx={{ color: 'white', fontWeight: 500, fontSize: '0.875rem' }}>{fileName}</Typography>
					<Typography sx={{ color: '#9ca3af', fontSize: '0.75rem' }}>{formatSize(fileSize)}</Typography>
				</Box>
				<IconButton
					onClick={handleDownload}
					sx={{ color: 'white', '&:hover': { bgcolor: 'rgba(255,255,255,0.1)' } }}
				>
					<Download sx={{ fontSize: 20 }} />
				</IconButton>
			</Box>
		</>
	)
}

type Props = {
	file: IAttachment | null
	onClose: () => void
}

export const PreviewDialog: FC<Props> = ({ file, onClose }) => {
	return (
		<Dialog
			open={!!file}
			onClose={onClose}
			maxWidth='lg'
			fullWidth
			slotProps={{
				paper: {
					sx: { bgcolor: 'transparent', boxShadow: 'none' },
				},
			}}
		>
			{file && (
				<PreviewContent
					key={file.id}
					fileKey={file.id}
					fileName={file.fileName}
					fileSize={file.fileSize}
					onClose={onClose}
				/>
			)}
		</Dialog>
	)
}
