import MuiButton from '@mui/material/Button'
import type { ButtonProps } from '@mui/material/Button'
import { styled } from '@mui/material/styles'

export const Button = styled(MuiButton)<ButtonProps>(() => ({
	minWidth: 36,
	width: 36,
	height: 36,
})) as typeof MuiButton
