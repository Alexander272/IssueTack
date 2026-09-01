import type { ComponentType } from 'react'
import type { SvgIconProps } from '@mui/material'
import { Image, FileText, Paperclip } from 'lucide-mui'

export const isImage = (m: string) => m.startsWith('image/')
export const isPdf = (m: string) => m === 'application/pdf'
export const isText = (m: string) => m.startsWith('text/')

export const getFileIcon = (mimeType: string): { icon: ComponentType<SvgIconProps>; bg: string; color: string } => {
	if (isImage(mimeType)) return { icon: Image, bg: 'linear-gradient(135deg, #f3e8ff, #e9d5ff)', color: '#9333ea' }
	if (isPdf(mimeType)) return { icon: FileText, bg: 'linear-gradient(135deg, #fee2e2, #fecaca)', color: '#dc2626' }
	if (isText(mimeType)) return { icon: FileText, bg: 'linear-gradient(135deg, #f3f4f6, #e5e7eb)', color: '#6b7280' }
	return { icon: Paperclip, bg: 'linear-gradient(135deg, #dbeafe, #bfdbfe)', color: '#2563eb' }
}