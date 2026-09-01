import { type FC } from 'react'
import { Box, Button, type SxProps, type Theme } from '@mui/material'
import { CheckCheckIcon, XIcon } from 'lucide-mui'

type Props = {
	disabled: boolean
	onSetAll: (value: boolean) => void
	sx?: SxProps<Theme>
}

export const SelectButtons: FC<Props> = ({ disabled, onSetAll, sx }) => (
	<Box sx={{ display: 'flex', gap: 1, ...sx }}>
		<Button disabled={disabled} onClick={() => onSetAll(true)} sx={{ textTransform: 'none' }}>
			<CheckCheckIcon sx={{ fontSize: 18, color: 'primary', mr: 0.5 }} />
			Включить всё
		</Button>
		<Button color='inherit' disabled={disabled} onClick={() => onSetAll(false)} sx={{ textTransform: 'none' }}>
			<XIcon sx={{ fontSize: 18, color: 'inherit', mr: 0.5 }} />
			Выключить всё
		</Button>
	</Box>
)
