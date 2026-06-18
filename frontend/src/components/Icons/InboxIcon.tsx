import { SvgIcon, type SvgIconProps } from '@mui/material'
import type { FC } from 'react'

export const InboxIcon: FC<SvgIconProps> = props => {
	return (
		<SvgIcon {...props}>
			<svg
				x='0px'
				y='0px'
				viewBox='0 0 122.883 104.293'
				enableBackground='new 0 0 122.883 104.293'
				xmlSpace='preserve'
			>
				<path
					fillRule='evenodd'
					clipRule='evenodd'
					d='M4.878,104.293h113.125c2.682,0,4.879-2.211,4.879-4.914l-0.115-38.84 c-0.006-2.645-0.387-4.012-1.338-6.492L102.379,4.38C101.516,2.132,100.408,0,97.998,0H25.729c-2.41,0-3.488,2.144-4.38,4.38 L1.22,54.865c-0.966,2.424-1.063,3.809-1.072,6.438L0,99.379C0,102.082,2.198,104.293,4.878,104.293L4.878,104.293z M107.496,57.518H82.111l-7.758,15.617H48.633l-7.862-15.617H15.195l17.922-41.943h57.184L107.496,57.518L107.496,57.518z'
				/>
			</svg>
		</SvgIcon>
	)
}
