import { type FC, useState } from 'react'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { toast } from 'react-toastify'
import { X, Download, Maximize2, Minimize2 } from 'lucide-mui'

import { formatSize } from '../../utils/size'
import { saveAs } from '@/utils/saveAs'
import { useGetAttachmentContentQuery } from '../../modules/attachments/attachmentsApiSlice'
import type { IAttachment } from '../../types/task'

type ContentProps = {
	fileKey: string
	fileName: string
	fileSize: number
	fullScreen: boolean
	onToggleFullScreen: () => void
	onClose: () => void
}

function PreviewContent({ fileKey, fileName, fileSize, fullScreen, onToggleFullScreen, onClose }: ContentProps) {
	const { data } = useGetAttachmentContentQuery(fileKey)
	const src = data?.url ?? ''

	const handleDownload = async () => {
		if (!src) return
		try {
			const res = await fetch(src)
			if (!res.ok) {
				throw new Error('Ошибка при получении файла')
			}
			saveAs(await res.blob(), fileName)
		} catch {
			toast.error('Ошибка скачивания файла')
		}
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
					height: fullScreen ? '100vh' : 'auto',
				}}
			>
				<IconButton
					onClick={onToggleFullScreen}
					title={fullScreen ? 'Свернуть' : 'На весь экран'}
					sx={{
						position: 'absolute',
						top: 8,
						right: 56,
						color: 'white',
						zIndex: 1,
						bgcolor: 'rgba(0,0,0,0.4)',
						'&:hover': { bgcolor: 'rgba(0,0,0,0.6)' },
					}}
				>
					{fullScreen ? <Minimize2 sx={{ fontSize: 24 }} /> : <Maximize2 sx={{ fontSize: 24 }} />}
				</IconButton>
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
					<Box
						component='img'
						src={src}
						sx={{
							maxWidth: '100%',
							maxHeight: fullScreen ? '100vh' : '80vh',
							objectFit: 'contain',
						}}
					/>
				) : (
					<Typography color='white'>Загрузка...</Typography>
				)}
			</Box>
			{!fullScreen && (
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
			)}
		</>
	)
}

type Props = {
	file: IAttachment | null
	onClose: () => void
}

export const PreviewDialog: FC<Props> = ({ file, onClose }) => {
	const [fullScreen, setFullScreen] = useState(false)

	const handleClose = () => {
		setFullScreen(false)
		onClose()
	}

	return (
		<Dialog
			open={!!file}
			onClose={handleClose}
			maxWidth='lg'
			fullWidth
			fullScreen={fullScreen}
			slotProps={{
				paper: {
					sx: {
						bgcolor: 'transparent',
						boxShadow: 'none',
						...(fullScreen
							? { maxWidth: '100vw', maxHeight: '100vh', width: '100vw', height: '100vh' }
							: {}),
					},
				},
			}}
		>
			{file && (
				<PreviewContent
					key={file.id}
					fileKey={file.id}
					fileName={file.fileName}
					fileSize={file.fileSize}
					fullScreen={fullScreen}
					onToggleFullScreen={() => setFullScreen(prev => !prev)}
					onClose={handleClose}
				/>
			)}
		</Dialog>
	)
}
