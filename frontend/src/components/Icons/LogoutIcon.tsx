import type { FC } from 'react'
import { SvgIcon, type SxProps, type Theme } from '@mui/material'

export const LogoutIcon: FC<SxProps<Theme>> = style => {
	return (
		<SvgIcon sx={style}>
			<svg xmlSpace='preserve' viewBox='0 0 91.8 122.9'>
				<path d='M81.8 64.6H41.4a3.2 3.2 0 0 1-3.2-3.2c0-1.7 1.4-3.1 3.2-3.1h40.3L68.6 43.2c-1.2-1.4-1-3.4.2-4.5 1.3-1.2 3.3-1 4.5.3L91 59.3c1 1.3 1 3 0 4.3L73.2 83.9a3.1 3.1 0 0 1-4.5.3 3.2 3.2 0 0 1-.2-4.5l13.2-15zm-11 35a3.2 3.2 0 1 1 6.4 0V114a9 9 0 0 1-9 9H9a9 9 0 0 1-9-9V9a9 9 0 0 1 9-9h59.2a9 9 0 0 1 9 9v14.3a3.2 3.2 0 0 1-6.4 0V9c0-.7-.3-1.4-.7-1.9-.5-.4-1.2-.7-2-.7h-59c-.8 0-1.5.3-2 .7-.4.5-.7 1.2-.7 2v104.8c0 .7.3 1.4.7 1.8.5.5 1.2.8 2 .8h59c.8 0 1.5-.3 2-.8.4-.4.7-1.1.7-1.8V99.6z' />
			</svg>
		</SvgIcon>
	)
}
