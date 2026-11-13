import { Box, Divider, Paper, Typography } from '@mui/material'
import { ChartsTooltipContainer, useAxesTooltip } from '@mui/x-charts'

export function WorkloadBarTooltip() {
	const axesData = useAxesTooltip()

	if (!axesData || axesData.length === 0) {
		return null
	}

	const currentAxis = axesData[0]

	const total = currentAxis.seriesItems.reduce((sum, item) => {
		const val = typeof item.value === 'number' ? item.value : 0
		return sum + val
	}, 0)

	return (
		<ChartsTooltipContainer trigger='axis'>
			<Paper sx={{ p: 1.5, minWidth: 210, boxShadow: 3 }}>
				{/* Заголовок из yAxis.valueFormatter */}
				<Typography variant='subtitle2' sx={{ fontWeight: 600, mb: 1 }}>
					{currentAxis.axisFormattedValue}
				</Typography>

				{/* Список всех серий */}
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
					{currentAxis.seriesItems.map(item => (
						<Box
							key={item.seriesId}
							sx={{
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'space-between',
								gap: 2,
							}}
						>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
								<Box
									sx={{
										width: 10,
										height: 10,
										borderRadius: '2px',
										bgcolor: item.color,
									}}
								/>
								<Typography variant='body2' color='text.secondary'>
									{item.formattedLabel}
								</Typography>
							</Box>
							<Typography variant='body2' sx={{ fontWeight: 500 }}>
								{item.formattedValue}
							</Typography>
						</Box>
					))}
				</Box>

				<Divider sx={{ my: 1 }} />

				{/* Итоговая строка «Всего» */}
				<Box
					sx={{
						display: 'flex',
						justifyContent: 'space-between',
						alignItems: 'center',
					}}
				>
					<Typography variant='body2' sx={{ fontWeight: 600 }}>
						Всего
					</Typography>
					<Typography variant='body2' sx={{ fontWeight: 700 }}>
						{total}
					</Typography>
				</Box>
			</Paper>
		</ChartsTooltipContainer>
	)
}
