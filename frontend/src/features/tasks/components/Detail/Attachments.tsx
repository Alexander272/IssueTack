import { Box, Dialog, IconButton, Typography, type SvgIconProps } from '@mui/material'
import { X, Download, Image, FileText, Paperclip } from 'lucide-mui'
import { toast } from 'react-toastify'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { ComponentType } from 'react'

import { API } from '@/app/api'
import { useAppSelector } from '@/hooks/redux'
import { getToken } from '@/features/user/userSlice'
import { getRealm } from '@/features/realms/realmSlice'
import type { IFetchError } from '@/app/types/error'
import type { IAttachment } from '../../types/task'
import { useUploadAttachmentMutation } from '../../modules/attachments/attachmentsApiSlice'
import { FileDropZone } from '../../modules/attachments/FileDropZone'

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

const formatSize = (bytes: number) => {
	if (bytes < 1024) return `${bytes} Б`
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`
	return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`
}

function useAuthFetch() {
	const token = useAppSelector(getToken)
	const realm = useAppSelector(getRealm)
	return useCallback(async (url: string) => {
		return fetch(url, {
			headers: {
				...(token ? { Authorization: `Bearer ${token}` } : {}),
				...(realm ? { realm: realm.id } : {}),
			},
		})
	}, [token, realm])
}

function ImageThumbnail({ file, authFetch }: { file: IAttachment; authFetch: (url: string) => Promise<Response> }) {
	const [src, setSrc] = useState('')
	const urlRef = useRef('')

	useEffect(() => {
		let active = true
		authFetch(API.attachments.content(file.id))
			.then(res => (res.ok ? res.blob() : null))
			.then(blob => {
				if (!active || !blob) return
				const url = URL.createObjectURL(blob)
				urlRef.current = url
				setSrc(url)
			})
		return () => {
			active = false
			if (urlRef.current) {
				URL.revokeObjectURL(urlRef.current)
				urlRef.current = ''
			}
		}
	}, [file.id, authFetch])

	if (!src) {
		const info = getFileIcon(file.mimeType)
		const IconComponent = info.icon
		return (
			<Box sx={{ height: 80, background: info.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
				<IconComponent sx={{ fontSize: 28, color: info.color }} />
			</Box>
		)
	}

	return (
		<Box sx={{ height: 80, overflow: 'hidden', bgcolor: '#f3f4f6' }}>
			<Box component='img' src={src} alt={file.fileName} sx={{ width: '100%', height: '100%', objectFit: 'cover' }} />
		</Box>
	)
}

function PreviewContent({ fileKey, fileName, fileSize, authFetch, onClose }: { fileKey: string; fileName: string; fileSize: number; authFetch: (url: string) => Promise<Response>; onClose: () => void }) {
	const [src, setSrc] = useState('')
	const urlRef = useRef('')

	useEffect(() => {
		let active = true
		authFetch(API.attachments.content(fileKey))
			.then(res => (res.ok ? res.blob() : null))
			.then(blob => {
				if (!active || !blob) return
				const url = URL.createObjectURL(blob)
				urlRef.current = url
				setSrc(url)
			})
		return () => { active = false }
	}, [fileKey, authFetch])

	useEffect(() => {
		return () => { if (urlRef.current) URL.revokeObjectURL(urlRef.current) }
	}, [])

	const handleDownload = () => {
		if (!src) return
		const a = document.createElement('a')
		a.href = src
		a.download = fileName
		a.click()
	}

	return (
		<>
			<Box sx={{ position: 'relative', bgcolor: '#000', display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 300 }}>
				<IconButton onClick={onClose} sx={{ position: 'absolute', top: 8, right: 8, color: 'white', zIndex: 1, bgcolor: 'rgba(0,0,0,0.4)', '&:hover': { bgcolor: 'rgba(0,0,0,0.6)' } }}>
					<X sx={{ fontSize: 24 }} />
				</IconButton>
				{src
					? <Box component='img' src={src} sx={{ maxWidth: '100%', maxHeight: '80vh', objectFit: 'contain' }} />
					: <Typography color='white'>Загрузка...</Typography>
				}
			</Box>
			<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2, py: 1.5, bgcolor: '#1f2937' }}>
				<Box>
					<Typography sx={{ color: 'white', fontWeight: 500, fontSize: '0.875rem' }}>{fileName}</Typography>
					<Typography sx={{ color: '#9ca3af', fontSize: '0.75rem' }}>{formatSize(fileSize)}</Typography>
				</Box>
				<IconButton onClick={handleDownload} sx={{ color: 'white', '&:hover': { bgcolor: 'rgba(255,255,255,0.1)' } }}>
					<Download sx={{ fontSize: 20 }} />
				</IconButton>
			</Box>
		</>
	)
}

function PreviewDialog({ file, authFetch, onClose }: { file: IAttachment | null; authFetch: (url: string) => Promise<Response>; onClose: () => void }) {
	return (
		<Dialog open={!!file} onClose={onClose} maxWidth='lg' fullWidth PaperProps={{ sx: { bgcolor: 'transparent', boxShadow: 'none' } }}>
			{file && <PreviewContent key={file.id} fileKey={file.id} fileName={file.fileName} fileSize={file.fileSize} authFetch={authFetch} onClose={onClose} />}
		</Dialog>
	)
}

export const Attachments = ({ attachments, canWork, taskId }: Props) => {
	const [uploadAttachment] = useUploadAttachmentMutation()
	const authFetch = useAuthFetch()
	const [previewFile, setPreviewFile] = useState<IAttachment | null>(null)

	const files = attachments ?? []
	const showUpload = canWork && taskId

	const handleDownload = useCallback(async (file: IAttachment) => {
		try {
			const res = await authFetch(API.attachments.content(file.id))
			if (!res.ok) {
				const body = await res.json().catch(() => ({}))
				throw body
			}
			const blob = await res.blob()
			const url = URL.createObjectURL(blob)
			const a = document.createElement('a')
			a.href = url
			a.download = file.fileName
			a.click()
			URL.revokeObjectURL(url)
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || fetchError.message || 'Ошибка скачивания файла')
		}
	}, [authFetch])

	const handleOpen = useCallback(async (file: IAttachment) => {
		if (isImage(file.mimeType)) {
			setPreviewFile(file)
			return
		}
		if (isPdf(file.mimeType) || isText(file.mimeType)) {
			try {
				const res = await authFetch(API.attachments.content(file.id))
				if (!res.ok) {
					const body = await res.json().catch(() => ({}))
					throw body
				}
				const blob = await res.blob()
				const url = URL.createObjectURL(blob)
				window.open(url, '_blank')
			} catch (error) {
				const fetchError = error as IFetchError
				toast.error(fetchError.data?.message || fetchError.message || 'Ошибка открытия файла')
			}
			return
		}
		handleDownload(file)
	}, [authFetch, handleDownload])

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
									<ImageThumbnail file={file} authFetch={authFetch} />
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

			<PreviewDialog file={previewFile} authFetch={authFetch} onClose={() => setPreviewFile(null)} />
		</Box>
	)
}
