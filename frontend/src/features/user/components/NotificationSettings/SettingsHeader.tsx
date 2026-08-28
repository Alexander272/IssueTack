import { type FC } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'
import { SaveIcon } from 'lucide-mui'

interface SettingsHeaderProps {
	dirty: boolean
	saving: boolean
	onReset: () => void
}

export const SettingsHeader: FC<SettingsHeaderProps> = ({ dirty, saving, onReset }) => {
	const { palette } = useTheme()
	return (
		<Box
			sx={{
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				mb: 3,
				flexWrap: 'wrap',
				gap: 2,
			}}
		>
			<Box>
				<Typography variant='h4' sx={{ fontWeight: 700, color: 'text.primary' }}>
					Настройки уведомлений
				</Typography>
				<Typography variant='body2' color={palette.text.secondary} sx={{ mt: 0.5 }}>
					Настройте, по каким событиям и категориям вы хотите получать уведомления
				</Typography>
			</Box>
			<Box sx={{ display: 'flex', gap: 1 }}>
				<Button
					type='submit'
					variant='outlined'
					disabled={!dirty || saving}
					sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500 }}
					startIcon={<SaveIcon sx={{ fontSize: 18 }} />}
				>
					Сохранить изменения
				</Button>
				<Button
					variant='outlined'
					color='inherit'
					onClick={onReset}
					disabled={!dirty}
					sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500 }}
				>
					Отменить
				</Button>
			</Box>
		</Box>
	)
}
