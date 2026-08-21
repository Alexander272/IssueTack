import { useRef, useState } from 'react'
import { Box, IconButton, Stack, Typography } from '@mui/material'
import { CloudUploadIcon, Paperclip, XIcon } from 'lucide-mui'

type Props = {
	files?: File[]
	onChange?: (files: File[]) => void
	onUpload?: (list: FileList) => void
}

const formatFileSize = (bytes: number) => {
	if (bytes < 1024) return `${bytes} Б`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`
	return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

export const FileDropZone = ({ files = [], onChange, onUpload }: Props) => {
	const inputRef = useRef<HTMLInputElement>(null)
	const [dragOver, setDragOver] = useState(false)

	const handleFiles = (list: FileList | null) => {
		if (!list || list.length === 0) return
		if (onUpload) {
			onUpload(list)
		} else {
			onChange?.([...files, ...Array.from(list)])
		}
	}

	return (
		<Box>
			<Box
				onClick={() => inputRef.current?.click()}
				onDragOver={e => {
					e.preventDefault()
					setDragOver(true)
				}}
				onDragLeave={() => setDragOver(false)}
				onDrop={e => {
					e.preventDefault()
					setDragOver(false)
					handleFiles(e.dataTransfer.files)
				}}
				sx={{
					border: '2px dashed #d1d5db',
					borderRadius: '12px',
					p: 4,
					textAlign: 'center',
					cursor: 'pointer',
					bgcolor: dragOver ? '#eff6ff' : '#fff',
					transition: 'background-color 0.15s ease-in-out 0s',
				}}
			>
				<CloudUploadIcon sx={{ fontSize: 32, color: '#9ca3af', mb: 1 }} />
				<Typography sx={{ fontSize: '0.875rem', color: '#374151' }}>
					<Typography component='span' sx={{ color: 'primary.main', fontWeight: 600 }}>
						Нажмите для загрузки
					</Typography>{' '}
					или перетащите файлы сюда
				</Typography>
				<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', mt: 0.5 }}>
					Скриншоты, логи, документы
				</Typography>
				<input
					ref={inputRef}
					type='file'
					multiple
					hidden
					onChange={e => {
						handleFiles(e.target.files)
						e.target.value = ''
					}}
				/>
			</Box>

			{files.length > 0 && (
				<Stack sx={{ mt: 2, gap: 1 }}>
					{files.map((file, index) => (
						<Box
							key={`${file.name}-${index}`}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1.5,
								p: 1.5,
								bgcolor: '#f9fafb',
								borderRadius: '8px',
								border: '1px solid #e5e7eb',
							}}
						>
							<Box
								sx={{
									width: 32,
									height: 32,
									borderRadius: '8px',
									bgcolor: '#dbeafe',
									color: '#2563eb',
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
									flexShrink: 0,
								}}
							>
								<Paperclip sx={{ fontSize: 16 }} />
							</Box>
							<Box sx={{ flex: 1, minWidth: 0 }}>
								<Typography
									sx={{
										fontSize: '0.8125rem',
										fontWeight: 500,
										color: '#1f2937',
										overflow: 'hidden',
										textOverflow: 'ellipsis',
										whiteSpace: 'nowrap',
									}}
								>
									{file.name}
								</Typography>
								<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af' }}>
									{formatFileSize(file.size)}
								</Typography>
							</Box>
							<IconButton
								size='small'
								onClick={() => onChange?.(files.filter((_, i) => i !== index))}
								sx={{ color: '#9ca3af' }}
							>
								<XIcon sx={{ fontSize: 14 }} />
							</IconButton>
						</Box>
					))}
				</Stack>
			)}
		</Box>
	)
}
