import type { DropPosition } from '../../utils/blocks'

export const ACCENT = '#3b4cca'
export const HIDDEN_COLOR = '#b45309'

export const thSx = {
	py: 2,
	px: 4,
	color: 'text.secondary',
	fontSize: '0.875rem',
	whiteSpace: 'nowrap',
} as const
export const rowCellSx = { py: 0.75, height: 46 } as const

export const statusLabel = (active: boolean) => (active ? 'Активна' : 'Неактивна')

export const buttonSx = {
	flexShrink: 0,
	height: 40,
	background: '#fff',
	borderColor: 'action.disabledBackground',
	textTransform: 'none',
	whiteSpace: 'nowrap',
	transition: 'all 0.3s ease-in-out',
}

// Прозрачные рамки всегда — высота строки не прыгает при появлении индикатора.
export const dropBordersSx = (position: DropPosition | null) => ({
	borderTop: '2px solid ',
	borderBottom: '2px solid',
	borderTopColor: position === 'before' ? ACCENT : 'transparent',
	borderBottomColor: position === 'after' ? ACCENT : 'transparent',
})
