import type { FC, PropsWithChildren } from 'react'
import { type SxProps, type Theme, Typography, useTheme } from '@mui/material'

import { BreadLink } from './Link'

type Props = {
	to: string
	active?: boolean
	sx?: SxProps<Theme>
}

export const Breadcrumb: FC<PropsWithChildren<Props>> = ({ children, to, active, sx }) => {
	const { palette } = useTheme()

	if (active)
		return (
			<Typography sx={{ color: palette.primary.main, px: 1.5, py: 0.5, mx: -0.8, ...sx }}>{children}</Typography>
		)
	return (
		<BreadLink to={to} sx={{ ...sx }}>
			{children}
		</BreadLink>
	)
}
