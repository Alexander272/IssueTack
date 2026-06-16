import { SvgIcon, type SxProps, type Theme } from '@mui/material'
import type { FC } from 'react'

export const CheckIcon: FC<SxProps<Theme>> = style => {
	return (
		<SvgIcon sx={style}>
			<svg
				x='0px'
				y='0px'
				viewBox='0 0 122.881 89.842'
				enableBackground='new 0 0 122.881 89.842'
				xmlSpace='preserve'
			>
				<path d='M1.232,55.541c-1.533-1.388-1.652-3.756-0.265-5.289c1.388-1.534,3.756-1.652,5.29-0.265l34.053,30.878l76.099-79.699 c1.429-1.501,3.804-1.561,5.305-0.132c1.502,1.428,1.561,3.803,0.133,5.305L43.223,88.683l-0.005-0.005 c-1.396,1.468-3.716,1.563-5.227,0.196L1.232,55.541L1.232,55.541z' />
			</svg>
		</SvgIcon>
	)
}
