import { type FC } from 'react'
import { Box, Divider, Stack, Typography, useTheme } from '@mui/material'
import { CircleCheckIcon, LightbulbIcon, TriangleAlertIcon } from 'lucide-mui'

export const HowItWorksCard: FC = () => {
	const { palette } = useTheme()
	return (
		<Box
			sx={{
				bgcolor: 'rgba(59,130,246,0.06)',
				border: '1px solid rgba(59,130,246,0.25)',
				borderRadius: '12px',
				p: 3,
				mb: 3,
			}}
		>
			<Box sx={{ display: 'flex', gap: 1.5 }}>
				<LightbulbIcon sx={{ fontSize: 22, color: '#3b82f6' }} />
				<Box sx={{ width: '100%' }}>
					<Typography variant='subtitle2' sx={{ fontWeight: 700 }}>
						Как работают уведомления
					</Typography>
					<Stack sx={{ mt: 2, gap: 1.5 }}>
						<Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
							<CircleCheckIcon sx={{ fontSize: 18, color: '#10b981' }} />
							<Typography variant='body2' color={palette.text.secondary}>
								Каждая <strong>категория привязана к группе</strong>. Включая уведомления по категории,
								вы подписываетесь на события по задачам этой категории в соответствующей группе.
							</Typography>
						</Box>
						<Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
							<CircleCheckIcon sx={{ fontSize: 18, color: '#10b981' }} />
							<Typography variant='body2' color={palette.text.secondary}>
								{/* <strong>Подписки по группам</strong> работают для всех задач в этой группе, независимо
								от категории.  */}
								Кнопки <strong>«Включить все» / «Выключить все»</strong> в заголовке группы позволяют
								быстро подписаться на все категории группы или отписаться от них.
							</Typography>
						</Box>
					</Stack>
					<Divider sx={{ my: 1.5 }} />
					<Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
						<TriangleAlertIcon sx={{ fontSize: 18, color: '#3b82f6' }} />
						<Typography variant='body2' color={palette.text.secondary}>
							<strong>Важно:</strong> задачи, назначенные лично вам, всегда приходят в уведомлениях,
							независимо от этих настроек.
						</Typography>
					</Box>
				</Box>
			</Box>
		</Box>
	)
}
