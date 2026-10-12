import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Box, IconButton, Stack, Typography } from '@mui/material'
import { toast } from 'react-toastify'
import { CloudUploadIcon, Paperclip, XIcon } from 'lucide-mui'

// Максимальный размер файла — должно совпадать с UPLOAD_MAX_SIZE бэкенда (по умолчанию 10 МБ).
const MAX_FILE_SIZE = 10 * 1024 * 1024

type Props = {
	files?: File[]
	onChange?: (files: File[]) => void
	onUpload?: (files: File[]) => void
	// compact — плитка-карточка под размер карточки файла: используется в сетке
	// вложений заявки (Attachments), где рядом лежат уже загруженные файлы.
	compact?: boolean
}

const formatFileSize = (bytes: number) => {
	if (bytes < 1024) return `${bytes} Б`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`
	return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

// Тащат именно файлы с диска, а не текст/элемент страницы: на этапе dragenter
// dataTransfer.types ещё содержит "Files". Без этой проверки оверлей вылезал бы
// при перетаскивании выделенного текста.
const dragHasFiles = (e: DragEvent) => Array.from(e.dataTransfer?.types ?? []).includes('Files')

export const FileDropZone = ({ files = [], onChange, onUpload, compact = false }: Props) => {
	const inputRef = useRef<HTMLInputElement>(null)
	const [dragging, setDragging] = useState(false)

	const handleFiles = (list: FileList | null) => {
		if (!list || list.length === 0) return
		const all = Array.from(list)
		const valid = all.filter(f => f.size <= MAX_FILE_SIZE)
		const rejected = all.filter(f => f.size > MAX_FILE_SIZE)
		if (rejected.length > 0) {
			toast.error(
				`${rejected.length} файл(ов) превышают лимит ${formatFileSize(MAX_FILE_SIZE)} и не будут загружены`,
			)
		}
		if (valid.length === 0) return
		if (onUpload) {
			onUpload(valid)
		} else {
			onChange?.([...files, ...valid])
		}
	}

	// window-слушатели вешаются один раз, а замыкание должно видеть актуальные
	// files/onUpload/onChange — поэтому handleFiles держим в ref.
	const handleFilesRef = useRef(handleFiles)
	useEffect(() => {
		handleFilesRef.current = handleFiles
	})

	// Перетаскивание слушаем на window, а не на самой плашке: зона захвата — весь
	// экран. Счётчик глубины нужен потому, что dragenter/dragleave всплывают от
	// каждого дочернего элемента, пока курсор движется внутри окна.
	useEffect(() => {
		let depth = 0

		const onDragEnter = (e: DragEvent) => {
			if (!dragHasFiles(e)) return
			e.preventDefault()
			depth += 1
			setDragging(true)
		}

		const onDragOver = (e: DragEvent) => {
			if (!dragHasFiles(e)) return
			e.preventDefault() // без этого браузер не разрешит drop
		}

		const onDragLeave = (e: DragEvent) => {
			// relatedTarget === null — курсор покинул окно; счётчик тут не спасёт,
			// поэтому гасим оверлей сразу.
			if (e.relatedTarget == null) {
				depth = 0
				setDragging(false)
				return
			}
			depth = Math.max(0, depth - 1)
			if (depth === 0) setDragging(false)
		}

		const onDrop = (e: DragEvent) => {
			if (!dragHasFiles(e)) return
			e.preventDefault()
			depth = 0
			setDragging(false)
			handleFilesRef.current(e.dataTransfer?.files ?? null)
		}

		window.addEventListener('dragenter', onDragEnter)
		window.addEventListener('dragover', onDragOver)
		window.addEventListener('dragleave', onDragLeave)
		window.addEventListener('drop', onDrop)
		return () => {
			window.removeEventListener('dragenter', onDragEnter)
			window.removeEventListener('dragover', onDragOver)
			window.removeEventListener('dragleave', onDragLeave)
			window.removeEventListener('drop', onDrop)
		}
	}, [])

	return (
		<Box>
			{compact ? (
				<Box
					onClick={() => inputRef.current?.click()}
					sx={{
						height: '100%',
						minHeight: 120,
						border: '2px dashed #d1d5db',
						borderRadius: '8px',
						display: 'flex',
						flexDirection: 'column',
						alignItems: 'center',
						justifyContent: 'center',
						gap: 0.5,
						p: 1,
						textAlign: 'center',
						cursor: 'pointer',
						bgcolor: dragging ? '#eff6ff' : '#fff',
						transition: 'background-color 0.15s ease-in-out 0s, border-color 0.15s ease-in-out 0s',
						'&:hover': { borderColor: '#93c5fd', bgcolor: '#f9fafb' },
					}}
				>
					<CloudUploadIcon sx={{ fontSize: 28, color: '#9ca3af' }} />
					<Typography sx={{ fontSize: '0.75rem', color: '#374151' }}>
						<Typography component='span' sx={{ color: 'primary.main', fontWeight: 600 }}>
							Нажмите для загрузки
						</Typography>
					</Typography>
					<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af' }}>или перетащите файлы сюда</Typography>
				</Box>
			) : (
				<Box
					onClick={() => inputRef.current?.click()}
					sx={{
						border: '2px dashed #d1d5db',
						borderRadius: '12px',
						p: 4,
						textAlign: 'center',
						cursor: 'pointer',
						bgcolor: dragging ? '#eff6ff' : '#fff',
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
				</Box>
			)}

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

			{dragging &&
				createPortal(
					<Box
						sx={{
							position: 'fixed',
							inset: 0,
							zIndex: 2000,
							display: 'flex',
							flexDirection: 'column',
							alignItems: 'center',
							justifyContent: 'center',
							gap: 1,
							bgcolor: 'rgba(239, 246, 255, 0.95)',
							border: '3px dashed #3b82f6',
							// Оверлей декоративный: обработку drag/drop держат window-слушатели,
							// поэтому он не должен перехватывать события у элементов под ним.
							pointerEvents: 'none',
						}}
					>
						<CloudUploadIcon sx={{ fontSize: 72, color: '#3b82f6' }} />
						<Typography sx={{ fontSize: '1.25rem', fontWeight: 600, color: '#1d4ed8' }}>
							Отпустите файлы для загрузки
						</Typography>
						<Typography sx={{ fontSize: '0.875rem', color: '#3b82f6' }}>
							Скриншоты, логи, документы — до {formatFileSize(MAX_FILE_SIZE)}
						</Typography>
					</Box>,
					document.body,
				)}

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
