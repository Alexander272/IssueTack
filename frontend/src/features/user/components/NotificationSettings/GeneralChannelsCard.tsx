import { type FC } from 'react'
import { Box, Typography, useTheme } from '@mui/material'
import { BellIcon, SettingsIcon } from 'lucide-mui'
import { useFormContext, useWatch } from 'react-hook-form'
import { Switch } from '@/components/Switch/Switch'
import type { INotificationSettings } from '../../notificationsApiSlice'

import { NotificationSectionCard } from './NotificationSectionCard'

export const GeneralChannelsCard: FC = () => {
	const { palette } = useTheme()
	const { setValue } = useFormContext<INotificationSettings>()
	const enabled = useWatch<INotificationSettings, 'enabled'>({ name: 'enabled' })

	const onToggle = (value: boolean) => {
		setValue('enabled', value, { shouldDirty: true })
	}

	return (
		<NotificationSectionCard
			title='Общие настройки каналов'
			subtitle='Единый переключатель — выключите, чтобы отключить все подписки ниже'
			icon={SettingsIcon}
		>
			<Box sx={{ p: 3 }}>
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						flexWrap: 'wrap',
						gap: 1,
					}}
				>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
						<BellIcon sx={{ fontSize: 22, color: enabled ? '#3b82f6' : palette.text.disabled }} />
						<Box>
							<Typography variant='body1' sx={{ fontWeight: 600 }}>
								Уведомления
							</Typography>
							<Typography variant='body2' color={palette.text.secondary}>
								Получать подписки по категориям и группам. Уведомления о задачах, назначенных лично вам,
								приходят всегда.
							</Typography>
						</Box>
					</Box>
					<Switch checked={enabled} onChange={(_, v) => onToggle(v)} />
				</Box>
			</Box>
		</NotificationSectionCard>
	)
}
