import { type FC, type ReactNode } from 'react'
import { Box, Typography, useTheme } from '@mui/material'
import type { SvgIconProps } from '@mui/material'
import type { ComponentType } from 'react'

interface NotificationSectionCardProps {
	title: string
	subtitle: string
	icon: ComponentType<SvgIconProps>
	iconColor?: string
	actions?: ReactNode
	children: ReactNode
}

export const NotificationSectionCard: FC<NotificationSectionCardProps> = ({
	title,
	subtitle,
	icon: Icon,
	iconColor = '#3b82f6',
	actions,
	children,
}) => {
	const { palette } = useTheme()
	return (
		<Box
			sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden', mb: 3 }}
		>
			<Box sx={{ px: 3, py: 2, borderBottom: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'space-between',
						flexWrap: 'wrap',
						gap: 1,
					}}
				>
					<Box>
						<Typography
							variant='subtitle1'
							sx={{ fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}
						>
							<Icon sx={{ fontSize: 20, color: iconColor }} />
							{title}
						</Typography>
						<Typography variant='body2' color={palette.text.secondary} sx={{ mt: 0.5 }}>
							{subtitle}
						</Typography>
					</Box>
					{actions}
				</Box>
			</Box>
			{children}
		</Box>
	)
}
