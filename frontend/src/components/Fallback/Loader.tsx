import type { FC } from 'react'
import {
	Box,
	CircularProgress,
	alpha,
	type CircularProgressProps,
	circularProgressClasses,
	Typography,
} from '@mui/material'

interface ILoaderProps extends Pick<CircularProgressProps, 'size' | 'thickness'> {
	text?: string
}

export const Loader: FC<ILoaderProps> = ({ size = 60, thickness = 5, text = 'Загрузка...' }) => {
	return (
		<Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', rowGap: 1 }}>
			<Box sx={{ position: 'relative' }}>
				<CircularProgress
					variant='determinate'
					sx={{
						color: 'grey.300',
					}}
					size={size}
					thickness={thickness}
					value={100}
				/>
				<CircularProgress
					variant='indeterminate'
					disableShrink
					sx={{
						color: theme => alpha(theme.palette.primary.main, 0.8),
						animationDuration: '2000ms',
						position: 'absolute',
						left: 0,
						[`& .${circularProgressClasses.circle}`]: {
							strokeLinecap: 'round',
						},
					}}
					size={size}
					thickness={thickness}
				/>
			</Box>
			<Typography color='text.secondary' variant='body2' component='span' children={text} />
		</Box>
	)
}
