import type { FC } from 'react'
import { SvgIcon, type SvgIconProps } from '@mui/material'

export const EditBoxIcon: FC<SvgIconProps> = props => {
	return (
		<SvgIcon {...props}>
			<svg
				xmlns='http://www.w3.org/2000/svg'
				shapeRendering='geometricPrecision'
				textRendering='geometricPrecision'
				imageRendering='optimizeQuality'
				fillRule='evenodd'
				clipRule='evenodd'
				viewBox='0 0 512 505.97'
			>
				<path d='M307.49 321.86L177.68 348.3l18.78-138.12 111.03 111.68zM80.99 3.91H281.1l-71.54 78.26H84.49c-3.55 0-6.23 2.68-6.23 6.24v339.3h339.3c3.35 0 6.23-2.88 6.23-6.24V317.04l78.26-78.08v182.51c0 44.62-39.87 84.5-84.49 84.5H46.7c-25.67 0-46.7-24.6-46.7-50.2V88.41c0-44.43 36.57-84.5 80.99-84.5zM237.43 168.4L394.75 4.01c4.74-3.94 9.48-5.46 14.95-2.33l99.15 96.01c3.93 4.74 4.73 10.21-.81 15.75L348.39 280.16 237.43 168.4z' />
			</svg>
		</SvgIcon>
	)
}
