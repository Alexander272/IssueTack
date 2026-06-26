import type { FC } from 'react'
import { SvgIcon, type SvgIconProps } from '@mui/material'

export const PlayIcon: FC<SvgIconProps> = props => {
	return (
		<SvgIcon {...props}>
			<svg x='0px' y='0px' viewBox='0 0 24 24' enableBackground='new 0 0 24 24' xmlSpace='preserve'>
				<path d='M8 5v14l11-7z' />
			</svg>
		</SvgIcon>
	)
}
