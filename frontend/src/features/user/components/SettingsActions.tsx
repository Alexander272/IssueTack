import { type FC } from 'react'
import { Box, Button } from '@mui/material'
import { SaveIcon } from 'lucide-mui'

interface SettingsActionsProps {
	dirty: boolean
	saving: boolean
	onReset: () => void
}

// Общие кнопки «Сохранить изменения»/«Отменить» для вкладок настроек уведомлений.
export const SettingsActions: FC<SettingsActionsProps> = ({ dirty, saving, onReset }) => {
	return (
		<Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mb: 3, flexWrap: 'wrap' }}>
			<Button
				type='submit'
				variant='outlined'
				disabled={!dirty || saving}
				sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500, background: '#fff' }}
				startIcon={<SaveIcon sx={{ fontSize: 18 }} />}
			>
				Сохранить изменения
			</Button>
			<Button
				variant='outlined'
				color='inherit'
				onClick={onReset}
				disabled={!dirty}
				sx={{ borderRadius: '8px', textTransform: 'none', fontWeight: 500, background: '#fff' }}
			>
				Отменить
			</Button>
		</Box>
	)
}