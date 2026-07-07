import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { useState } from 'react'

export const Notifications = () => {
	const [checked, setChecked] = useState(true)

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ px: 2.5, py: 1.5, borderBottom: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
				<Typography sx={{ fontSize: '0.75rem', fontWeight: 700, color: '#374151', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
					Уведомления
				</Typography>
			</Box>
			<Box sx={{ p: 2.5 }}>
				<FormControlLabel
					control={
						<Checkbox
							checked={checked}
							onChange={e => setChecked(e.target.checked)}
							sx={{ '&.Mui-checked': { color: 'primary.main' } }}
						/>
					}
					label={<Typography sx={{ fontSize: '0.8125rem', color: '#374151' }}>Получать уведомления</Typography>}
				/>
				<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', mt: 0.5 }}>
					Вы будете получать уведомления о всех изменениях по этой заявке
				</Typography>
			</Box>
		</Box>
	)
}
