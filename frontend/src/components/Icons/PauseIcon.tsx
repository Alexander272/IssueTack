import type { FC } from 'react'
import { SvgIcon, type SvgIconProps } from '@mui/material'

export const PauseIcon: FC<SvgIconProps> = props => {
	return (
		<SvgIcon {...props}>
			<svg x='0px' y='0px' viewBox='0 0 24 24' enableBackground='new 0 0 24 24' xmlSpace='preserve'>
				<path d='M6 19h4V5H6v14zm8-14v14h4V5h-4z' />
			</svg>
		</SvgIcon>
	)
}
