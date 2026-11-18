import { Box, Typography } from '@mui/material'

interface Props {
	done: number
	total: number
}

export const TaskProgressBar = ({ done, total }: Props) => {
	const isEmpty = total === 0
	const percent = isEmpty ? 0 : Math.min(100, Math.round((done / total) * 100))
	const isComplete = !isEmpty && done >= total

	if (isEmpty) {
		return (
			<Box
				sx={{
					display: 'inline-flex',
					alignItems: 'center',
					justifyContent: 'center',
					minWidth: 46,
					height: 24, // Совпадает по высоте с заполненным состоянием
					px: 1,
					bgcolor: '#f8fafc',
					border: '1px dashed #cbd5e1',
					borderRadius: '6px',
				}}
			>
				<Typography
					sx={{
						fontSize: '0.7rem',
						fontWeight: 500,
						color: '#94a3b8',
						lineHeight: 1,
					}}
				>
					—
				</Typography>
			</Box>
		)
	}

	return (
		<Box
			sx={{
				display: 'inline-flex',
				flexDirection: 'column',
				alignItems: 'center',
				justifyContent: 'center',
				minWidth: 46,
				height: 24,
				px: 1,
				bgcolor: isComplete ? '#ecfdf5' : '#f8fafc',
				border: '1px solid',
				borderColor: isComplete ? '#a7f3d0' : '#e2e8f0',
				borderRadius: '6px',
				transition: 'background-color 0.2s, border-color 0.2s',
			}}
		>
			<Typography
				sx={{
					fontSize: '0.7rem',
					fontWeight: 600,
					fontVariantNumeric: 'tabular-nums',
					color: isComplete ? '#047857' : '#334155',
					lineHeight: 1,
					mb: '4px',
				}}
			>
				{done}/{total}
			</Typography>
			<Box
				sx={{
					width: '100%',
					height: 3,
					bgcolor: '#cbd5e1',
					borderRadius: '999px',
					overflow: 'hidden',
				}}
			>
				<Box
					sx={{
						width: `${percent}%`,
						height: '100%',
						bgcolor: isComplete ? '#10b981' : '#3b82f6',
						transition: 'width 0.25s ease, background-color 0.2s',
					}}
				/>
			</Box>
		</Box>
	)
}
