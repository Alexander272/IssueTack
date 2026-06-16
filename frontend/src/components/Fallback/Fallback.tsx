import type { FC } from 'react'
import { Container, type SxProps, type Theme } from '@mui/material'

import { Loader } from './Loader'

export const Fallback: FC<SxProps<Theme>> = style => {
	return (
		<Container sx={{ height: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', ...style }}>
			<Loader />
		</Container>
	)
}
