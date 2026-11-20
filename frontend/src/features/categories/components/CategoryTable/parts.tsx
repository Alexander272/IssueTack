import { memo, type FC, type ReactNode } from 'react'
import { Box, Button, IconButton, Paper, Stack, TableCell, Tooltip, Typography } from '@mui/material'
import { ChevronRightIcon, EyeOffIcon, GripVerticalIcon, MoreHorizontalIcon } from 'lucide-mui'

import { getSmartDate } from '@/utils/date'
import { HIDDEN_COLOR } from './tableStyles'

export const CountBadge: FC<{ children: ReactNode }> = ({ children }) => (
	<Typography
		component='span'
		sx={{
			fontSize: '0.75rem',
			color: 'text.secondary',
			bgcolor: 'background.paper',
			border: '1px solid',
			borderColor: 'divider',
			borderRadius: '10px',
			px: 0.875,
			lineHeight: '18px',
			fontWeight: 400,
			whiteSpace: 'nowrap',
		}}
	>
		{children}
	</Typography>
)

export const HiddenMarker: FC<{ label: string }> = ({ label }) => (
	<Tooltip title={label}>
		<EyeOffIcon sx={{ fontSize: 14, color: HIDDEN_COLOR }} />
	</Tooltip>
)

export const NameCell: FC<{
	name: string
	description?: string | null
	weight?: number
	italic?: boolean
	adornment?: ReactNode
}> = ({ name, description, weight = 500, italic = false, adornment }) => (
	<Box sx={{ minWidth: 0 }}>
		<Stack direction='row' spacing={0.75} sx={{ alignItems: 'center' }}>
			<Typography sx={{ fontWeight: weight, fontStyle: italic ? 'italic' : 'normal' }}>{name}</Typography>
			{adornment}
		</Stack>
		{description && (
			<Typography
				sx={{
					fontSize: '0.78rem',
					color: 'text.secondary',
					maxWidth: 360,
					overflow: 'hidden',
					textOverflow: 'ellipsis',
					whiteSpace: 'nowrap',
				}}
				title={description}
			>
				{description}
			</Typography>
		)}
	</Box>
)

export const RowGrip: FC<{ enabled: boolean }> = ({ enabled }) => (
	<Box
		className='row-grip'
		sx={{
			display: 'flex',
			width: 16,
			color: 'text.disabled',
			opacity: 0,
			cursor: enabled ? 'grab' : 'default',
			visibility: enabled ? 'visible' : 'hidden',
		}}
		onClick={e => e.stopPropagation()}
	>
		<GripVerticalIcon sx={{ fontSize: 16 }} />
	</Box>
)

export const ExpandButton: FC<{ open: boolean; onToggle: () => void }> = ({ open, onToggle }) => (
	<IconButton
		size='small'
		onClick={e => {
			e.stopPropagation()
			onToggle()
		}}
		sx={{
			p: 0.5,
			color: 'text.secondary',
			transform: open ? 'rotate(90deg)' : 'none',
			transition: 'transform .15s',
		}}
	>
		<ChevronRightIcon sx={{ fontSize: 16 }} />
	</IconButton>
)

export const DueDaysCell: FC<{ days?: number | null }> = ({ days }) => (
	<TableCell sx={{ color: 'text.secondary' }}>{days != null ? `${days} дн.` : ''}</TableCell>
)

export const UpdatedAtCell: FC<{ value: string; fontWeight?: number }> = ({ value, fontWeight }) => (
	<TableCell
		sx={{ color: 'text.secondary', fontSize: '0.8rem', fontWeight }}
		title={new Date(value).toLocaleString('ru-RU')}
	>
		{getSmartDate(value)}
	</TableCell>
)

export const RowMenuCell: FC<{ onOpen: (anchor: HTMLElement) => void }> = ({ onOpen }) => (
	<TableCell align='right' sx={{ width: 58, pr: 1 }}>
		<IconButton
			onClick={e => {
				e.stopPropagation()
				onOpen(e.currentTarget)
			}}
		>
			<MoreHorizontalIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
		</IconButton>
	</TableCell>
)

export const OrderSaveBar = memo(function OrderSaveBar({
	saving,
	onSave,
	onReset,
}: {
	saving: boolean
	onSave: () => void
	onReset: () => void
}) {
	return (
		<Paper
			elevation={8}
			sx={{
				position: 'fixed',
				left: '50%',
				bottom: 20,
				transform: 'translateX(-50%)',
				bgcolor: '#171b22',
				color: '#f6f7f9',
				px: 2,
				py: 1.25,
				borderRadius: 3,
				display: 'flex',
				alignItems: 'center',
				gap: 1.75,
				zIndex: 1200,
			}}
		>
			<Typography sx={{ fontSize: '0.875rem' }}>Порядок изменён</Typography>
			<Button
				size='small'
				variant='outlined'
				onClick={onReset}
				disabled={saving}
				sx={{ color: '#f6f7f9', borderColor: 'rgba(128,128,128,.5)', textTransform: 'none' }}
			>
				Отменить
			</Button>
			<Button size='small' variant='contained' onClick={onSave} disabled={saving} sx={{ textTransform: 'none' }}>
				Сохранить порядок
			</Button>
		</Paper>
	)
})
