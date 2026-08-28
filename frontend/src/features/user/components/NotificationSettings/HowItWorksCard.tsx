import { type FC } from 'react'
import { Box, Divider, Typography, useTheme } from '@mui/material'
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
				<LightbulbIcon sx={{ fontSize: 22, color: '#3b82f6', mt: 0.3 }} />
				<Box sx={{ width: '100%' }}>
					<Typography variant='subtitle2' sx={{ fontWeight: 700 }}>
						Как работают уведомления
					</Typography>
					<Box sx={{ mt: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
						<Box sx={{ display: 'flex', gap: 1 }}>
							<CircleCheckIcon sx={{ fontSize: 18, color: '#10b981', mt: 0.3 }} />
							<Typography variant='body2' color={palette.text.secondary}>
								<strong>Подписки по категориям</strong> работают глобально — вы получаете уведомления
								по задачам этой категории во всех группах.
							</Typography>
						</Box>
						<Box sx={{ display: 'flex', gap: 1 }}>
							<CircleCheckIcon sx={{ fontSize: 18, color: '#10b981', mt: 0.3 }} />
							<Typography variant='body2' color={palette.text.secondary}>
								<strong>Подписки по группам</strong> работают для всех задач в этой группе, независимо от
								категории.
							</Typography>
						</Box>
					</Box>
					<Box
						sx={{
							mt: 1.5,
							bgcolor: 'white',
							borderRadius: '8px',
							border: '1px solid rgba(59,130,246,0.3)',
							p: 1.5,
						}}
					>
						<Box sx={{ display: 'flex', gap: 1 }}>
							<LightbulbIcon sx={{ fontSize: 18, color: '#f59e0b', mt: 0.3 }} />
							<Typography variant='body2'>
								<strong>Логика «ИЛИ»:</strong> если хотя бы одна подписка включена (в категории или в
								группе) — уведомление придёт. Если обе включены — уведомление придёт один раз (без
								дублирования).
							</Typography>
						</Box>
					</Box>
					<Divider sx={{ my: 1.5 }} />
					<Box sx={{ display: 'flex', gap: 1 }}>
						<TriangleAlertIcon sx={{ fontSize: 18, color: '#3b82f6', mt: 0.3 }} />
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
